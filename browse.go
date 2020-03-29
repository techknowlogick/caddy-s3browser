package s3browser

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strings"

	"github.com/caddyserver/caddy/caddyhttp/httpserver"
)

type Browse struct {
	Next     httpserver.Handler
	Config   Config
	Fs       map[string]Directory
	Template *template.Template
}

func (b Browse) ServeHTTP(w http.ResponseWriter, r *http.Request) (int, error) {
	fullPath := r.URL.Path
	if fullPath == "" {
		fullPath = "/"
	}

	switch r.Method {
	case http.MethodGet, http.MethodHead:
		// proceed, noop
	case "PROPFIND", http.MethodOptions:
		return http.StatusNotImplemented, nil
	default:
		return b.Next.ServeHTTP(w, r)
	}

	if dir, ok := b.Fs[fullPath]; ok {
		return b.serveDirectory(w, r, dir)
	}

	return b.Next.ServeHTTP(w, r)
}

func (b Browse) serveDirectory(w http.ResponseWriter, r *http.Request, dir Directory) (int, error) {
	var buf *bytes.Buffer
	var err error
	acceptHeader := strings.ToLower(strings.Join(r.Header["Accept"], ","))
	switch {
	case strings.Contains(acceptHeader, "application/json"):
		if buf, err = b.formatAsJSON(dir); err != nil {
			return http.StatusInternalServerError, err
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
	default:
		if buf, err = b.formatAsHTML(dir); err != nil {
			fmt.Println(err)
			return http.StatusInternalServerError, err
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
	}
	buf.WriteTo(w)

	return http.StatusOK, nil
}

func (b Browse) formatAsJSON(listing Directory) (*bytes.Buffer, error) {
	marsh, err := json.Marshal(listing)
	if err != nil {
		return nil, err
	}

	buf := new(bytes.Buffer)
	_, err = buf.Write(marsh)
	return buf, err
}

func (b Browse) formatAsHTML(listing Directory) (*bytes.Buffer, error) {
	buf := new(bytes.Buffer)
	err := b.Template.Execute(buf, listing)
	return buf, err
}
