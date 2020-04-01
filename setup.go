package s3browser

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/caddyserver/caddy"
	"github.com/caddyserver/caddy/caddyhttp/httpserver"
	"github.com/minio/minio-go/v6"
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
	{
		cfg, err := ParseConfig(c)
		if err != nil {
			return err
		}
		b.Config = cfg
	}

	if b.Config.Debug {
		fmt.Println("Config:")
		fmt.Println(b.Config)
	}
	if b.Config.Debug {
		fmt.Println("Fetching Files...")
	}
	b.Fs, err = buildS3FsCache(b)
	if b.Config.Debug {
		fmt.Println("Files...")
		fmt.Println(b.Fs)
	}
	if err != nil {
		return err
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
				fmt.Println("Updating Files..")
			}
			if b.Fs, err = buildS3FsCache(b); err != nil {
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

func buildS3FsCache(b *Browse) (S3FsCache, error) {
	var err error

	fs := make(S3FsCache)
	fs["/"] = Directory{Path: "/"}

	var minioClient *minio.Client
	if b.Config.Region == "" {
		minioClient, err = minio.New(b.Config.Endpoint, b.Config.Key, b.Config.Secret, b.Config.Secure)
	} else {
		minioClient, err = minio.NewWithRegion(b.Config.Endpoint, b.Config.Key, b.Config.Secret, b.Config.Secure, b.Config.Region)
	}
	if err != nil {
		return fs, err
	}

	if !b.Config.Secure {
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		minioClient.SetCustomTransport(tr)
	}

	objectCh := minioClient.ListObjectsV2(
		b.Config.Bucket,
		"",   // prefix
		true, // recursive
		nil,  // doneChan
	)

	for obj := range objectCh {
		if obj.Err != nil {
			continue
		}

		objDir, objName := path.Split(obj.Key)

		// Ensure objDir starts with / but doesn't end with one
		objDir = "/" + strings.Trim(objDir, "/")

		// Add missing parent directories in `fs`
		if _, ok := fs[objDir]; !ok {
			dirs := strings.Split(strings.Trim(objDir, "/"), "/")

			parentPath := "/"
			for _, curr := range dirs {
				if b.Config.Debug {
					fmt.Printf("dirs: %q parentPath: %s curr: %s\n", dirs, parentPath, curr)
				}

				currPath := path.Join(parentPath, curr)
				if _, ok := fs[currPath]; !ok {
					if b.Config.Debug {
						fmt.Printf("+  dir: %s\n", currPath)
					}

					// Add to parent Node
					parentNode := fs[parentPath]
					parentNode.Folders = append(parentNode.Folders, Folder{Name: curr})
					fs[parentPath] = parentNode

					// Add own Node
					fs[currPath] = Directory{Path: currPath}
				}

				if parentPath != "/" {
					parentPath += "/"
				}
				parentPath += curr
			}
		}

		// Add the object
		if objName != "" { // "": obj is the directory itself
			if b.Config.Debug {
				fmt.Printf("+ file: %s/%s\n", objDir, objName)
			}

			fsCopy := fs[objDir]
			fsCopy.Files = append(fsCopy.Files, File{
				Name:  objName,
				Bytes: obj.Size,
				Date:  obj.LastModified,
			})
			fs[objDir] = fsCopy
		}
	}

	return fs, nil
}
