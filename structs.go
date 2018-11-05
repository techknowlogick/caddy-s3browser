package s3browser

import (
	"time"
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
