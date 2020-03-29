package s3browser

import (
	"crypto/tls"
	"fmt"
	"github.com/caddyserver/caddy"
	"github.com/caddyserver/caddy/caddyhttp/httpserver"
	"github.com/minio/minio-go/v6"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"
)

var (
	updating bool
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
		cfg, err := parseDirective(c)
		if err != nil {
			return err
		}
		b.Config = cfg
	}

	if b.Config.Debug {
		fmt.Println("Config:")
		fmt.Println(b.Config)
	}
	updating = true
	if b.Config.Debug {
		fmt.Println("Fetching Files...")
	}
	b.Fs, err = getFiles(b)
	if b.Config.Debug {
		fmt.Println("Files...")
		fmt.Println(b.Fs)
	}
	updating = false
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
			if !updating {
				if b.Config.Debug {
					fmt.Println("Updating Files..")
				}
				if b.Fs, err = getFiles(b); err != nil {
					fmt.Println(err)
					updating = false
				}
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

func getFiles(b *Browse) (map[string]Directory, error) {
	var err error
	updating = true
	fs := make(map[string]Directory)
	fs["/"] = Directory{
		Path: "/",
	}
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

	doneCh := make(chan struct{})
	defer close(doneCh)
	objectCh := minioClient.ListObjects(b.Config.Bucket, "", true, doneCh)

	for obj := range objectCh {
		if obj.Err != nil {
			continue
		}

		dir, file := path.Split(obj.Key)
		if len(dir) > 0 && dir[:0] != "/" {
			dir = "/" + dir
		}
		if dir == "" {
			dir = "/" // if dir is empty, then set to root
		}
		// Note: dir should start & end with / now

		if len(getFolders(dir)) < 3 {
			// files are in root
			// less than three bc "/" split becomes ["",""]
			// Do nothing as file will get added below & root already exists
		} else {
			// TODO: loop through folders and ensure they are in the tree
			// make sure to add folder to parent as well
			foldersLen := len(getFolders(dir))
			for i := 2; i < foldersLen; i++ {
				parent := getParent(getFolders(dir), i)
				folder := getFolder(getFolders(dir), i)
				if b.Config.Debug {
					fmt.Printf("folders: %q i: %d parent: %s folder: %s\n", getFolders(dir), i, parent, folder)
				}

				// check if parent exists
				if _, ok := fs[parent]; !ok {
					// create parent
					fs[parent] = Directory{
						Path:    parent,
						Folders: []Folder{Folder{Name: getFolder(getFolders(dir), i)}},
					}
				}
				// check if folder itself exists
				if _, ok := fs[folder]; !ok {
					// create parent
					fs[folder] = Directory{
						Path: folder,
					}
					tmp := fs[parent]
					tmp.Folders = append(fs[parent].Folders, Folder{Name: getFolder(getFolders(dir), i)})
					fs[parent] = tmp
				}
			}
		}

		// STEP Two
		// add file to directory
		tempFile := File{Name: file, Bytes: obj.Size, Date: obj.LastModified, Folder: joinFolders(getFolders(dir))}
		fsCopy := fs[joinFolders(getFolders(dir))]
		fsCopy.Path = joinFolders(getFolders(dir))
		fsCopy.Files = append(fsCopy.Files, tempFile) // adding file list of files
		fs[joinFolders(getFolders(dir))] = fsCopy
	} // end looping through all the files
	updating = false
	return fs, nil
}

func getFolders(s string) []string {
	// first and last entry should be empty
	return strings.Split(s, "/")
}

func joinFolders(s []string) string {
	return strings.Join(s, "/")
}

func getParent(s []string, i int) string {
	// trim one from end
	if i < 3 {
		return "/"
	}
	s[i-1] = ""
	return joinFolders(s[0:(i)])
}

func getFolder(s []string, i int) string {
	if i < 3 {
		s[2] = ""
		return joinFolders(s[0:3])
	}
	s[i] = ""
	return joinFolders(s[0:(i + 1)])
}

func parseDirective(c *caddy.Controller) (cfg Config, err error) {
	c.NextArg() // skip block beginning: "s3browser"

	cfg = Config{
		Secure:  true,
		Debug:   false,
		Refresh: 5 * time.Minute,
	}
	for c.NextBlock() {
		var err error
		switch c.Val() {
		case "key":
			cfg.Key, err = StringArg(c)
		case "secret":
			cfg.Secret, err = StringArg(c)
		case "endpoint":
			cfg.Endpoint, err = StringArg(c)
		case "region":
			cfg.Region, err = StringArg(c)
		case "bucket":
			cfg.Bucket, err = StringArg(c)
		case "secure":
			cfg.Secure, err = BoolArg(c)
		case "refresh":
			cfg.Refresh, err = DurationArg(c)
		case "debug":
			cfg.Debug, err = BoolArg(c)
		default:
			err = c.Errf("Unknown s3browser arg: %s", c.Val())
		}
		if err != nil {
			return cfg, c.Errf("Error parsing %s: %s", c.Val(), err)
		}
	}
	return cfg, nil
}

// Assert only one arg and return it
func StringArg(c *caddy.Controller) (string, error) {
	args := c.RemainingArgs()
	if len(args) != 1 {
		return "", c.ArgErr()
	}
	return args[0], nil
}

func DurationArg(c *caddy.Controller) (time.Duration, error) {
	str, err := StringArg(c)
	if err != nil {
		return 0 * time.Second, err
	}
	return time.ParseDuration(str)
}

func BoolArg(c *caddy.Controller) (bool, error) {
	args := c.RemainingArgs()
	if len(args) != 1 {
		return true, c.ArgErr()
	}
	return strconv.ParseBool(args[0])
}
