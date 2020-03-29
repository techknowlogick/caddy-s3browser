package s3browser

import (
	"path/filepath"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
)

type Directory struct {
	Path    string
	Folders []Folder
	Files   []File
}

type Folder struct {
	Name string
}

type File struct {
	Folder string
	Bytes  int64
	Name   string
	Date   time.Time
}

type Config struct {
	Endpoint string
	Region   string
	Key      string
	Secret   string
	Secure   bool
	Bucket   string
	Refresh  time.Duration
	Debug    bool
}

type Node struct {
	Link         string
	ReadableName string
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

func (d Directory) ReadableName() string {
	return cleanPath(d.Path)
}

func (f Folder) ReadableName() string {
	return cleanPath(f.Name)
}

func cleanPath(s string) string {
	return filepath.Base(filepath.Dir(s))
}

func (d Directory) Breadcrumbs() []Node {
	link := "/"

	nodes := []Node{
		Node{Link: link, ReadableName: "/"},
	}

	for _, folder := range strings.Split(d.Path, "/") {
		if len(folder) == 0 {
			continue
		}

		link += folder + "/"
		nodes = append(nodes, Node{Link: link, ReadableName: folder})
	}

	return nodes
}
