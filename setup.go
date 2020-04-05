package s3browser

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/caddyserver/caddy"
	"github.com/caddyserver/caddy/caddyhttp/httpserver"
)

func init() {
	caddy.RegisterPlugin("s3browser", caddy.Plugin{
		ServerType: "http",
		Action:     setup,
	})
}

// setup configures a new S3BROWSER middleware instance.
func setup(c *caddy.Controller) error {
	var err error

	b := &Browse{}

	// Parse config
	b.Config, err = ParseConfig(c)
	if err != nil {
		return err
	}

	// Initialize logger
	var l *log.Logger
	{
		output := ioutil.Discard
		if b.Config.Debug {
			output = os.Stdout
		}
		prefix := fmt.Sprintf("[s3browser][%s] ", b.Config.Bucket)
		l = log.New(output, prefix, log.LstdFlags)
	}

	l.Printf("Config:\n\t%#v\n", b.Config)

	{
		l.Println("Initializing S3 Cache")
		b.S3Cache, err = NewS3Cache(b.Config, l)
		if err == nil {
			err = b.S3Cache.Refresh()
		}
		if err != nil {
			return err
		}
	}

	b.Refresh = make(chan struct{})

	// Goroutine to trigger cache refresh (periodic/by request)
	go func() {
		for {
			select {
			case <-b.Refresh:
				l.Println("Refresh: API")
			case <-time.After(b.Config.Refresh):
				l.Println("Refresh: Periodic")
			}
			err := b.S3Cache.Refresh()
			if err != nil {
				l.Println(err)
			}
		}
	}()

	// Prepare template
	{
		l.Println("Parsing template")
		b.Template, err = parseTemplate()
		if err != nil {
			return err
		}

		// Try to render now to catch any error in template
		dir, _ := b.S3Cache.GetDir("/")
		err = b.renderHTML(ioutil.Discard, dir)
		if err != nil {
			return err
		}
	}

	// Add to Caddy
	cfg := httpserver.GetConfig(c)
	cfg.AddMiddleware(func(next httpserver.Handler) httpserver.Handler {
		l.Println("Initialization complete")
		b.Next = next
		return b
	})

	return nil
}
