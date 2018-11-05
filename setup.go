package s3browser

import (
	"strings"
	"path"
	"fmt"
	"html/template"
	"github.com/mholt/caddy"
	"github.com/mholt/caddy/caddyhttp/httpserver"
	"github.com/minio/minio-go"
)

func init() {
	caddy.RegisterPlugin("s3browser", caddy.Plugin{
		ServerType: "http",
		Action:     setup,
	})
}

// setup configures a new S3BROWSER middleware instance.
func setup(c *caddy.Controller) error {
	fs := make(map[string]Directory)
	cfg := httpserver.GetConfig(c)

	b := &Browse{}
	if err := parse(b, c); err != nil {
		return err
	}

	fs["/"] = Directory{
		Path: "/",
		CanGoUp: false,
	}

	minioClient, err := minio.New(b.Config.Endpoint, b.Config.Key, b.Config.Secret, true)
	if err != nil {
		return err
	}

	b.Client = minioClient

	doneCh := make(chan struct{})
	defer close(doneCh)
	objectCh := b.Client.ListObjectsV2(b.Config.Bucket, "", true, doneCh)

	for obj := range objectCh {
		// fmt.Println(obj)
		if obj.Err != nil {
			continue
		}
		
		dir, file := path.Split(obj.Key)
		// if len(dir) > 0 && dir[len(dir)-1:] == "/" {
		// 	dir = dir[:len(dir)-1]
		// }
		if len(dir) > 0 && dir[:0] == "/" {
			dir = dir[1:]
		}
		if dir == "" {
			dir = "/"
		}

		// STEP ONE Check dir hirearchy
		// need to split on directory split
		fmt.Println("=====start file processing")
		if _, ok := fs[dir]; !ok {
			fmt.Printf("=====fsdir %s no exist\n", dir)
			tempDir := strings.Split(dir, "/")
			if len(tempDir) > 0 {
				tempDir = tempDir[:len(tempDir)-1]
			}
			built := ""
			// also loop through breadcrumb to check those as well
			fmt.Printf("=====loop over %v\n", tempDir)
			for _, tempFolder := range tempDir {
				if len(tempFolder) < 1 {
					continue
				}
				if len(built) < 1 {
					built = tempFolder
				}else {
					built = built + "/" +tempFolder +"/"
				}
				fmt.Printf("=====dealing with %s\n", built)
				
				if _, ok2 := fs[built+"/"]; !ok2 {
					fmt.Printf("=========== no exists %s \n",built+"/")
					fs[built+"/"] = Directory{
						Path: built+"/",
						CanGoUp: true,
					}
					// also find parent and inject as a folder
					count := strings.Count(built, "/")
					if count > 0 {
						removeEnd := strings.SplitN(built, "/", count)
						fmt.Printf("wtf is removeEnd1 %s\n", removeEnd)
						if len(removeEnd) > 0 {
							removeEnd = removeEnd[:len(removeEnd)-1]
						}
						fmt.Printf("wtf is removeEnd2 %s\n", removeEnd)
						noEnd := strings.Join(removeEnd,"/")+"/"
						fmt.Printf("wtf is noEnd %s\n", noEnd)
						tempFs := fs[noEnd]
						tempFs.Folders = append(tempFs.Folders, Folder{Name: built})
						fs[noEnd] = tempFs
						fmt.Printf("injecting %s into %s\n", built, noEnd)
					} else {
						tempFs := fs["/"]
						tempFs.Folders = append(tempFs.Folders, Folder{Name: built+"/"})
						fs["/"] = tempFs
						fmt.Printf("injecting %s into %s\n", built, "/")
					}
				}
			}
		} // if hierachy exists?

		// STEP Two
		// add file to directory
		tempFile := File{Name: file, Bytes: obj.Size, Date: obj.LastModified, Folder: fmt.Sprintf("/%s",dir)}
		y := fs[dir]
		if dir != "/" {
			y.CanGoUp = true
		}
		y.Files = append(y.Files, tempFile)
		fs[dir] = y
	}

	b.Fs = fs

	fmt.Println(fs)

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

func parse(b *Browse, c *caddy.Controller) (err error) {
	c.RemainingArgs()
	b.Config = Config{}
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

const defaultTemplate = `<!DOCTYPE html>
<html>
	<head>
		<title>{{ .Path }} | S3 Browser</title>

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
										{{ html .Name }}
									</a>
								</td>
								<td class="size">
									-
								</td>
								<td class="modified">
									-
								</td>
							</tr>
						{{ end }}
						{{ range .Files }}
							<tr>
								<td class="icon">
									<span class="glyphicon glyphicon-file" aria-hidden="true"></span>
								</td>
								<td class="name">
									<a href="./{{ html .Name }}">
										{{ html .Name }}
									</a>
								</td>
								<td class="size">
									{{ .Bytes }}
								</td>
								<td class="modified">
									{{ .Date }}
								</td>
							</tr>
						{{ end }}
					</tbody>
				</table>
			</div>
		</div>
	</body>
</html>
`
