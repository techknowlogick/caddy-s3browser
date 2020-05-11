package s3browser

// This file handles Plugin initialization.
// The methods are in the same order Caddy calls them:
//  - init
//  - CaddyModule
//  - CaddyModule:New
//  - parseCaddyfile (see caddyfile.go)
//  - UnmarshalCaddyfile
//  - Provision
//  - Validate
//
// After all that the plugin is installed as a middleware
// and Caddy calls ServeHTTP (serve.go) for each request.

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"time"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"go.uber.org/zap"
)

// Interface guards
var (
	_ caddy.Provisioner           = (*S3Browser)(nil)
	_ caddy.Validator             = (*S3Browser)(nil)
	_ caddyfile.Unmarshaler       = (*S3Browser)(nil)
	_ caddyhttp.MiddlewareHandler = (*S3Browser)(nil)
)

func init() {
	caddy.RegisterModule(S3Browser{})
	httpcaddyfile.RegisterHandlerDirective("s3browser", parseCaddyfile)
}

type S3Browser struct {
	// Config (these fields must be public)
	SiteName          string        `json:"site_name,omitempty"`
	Endpoint          string        `json:"endpoint,omitempty"`
	Region            string        `json:"region,omitempty"`
	Key               string        `json:"key,omitempty"`
	Secret            string        `json:"secret,omitempty"`
	Bucket            string        `json:"bucket,omitempty"`
	Secure            bool          `json:"secure,omitempty"`
	RefreshInterval   time.Duration `json:"refresh_interval,omitempty"`
	RefreshAPISecret  string        `json:"refresh_api_secret,omitempty"`
	Debug             bool          `json:"debug,omitempty"`
	SignedURLRedirect bool          `json:"signed_url_redirect,omitempty"`
	SemanticSort      bool          `json:"semantic_sort,omitempty"`

	s3Cache        S3FsCache
	template       *template.Template
	refreshTrigger chan struct{}

	log *zap.Logger
}

func (S3Browser) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "http.handlers.s3browser",
		New: func() caddy.Module { return new(S3Browser) },
	}
}

func (b *S3Browser) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	d.NextArg() // skip block beginning: "s3browser"

	for d.NextBlock(0) {
		var err error
		switch d.Val() {
		case "site_name":
			err = parseStringArg(d, &b.SiteName)
		case "endpoint":
			err = parseStringArg(d, &b.Endpoint)
		case "region":
			err = parseStringArg(d, &b.Region)
		case "key":
			err = parseStringArg(d, &b.Key)
		case "secret":
			err = parseStringArg(d, &b.Secret)
		case "bucket":
			err = parseStringArg(d, &b.Bucket)
		case "secure":
			err = parseBoolArg(d, &b.Secure)
		case "refresh_interval":
			err = parseDurationArg(d, &b.RefreshInterval)
		case "refresh_api_secret":
			err = parseStringArg(d, &b.RefreshAPISecret)
		case "debug":
			err = parseBoolArg(d, &b.Debug)
		case "signed_url_redirect":
			err = parseBoolArg(d, &b.SignedURLRedirect)
		case "semantic_sort":
			err = parseBoolArg(d, &b.SemanticSort)
		default:
			err = d.Errf("not a valid s3browser option")
		}
		if err != nil {
			return d.Errf("Error parsing %s: %s", d.Val(), err)
		}
	}

	return nil
}

func (b *S3Browser) Provision(ctx caddy.Context) (err error) {
	b.log = ctx.Logger(b)

	{
		b.log.Debug("Initializing S3 Cache")
		// Manually create the client so we can check the error
		c, err := NewS3Client(b.Endpoint, b.Key, b.Secret, b.Secure, b.Bucket)
		if err == nil {
			b.s3Cache, err = NewS3Cache(c, b.log)
			if err == nil {
				err = b.s3Cache.Refresh()
			}
		}
		if err != nil {
			return err
		}
	}

	b.refreshTrigger = make(chan struct{})

	// Goroutine to trigger cache refresh (periodic/by request)
	go func() {
		for {
			select {
			case <-b.refreshTrigger:
				b.log.Debug("refresh", zap.String("source", "api"))
			case <-time.After(b.RefreshInterval):
				b.log.Debug("refresh", zap.String("source", "timer"))
			}
			err := b.s3Cache.Refresh()
			if err != nil {
				b.log.Error("Could not refresh", zap.Error(err))
			}
		}
	}()

	// Prepare template
	{
		b.log.Debug("Parsing template")
		b.template, err = parseTemplate()
		if err != nil {
			return err
		}

		// Try to render now to catch any error in template
		dir, _ := b.s3Cache.GetDir("/")
		err = b.renderHTML(ioutil.Discard, dir)
		if err != nil {
			return err
		}
	}

	return nil
}

func (b *S3Browser) Validate() error {
	if b.SiteName == "" {
		return fmt.Errorf("no sitename")
	}
	if b.Endpoint == "" {
		return fmt.Errorf("no endpoint")
	}
	if b.Region == "" {
		return fmt.Errorf("no region")
	}
	if b.Key == "" {
		return fmt.Errorf("no key")
	}
	if b.Secret == "" {
		return fmt.Errorf("no secret")
	}
	if b.Bucket == "" {
		return fmt.Errorf("no bucket")
	}
	return nil
}

func (b *S3Browser) newS3Client() S3Client {
	c, err := NewS3Client(b.Endpoint, b.Key, b.Secret, b.Secure, b.Bucket)
	if err != nil {
		// Should never happen because we already validated the params in Provision
		b.log.Fatal("NewS3Client failed", zap.Error(err))
	}
	return c
}
