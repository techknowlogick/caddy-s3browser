package s3browser

import (
	"encoding/json"
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
	Refresh  chan struct{}
}

func (b Browse) ServeHTTP(w http.ResponseWriter, r *http.Request) (int, error) {
	fullPath := r.URL.Path
	if fullPath == "" {
		fullPath = "/"
	}

	switch r.Method {
	case http.MethodGet, http.MethodHead:
		// proceed, noop
	case http.MethodPost:
		return b.serveAPI(w, r)
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

func (b Browse) serveAPI(w http.ResponseWriter, r *http.Request) (int, error) {
	// trigger refresh
	b.Refresh <- struct{}{}
	return http.StatusOK, nil
}

func (b Browse) serveDirectory(w http.ResponseWriter, r *http.Request, dir Directory) (int, error) {
	renderFunc := b.renderHTML
	contentType := "text/html"

	acceptHeader := strings.ToLower(strings.Join(r.Header["Accept"], ","))
	if strings.Contains(acceptHeader, "application/json") {
		renderFunc = b.renderJSON
		contentType = "application/json"
	}

	w.Header().Set("Content-Type", contentType+"; charset=utf-8")

	if err := renderFunc(w, dir); err != nil {
		return http.StatusInternalServerError, err
	}

	return http.StatusOK, nil
}

func (b Browse) renderJSON(w http.ResponseWriter, listing Directory) error {
	marsh, err := json.Marshal(listing)
	if err != nil {
		return err
	}

	_, err = w.Write(marsh)
	return err
}

func (b Browse) renderHTML(w http.ResponseWriter, listing Directory) error {
	return b.Template.Execute(w, listing)
}
