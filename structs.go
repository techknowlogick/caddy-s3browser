package s3browser

import (
	"time"
	"github.com/dustin/go-humanize"
)

type Directory struct {
	Path    string
	CanGoUp bool
	Folders []Folder
	Files   []File
}

type Folder struct {
	Name string
}

type File struct {
	Folder string
	Bytes int64
	Name  string
	Date  time.Time
}

type Config struct {
	Key      string
	Bucket   string
	Secret   string
	Endpoint string
}

// HumanSize returns the size of the file as a human-readable string
// in IEC format (i.e. power of 2 or base 1024).
func (fi File) HumanSize() string {
	return humanize.IBytes(uint64(fi.Bytes))
}

// HumanModTime returns the modified time of the file as a human-readable string.
func (fi File) HumanModTime(format string) string {
	return fi.Date.Format(format)
}