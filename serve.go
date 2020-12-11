package s3browser

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
)

func (b S3Browser) ServeHTTP(w http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {
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
		return caddyhttp.Error(http.StatusNotImplemented, nil)
	default:
		return next.ServeHTTP(w, r)
	}

	if dir, ok := b.s3Cache.GetDir(fullPath); ok {
		return b.serveDirectory(w, r, dir)
	}

	if _, ok := b.s3Cache.GetFile(fullPath); ok {
		if b.SignedURLRedirect {
			return b.signedRedirect(w, r, normalizePath(fullPath))
		}
		return b.serveFile(w, r, normalizePath(fullPath))
	}

	return next.ServeHTTP(w, r)
}

func (b *S3Browser) serveAPI(w http.ResponseWriter, r *http.Request) error {
	if b.RefreshAPISecret != "" {
		if _, pwd, ok := r.BasicAuth(); !ok || pwd != b.RefreshAPISecret {
			return caddyhttp.Error(http.StatusUnauthorized, nil)
		}
	}

	b.refreshTrigger <- struct{}{}
	return nil
}

func (b *S3Browser) serveDirectory(w http.ResponseWriter, r *http.Request, dir Directory) error {
	renderFunc := b.renderHTML
	contentType := "text/html"

	acceptHeader := strings.ToLower(strings.Join(r.Header["Accept"], ","))
	if strings.Contains(acceptHeader, "application/json") {
		renderFunc = b.renderJSON
		contentType = "application/json"
	}

	w.Header().Set("Content-Type", contentType+"; charset=utf-8")

	return renderFunc(w, dir)
}

func (b *S3Browser) renderJSON(w io.Writer, dir Directory) error {
	var data []byte
	var err error
	if !b.Debug {
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

func (b *S3Browser) renderHTML(w io.Writer, dir Directory) error {
	return b.template.Execute(w, TemplateArgs{
		SiteName: b.SiteName,
		Dir:      dir,
	})
}

func (b *S3Browser) signedRedirect(w http.ResponseWriter, r *http.Request, filePath string) error {
	client := b.newS3Client()
	url, err := client.s3.PresignedGetObject(b.Bucket, filePath[1:], 10*time.Minute, nil)
	if err == nil {
		return err
	}
	http.Redirect(w, r, url.String(), http.StatusTemporaryRedirect)
	return nil
}

func (b *S3Browser) serveFile(w http.ResponseWriter, r *http.Request, filePath string) error {
	client := b.newS3Client()

	var rangeHdr string
	if val, ok := r.Header["Range"]; ok {
		rangeHdr = val[0]
	}

	reader, _, headers, err := client.GetObject(filePath, rangeHdr)
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", headers.Get("Content-Type"))
	w.Header().Set("Content-Length", headers.Get("Content-Length"))
	if headers.Get("Content-Range") != "" {
		w.Header().Set("Content-Range", headers.Get("Content-Range"))
	}

	if _, err := io.Copy(w, reader); err != nil {
		return err
	}

	return nil
}
