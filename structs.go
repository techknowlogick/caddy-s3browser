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
	Byetes uint64
	Name   string
	Date   time.Time
}
