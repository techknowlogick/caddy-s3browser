package s3browser

import (
	"encoding/json"
	"html/template"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/caddyserver/caddy/caddyhttp/httpserver"
)

// Static check that Browse implements Caddy's Handler type
var _ httpserver.Handler = (*Browse)(nil)

type Browse struct {
	Next     httpserver.Handler
	Config   Config
	S3Cache  S3FsCache
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

	if dir, ok := b.S3Cache.GetDir(fullPath); ok {
		return b.serveDirectory(w, r, dir)
	}

	if _, ok := b.S3Cache.GetFile(fullPath); ok {
		if b.Config.SignedURLRedirect {
			return b.signedRedirect(w, r, normalizePath(fullPath))
		}
		if b.Config.SkipServing {
			return b.Next.ServeHTTP(w, r)
		}
		return b.serveFile(w, r, normalizePath(fullPath))
	}

	return b.Next.ServeHTTP(w, r)
}

func (b Browse) serveAPI(w http.ResponseWriter, r *http.Request) (int, error) {
	if b.Config.APISecret != "" {
		if _, pwd, ok := r.BasicAuth(); !ok || pwd != b.Config.APISecret {
			return http.StatusUnauthorized, nil
		}
	}

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

func (b Browse) renderJSON(w io.Writer, dir Directory) error {
	var data []byte
	var err error
	if !b.Config.Debug {
		data, err = json.Marshal(dir)
	} else {
		data, err = json.MarshalIndent(dir, "", "  ")
	}
	if err != nil {
		return err
	}

	_, err = w.Write(data)
	return err
}

func (b Browse) renderHTML(w io.Writer, dir Directory) error {
	return b.Template.Execute(w, TemplateArgs{
		SiteName:     b.Config.SiteName,
		Dir:          dir,
		SemanticSort: b.Config.SemanticSort,
	})
}

func (b Browse) signedRedirect(w http.ResponseWriter, r *http.Request, filePath string) (int, error) {
	client := NewS3Client(b.Config)
	url, err := client.s3.PresignedGetObject(b.Config.Bucket, filePath[1:], 10*time.Minute, nil)
	if err != nil {

	}
	http.Redirect(w, r, url.String(), http.StatusTemporaryRedirect)
	return http.StatusTemporaryRedirect, nil
}

func (b Browse) serveFile(w http.ResponseWriter, r *http.Request, filePath string) (int, error) {
	client := NewS3Client(b.Config)

	var rangeHdr string
	if val, ok := r.Header["Range"]; ok {
		rangeHdr = val[0]
	}

	reader, _, headers, err := client.GetObject(filePath, rangeHdr)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	w.Header().Set("Content-Type", headers.Get("Content-Type"))
	w.Header().Set("Content-Length", headers.Get("Content-Length"))
	if headers.Get("Content-Range") != "" {
		w.Header().Set("Content-Range", headers.Get("Content-Range"))
	}

	if _, err := io.Copy(w, reader); err != nil {
		return http.StatusInternalServerError, err
	}

	return http.StatusOK, nil
}
