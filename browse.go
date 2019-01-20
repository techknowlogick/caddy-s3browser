package s3browser

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"

	"github.com/mholt/caddy/caddyhttp/httpserver"
	"github.com/minio/minio-go"
)

type Browse struct {
	Next     httpserver.Handler
	Config   Config
	Client   *minio.Client
	Fs       map[string]Directory
	Template *template.Template
}

func (b Browse) ServeHTTP(w http.ResponseWriter, r *http.Request) (int, error) {

	path := r.URL.Path
	if path == "" {
		path = "/"
	}
	if _, ok := b.Fs[path]; !ok {
		return b.Next.ServeHTTP(w, r)
	}
	switch r.Method {
	case http.MethodGet, http.MethodHead:
		// proceed, noop
	case "PROPFIND", http.MethodOptions:
		return http.StatusNotImplemented, nil
	default:
		return b.Next.ServeHTTP(w, r)
	}

	var buf *bytes.Buffer
	var err error
	if buf, err = b.formatAsHTML(b.Fs[path]); err != nil {
		fmt.Println(err)
		return http.StatusInternalServerError, err
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	buf.WriteTo(w)

	return http.StatusOK, nil
}

func (b Browse) formatAsHTML(listing Directory) (*bytes.Buffer, error) {
	buf := new(bytes.Buffer)
	err := b.Template.Execute(buf, listing)
	return buf, err
}
