package s3browser

import (
	"crypto/tls"
	"fmt"
	"github.com/mholt/caddy"
	"github.com/mholt/caddy/caddyhttp/httpserver"
	"github.com/minio/minio-go"
	"html/template"
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
	cfg := httpserver.GetConfig(c)

	b := &Browse{}
	if err = parse(b, c); err != nil {
		return err
	}
	updating = true
	b.Fs, err = getFiles(b)
	updating = false
	if err != nil {
		return err
	}
	var duration time.Duration
	if b.Config.Refresh == "" {
		b.Config.Refresh = "5m"
	}
	duration, err = time.ParseDuration(b.Config.Refresh)
	if err != nil {
		fmt.Println("error parsing refresh, falling back to default of 5 minutes")
		duration = 5 * time.Minute
	}
	ticker := time.NewTicker(duration)
	go func() {
		// create more indexes every X minutes based off interval
		for range ticker.C {
			if !updating {
				if b.Fs, err = getFiles(b); err != nil {
					fmt.Println(err)
					updating = false
				}
			}
		}
	}()

	tpl, err := template.New("listing").Parse(defaultTemplate)
	if err != nil {
		return err
	}
	b.Template = tpl

	cfg.AddMiddleware(func(next httpserver.Handler) httpserver.Handler {
		b.Next = next
		return b
	})

	return nil
}

func getFiles(b *Browse) (map[string]Directory, error) {
	updating = true
	fs := make(map[string]Directory)
	fs["/"] = Directory{
		Path:    "/",
		CanGoUp: false,
	}
	minioClient, err := minio.New(b.Config.Endpoint, b.Config.Key, b.Config.Secret, b.Config.Secure)
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
		if len(dir) > 0 && dir[:0] == "/" {
			dir = dir[1:]
		}
		if dir == "" {
			dir = "/"
		}

		// STEP ONE Check dir hirearchy
		// need to split on directory split
		if _, ok := fs[dir]; !ok {
			tempDir := strings.Split(dir, "/")
			if len(tempDir) > 0 {
				tempDir = tempDir[:len(tempDir)-1]
			}
			built := ""
			// also loop through breadcrumb to check those as well
			for _, tempFolder := range tempDir {
				if len(tempFolder) < 1 {
					continue
				}
				if len(built) < 1 {
					built = tempFolder
				} else {
					built = built + "/" + tempFolder + "/"
				}

				if _, ok2 := fs[built]; !ok2 {
					fs[built] = Directory{
						Path:    built + "/",
						CanGoUp: true,
					}
					// also find parent and inject as a folder
					count := strings.Count(built, "/")
					if count > 0 {
						removeEnd := strings.SplitN(built, "/", count)
						if len(removeEnd) > 0 {
							removeEnd = removeEnd[:len(removeEnd)-1]
						}
						noEnd := strings.Join(removeEnd, "/") + "/"
						tempFs := fs[noEnd]
						tempFs.Folders = append(tempFs.Folders, Folder{Name: built})
						fs[noEnd] = tempFs
					} else {
						tempFs := fs["/"]
						tempFs.Folders = append(tempFs.Folders, Folder{Name: built + "/"})
						fs["/"] = tempFs
					}
				}
			}
		} // if hierachy exists?

		// STEP Two
		// add file to directory
		tempFile := File{Name: file, Bytes: obj.Size, Date: obj.LastModified, Folder: fmt.Sprintf("/%s", dir)}
		y := fs[dir]
		if dir != "/" {
			y.CanGoUp = true
		}
		y.Path = dir
		y.Files = append(y.Files, tempFile)
		fs[dir] = y
	} // end looping through all the files
	updating = false
	return fs, nil
}

func parse(b *Browse, c *caddy.Controller) (err error) {
	c.RemainingArgs()
	b.Config = Config{}
	b.Config.Secure = true
	for c.NextBlock() {
		var err error
		switch c.Val() {
		case "key":
			b.Config.Key, err = StringArg(c)
		case "secret":
			b.Config.Secret, err = StringArg(c)
		case "endpoint":
			b.Config.Endpoint, err = StringArg(c)
		case "bucket":
			b.Config.Bucket, err = StringArg(c)
		case "secure":
			b.Config.Secure, err = BoolArg(c)
		case "refresh":
			b.Config.Refresh, err = StringArg(c)
		default:
			return c.Errf("Unknown s3browser arg: %s", c.Val())
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// Assert only one arg and return it
func StringArg(c *caddy.Controller) (string, error) {
	args := c.RemainingArgs()
	if len(args) != 1 {
		return "", c.ArgErr()
	}
	return args[0], nil
}

func BoolArg(c *caddy.Controller) (bool, error) {
	args := c.RemainingArgs()
	if len(args) != 1 {
		return true, c.ArgErr()
	}
	return strconv.ParseBool(args[0])
}

const defaultTemplate = `<!DOCTYPE html>
<html>
	<head>
		<title>{{ .ReadableName }} | S3 Browser</title>

		<meta charset="utf-8">
		<meta name="viewport" content="width=device-width, initial-scale=1">
		<meta http-equiv="X-UA-Compatible" content="IE=edge">

		<link rel="stylesheet" href="//cdnjs.cloudflare.com/ajax/libs/twitter-bootstrap/3.3.6/css/bootstrap.min.css">
		<link rel="stylesheet" href="//cdnjs.cloudflare.com/ajax/libs/flat-ui/2.3.0/css/flat-ui.min.css">

		<style>
			body {
				cursor: default;
			}

			.navbar {
				margin-bottom: 20px;
			}

			.credits {
				padding-left: 15px;
				padding-right: 15px;
			}

			h1 {
				font-size: 20px;
				margin: 0;
			}

			th .glyphicon {
				font-size: 15px;
			}

			table .icon {
				width: 30px;
			}
		</style>
    <!-- template source from https://raw.githubusercontent.com/dockhippie/caddy/master/rootfs/etc/caddy/browse.tmpl -->
	</head>
	<body>
		<nav class="navbar navbar-inverse navbar-static-top">
			<div class="container-fluid">
				<div class="navbar-header">
					<a class="navbar-brand" href="/">
						S3 Browser
					</a>
				</div>

				<div class="navbar-text navbar-right hidden-xs credits">
					Powered by <a href="https://caddyserver.com">Caddy</a>
				</div>
			</div>
		</nav>

		<div class="container-fluid">
			<ol class="breadcrumb">
				<li>
					<a href="/"><span class="glyphicon glyphicon-home" aria-hidden="true"></span></a>
				</li>
				{{ range .Breadcrumbs }}
					<li>
						<a href="/{{ html .Link }}">
							{{ html .ReadableName }}
						</a>
					</li>
				{{ end }}
			</ol>

			<div class="panel panel-default">
				<table class="table table-hover table-striped">
					<thead>
						<tr>
							<th class="icon"></th>
							<th class="name">
								Name
							</th>
							<th class="size col-sm-2">
								Size
							</th>
							<th class="modified col-sm-2">
								Modified
							</th>
						</tr>
					</thead>

					<tbody>
						{{ if .CanGoUp }}
							<tr>
								<td>
									<span class="glyphicon glyphicon-arrow-left" aria-hidden="true"></span>
								</td>
								<td>
									<a href="..">
										Go up
									</a>
								</td>
								<td>
									&mdash;
								</td>
								<td>
									&mdash;
								</td>
							</tr>
						{{ end }}
						{{ range .Folders }}
							<tr>
								<td class="icon">
									<span class="glyphicon glyphicon-folder-close" aria-hidden="true"></span>
								</td>
								<td class="name">
									<a href="/{{ html .Name }}">
										{{ .ReadableName }}
									</a>
								</td>
								<td class="size">
									&mdash;
								</td>
								<td class="modified">
									&mdash;
								</td>
							</tr>
						{{ end }}
						{{ range .Files }}
							{{ if ne .Name ""}}
							<tr>
								<td class="icon">
									<span class="glyphicon glyphicon-file" aria-hidden="true"></span>
								</td>
								<td class="name">
									<a href="./{{ html .Name }}">
										{{ .Name }}
									</a>
								</td>
								<td class="size">
									{{ .HumanSize }}
								</td>
								<td class="modified">
									{{ .HumanModTime "01/02/2006 03:04:05 PM" }}
								</td>
							</tr>
							{{ end }}
						{{ end }}
					</tbody>
				</table>
			</div>
		</div>
	</body>
</html>
`
