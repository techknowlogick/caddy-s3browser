package s3browser

import (
	"path"
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
	Name  string
	Bytes int64
	Date  time.Time
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

func (d Directory) Name() string {
	return path.Base(d.Path)
}

func (f Folder) Url(parent Directory) string {
	return path.Join(parent.Path, f.Name)
}

func (d Directory) Breadcrumbs() []Node {
	nodes := []Node{ // TODO: remove need for Node
		Node{Link: "/", Name: "Home"}, // TODO: Home icon
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
