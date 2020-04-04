package s3browser

import (
	"crypto/tls"
	"log"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/minio/minio-go/v6"
)

type S3FsCache struct {
	s3     *minio.Client
	logger *log.Logger
	bucket string
	data   map[string]Directory
}

type Directory struct {
	Path    string
	Folders []Folder
	Files   []File
}

type Folder struct {
	Name string
}

type File struct {
	Name  string
	Bytes int64
	Date  time.Time
}

type Node struct {
	Link string
	Name string
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

func (f File) Url(parent Directory) string {
	return path.Join(parent.Path, f.Name)
}

func (f Folder) Url(parent Directory) string {
	return path.Join(parent.Path, f.Name)
}

func (d Directory) Name() string {
	return path.Base(d.Path)
}

func (d Directory) Breadcrumbs() []Node {
	nodes := []Node{
		Node{Link: "/", Name: "Home"},
	}

	if d.Path == "/" {
		return nodes
	}

	currPath := ""
	for _, currName := range strings.Split(strings.Trim(d.Path, "/"), "/") {
		currPath += "/" + currName
		nodes = append(nodes, Node{Link: currPath, Name: currName})
	}

	return nodes
}

func NewS3Cache(cfg Config, l *log.Logger) (fs S3FsCache, err error) {
	fs.bucket = cfg.Bucket
	fs.logger = l

	if cfg.Region == "" {
		fs.s3, err = minio.New(cfg.Endpoint, cfg.Key, cfg.Secret, cfg.Secure)
	} else {
		fs.s3, err = minio.NewWithRegion(cfg.Endpoint, cfg.Key, cfg.Secret, cfg.Secure, cfg.Region)
	}
	if err != nil {
		return
	}

	if !cfg.Secure {
		fs.s3.SetCustomTransport(&http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		})
	}

	return fs, err
}

func (fs *S3FsCache) GetDir(dirPath string) (Directory, bool) {
	dir, ok := fs.data[normalizePath(dirPath)]
	return dir, ok
}

func (fs *S3FsCache) GetFile(filePath string) (File, bool) {
	dirPath, fileName := path.Split(filePath)
	dirPath = normalizePath(dirPath)

	if dir, ok := fs.data[dirPath]; ok {
		for _, file := range dir.Files {
			if file.Name == fileName {
				return file, true
			}
		}
	}

	return File{}, false
}

func (fs *S3FsCache) Refresh() (err error) {
	fs.logger.Println("Refreshing S3 cache")

	newData := map[string]Directory{
		"/": Directory{Path: "/"},
	}

	objectCh := fs.s3.ListObjectsV2(
		fs.bucket,
		"",   // prefix
		true, // recursive
		nil,  // doneChan
	)

	for obj := range objectCh {
		if obj.Err != nil {
			fs.logger.Printf("Err: %s", obj.Err)
			continue
		}

		objDir, objName := path.Split(obj.Key)
		objDir = normalizePath(objDir)

		// Add missing parent directories in `newData`
		if _, ok := newData[objDir]; !ok {
			// Split objDir into its path components
			dirs := strings.Split(objDir[1:], "/") // [1:]: skip leading /

			parentPath := "/"
			for _, curr := range dirs {
				currPath := path.Join(parentPath, curr)
				if _, ok := newData[currPath]; !ok {
					fs.logger.Printf("+  dir: %s\n", currPath)

					// Add to parent Node
					parentNode := newData[parentPath]
					parentNode.Folders = append(parentNode.Folders, Folder{Name: curr})
					newData[parentPath] = parentNode

					// Add own Node
					newData[currPath] = Directory{Path: currPath}
				}

				if parentPath != "/" {
					parentPath += "/"
				}
				parentPath += curr
			}
		}

		// Add the object
		if objName != "" { // "": obj is the directory itself
			fs.logger.Printf("+ file: %s/%s\n", objDir, objName)

			fsCopy := newData[objDir]
			fsCopy.Files = append(fsCopy.Files, File{
				Name:  objName,
				Bytes: obj.Size,
				Date:  obj.LastModified,
			})
			newData[objDir] = fsCopy
		}
	}

	fs.data = newData

	fs.logger.Println("S3 cache updated")
	return nil
}

// Ensure path starts with / and doesn't end with one
func normalizePath(path string) string {
	return "/" + strings.Trim(path, "/")
}
