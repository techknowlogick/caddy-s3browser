package s3browser

import (
	"path"
	"strings"
	"sync"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/minio/minio-go/v6"
	"go.uber.org/zap"
)

type S3FsCache struct {
	lock   sync.RWMutex
	s3     S3Client
	logger *zap.Logger
	bucket string
	data   map[string]Directory
}

type Directory struct {
	Path    string
	Folders []string
	Files   map[string]File
}

type File struct {
	Bytes int64
	Date  time.Time
}

// HumanSize returns the size of the file as a human-readable string
// in IEC format (i.e. power of 2 or base 1024).
func (f File) HumanSize() string {
	return humanize.IBytes(uint64(f.Bytes))
}

// HumanModTime returns the modified time of the file as a human-readable string.
func (f File) HumanModTime(format string) string {
	return f.Date.Format(format)
}

func NewS3Cache(client S3Client, l *zap.Logger) (fs S3FsCache, err error) {
	return S3FsCache{
		s3:     client,
		logger: l,
	}, nil
}

func (fs *S3FsCache) GetDir(dirPath string) (Directory, bool) {
	dir, ok := fs.data[normalizePath(dirPath)]
	return dir, ok
}

func (fs *S3FsCache) GetFile(filePath string) (File, bool) {
	dirPath, fileName := path.Split(filePath)
	dirPath = normalizePath(dirPath)

	dir, ok := fs.GetDir(dirPath)
	if !ok {
		return File{}, false
	}

	file, ok := dir.Files[fileName]
	return file, ok
}

func (fs *S3FsCache) Refresh() (err error) {
	fs.logger.Info("Refreshing S3 cache")

	newData := map[string]Directory{}
	addDirectory(fs.logger, newData, "/")

	fs.s3.ForEachObject(func(obj minio.ObjectInfo) {
		objDir, objName := path.Split(obj.Key)
		objDir = normalizePath(objDir)

		// Add any missing parent directories in `newData`
		if _, ok := newData[objDir]; !ok {
			addDirectory(fs.logger, newData, objDir)
		}

		// Add the object
		if objName != "" { // "": obj is the directory itself
			fs.logger.Debug("file", zap.String("dir", objDir), zap.String("name", objName))

			fsCopy := newData[objDir]
			fsCopy.Files[objName] = File{
				Bytes: obj.Size,
				Date:  obj.LastModified,
			}
			newData[objDir] = fsCopy
		}
	})

	fs.data = newData

	fs.logger.Info("S3 cache updated")
	return nil
}

// Ensure path starts with / and doesn't end with one
func normalizePath(p string) string {
	if p == "" {
		return "/"
	}
	return "/" + strings.Trim(path.Clean(p), "/")
}

// Add directory
// `dirPath` must be normalized
func addDirectory(logger *zap.Logger, outData map[string]Directory, dirPath string) {
	// Split dirPath into its path components
	dirs := strings.Split(dirPath[1:], "/") // [1:]: skip leading /

	parentPath := "/"
	for _, curr := range dirs {
		currPath := path.Join(parentPath, curr)
		if _, ok := outData[currPath]; !ok {
			logger.Debug("dir", zap.String("path", currPath))

			// Add to parent Node
			parentNode := outData[parentPath]
			parentNode.Folders = append(parentNode.Folders, curr)
			outData[parentPath] = parentNode

			// Add own Node
			outData[currPath] = Directory{
				Path:    currPath,
				Folders: []string{},
				Files:   map[string]File{},
			}
		}

		if parentPath != "/" {
			parentPath += "/"
		}
		parentPath += curr
	}
}
