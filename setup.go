package s3browser

import (
	"fmt"
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

	if b.Config.Debug {
		fmt.Println("Config:")
		fmt.Println(b.Config)
	}

	{
		if b.Config.Debug {
			fmt.Println("Initialising S3 Cache...")
		}
		b.S3Cache, err = NewS3Cache(b.Config)
		if err == nil {
			err = b.S3Cache.Refresh()
		}
		if err != nil {
			return err
		}
		if b.Config.Debug {
			fmt.Println("S3 Cache:")
			fmt.Println(b.S3Cache)
		}
	}

	b.Refresh = make(chan struct{})

	// Goroutine to trigger cache refresh (periodic/by request)
	go func() {
		for {
			select {
			case <-b.Refresh: // refresh API call
			case <-time.After(b.Config.Refresh * time.Second): // refresh after configured time
			}
			if b.Config.Debug {
				fmt.Println("Updating Files...")
			}
			err := b.S3Cache.Refresh()
			if err != nil {
				fmt.Println(err)
			}
		}
	}()

	// Prepare template
	{
		tpl, err := parseTemplate()
		if err != nil {
			return err
		}
		b.Template = tpl
	}

	// Add to Caddy
	cfg := httpserver.GetConfig(c)
	cfg.AddMiddleware(func(next httpserver.Handler) httpserver.Handler {
		b.Next = next
		return b
	})

	return nil
}
