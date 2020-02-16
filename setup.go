package s3browser

import (
	"crypto/tls"
	"fmt"
	"github.com/caddyserver/caddy"
	"github.com/caddyserver/caddy/caddyhttp/httpserver"
	"github.com/minio/minio-go/v6"
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
	if b.Config.Debug {
		fmt.Println("Config:")
		fmt.Println(b.Config)
	}
	updating = true
	if b.Config.Debug {
		fmt.Println("Fetching Files..")
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
		Path: "/",
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

func parse(b *Browse, c *caddy.Controller) (err error) {
	c.RemainingArgs()
	b.Config = Config{}
	b.Config.Secure = true
	b.Config.Debug = false
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
		case "debug":
			b.Config.Debug, err = BoolArg(c)
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
		<meta name="viewport" content="width=device-width, initial-scale=1.0">
<style>
* { padding: 0; margin: 0; }
body {
	font-family: sans-serif;
	text-rendering: optimizespeed;
	background-color: #ffffff;
}
a {
	color: #006ed3;
	text-decoration: none;
}
a:hover,
h1 a:hover {
	color: #319cff;
}
header,
#summary {
	padding-left: 5%;
	padding-right: 5%;
}
th:first-child,
td:first-child {
	width: 5%;
}
th:last-child,
td:last-child {
	width: 5%;
}
header {
	padding-top: 25px;
	padding-bottom: 15px;
	background-color: #f2f2f2;
}
h1 {
	font-size: 20px;
	font-weight: normal;
	white-space: nowrap;
	overflow-x: hidden;
	text-overflow: ellipsis;
	color: #999;
}
h1 a {
	color: #000;
	margin: 0 4px;
}
h1 a:hover {
	text-decoration: underline;
}
h1 a:first-child {
	margin: 0;
}
main {
	display: block;
}
.meta {
	font-size: 12px;
	font-family: Verdana, sans-serif;
	border-bottom: 1px solid #9C9C9C;
	padding-top: 10px;
	padding-bottom: 10px;
}
.meta-item {
	margin-right: 1em;
}
#filter {
	padding: 4px;
	border: 1px solid #CCC;
}
table {
	width: 100%;
	border-collapse: collapse;
}
tr {
	border-bottom: 1px dashed #dadada;
}
tbody tr:hover {
	background-color: #ffffec;
}
th,
td {
	text-align: left;
	padding: 10px 0;
}
th {
	padding-top: 15px;
	padding-bottom: 15px;
	font-size: 16px;
	white-space: nowrap;
}
th a {
	color: black;
}
th svg {
	vertical-align: middle;
}
td {
	white-space: nowrap;
	font-size: 14px;
}
td:nth-child(2) {
	width: 80%;
}
td:nth-child(3) {
	padding: 0 20px 0 20px;
}
th:nth-child(4),
td:nth-child(4) {
	text-align: right;
}
td:nth-child(2) svg {
	position: absolute;
}
td .name,
td .goup {
	margin-left: 1.75em;
	word-break: break-all;
	overflow-wrap: break-word;
	white-space: pre-wrap;
}
.icon {
	margin-right: 5px;
}
.icon.sort {
	display: inline-block;
	width: 1em;
	height: 1em;
	position: relative;
	top: .2em;
}
.icon.sort .top {
	position: absolute;
	left: 0;
	top: -1px;
}
.icon.sort .bottom {
	position: absolute;
	bottom: -1px;
	left: 0;
}
footer {
	padding: 40px 20px;
	font-size: 12px;
	text-align: center;
}
@media (max-width: 600px) {
	.hideable {
		display: none;
	}
	td:nth-child(2) {
		width: auto;
	}
	th:nth-child(3),
	td:nth-child(3) {
		padding-right: 5%;
		text-align: right;
	}
	h1 {
		color: #000;
	}
	h1 a {
		margin: 0;
	}
	#filter {
		max-width: 100px;
	}
}
<!-- template source from https://github.com/caddyserver/caddy/blob/a2d71bdd94c0ca51dfb3b816b61911dac799581f/caddyhttp/browse/setup.go -->
</style>
	</head>
	<body>
		<svg version="1.1" xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" height="0" width="0" style="position: absolute;">
			<defs>
				<!-- Folder -->
				<g id="folder" fill-rule="nonzero" fill="none">
					<path d="M285.22 37.55h-142.6L110.9 0H31.7C14.25 0 0 16.9 0 37.55v75.1h316.92V75.1c0-20.65-14.26-37.55-31.7-37.55z" fill="#FFA000"/>
					<path d="M285.22 36H31.7C14.25 36 0 50.28 0 67.74v158.7c0 17.47 14.26 31.75 31.7 31.75H285.2c17.44 0 31.7-14.3 31.7-31.75V67.75c0-17.47-14.26-31.75-31.7-31.75z" fill="#FFCA28"/>
				</g>
				<!-- File -->
				<g id="file" stroke="#000" stroke-width="25" fill="#FFF" fill-rule="evenodd" stroke-linecap="round" stroke-linejoin="round">
					<path d="M13 24.12v274.76c0 6.16 5.87 11.12 13.17 11.12H239c7.3 0 13.17-4.96 13.17-11.12V136.15S132.6 13 128.37 13H26.17C18.87 13 13 17.96 13 24.12z"/>
					<path d="M129.37 13L129 113.9c0 10.58 7.26 19.1 16.27 19.1H249L129.37 13z"/>
				</g>
			</defs>
		</svg>
		<header>
			<h1>
				{{ range $i, $crumb := .Breadcrumbs }}
						<a href="/{{ html $crumb.Link }}">
							{{ html $crumb.ReadableName }}
						</a>
						{{if ne $i 0}}/{{end}}
				{{ end }}
			</h1>
		</header>
		<main>
			<div class="listing">
				<table aria-describedby="summary">
					<thead>
					<tr>
						<th></th>
						<th>
							Name
						</th>
						<th>
							Size
						</th>
						<th class="hideable">
							Modified
						</th>
						<th class="hideable"></th>
					</tr>
					</thead>
					<tbody>
					{{ if ne .Path "/" }}
					<tr>
						<td></td>
						<td>
							<a href="..">
								<span class="goup">Go up</span>
							</a>
						</td>
						<td>&mdash;</td>
						<td class="hideable">&mdash;</td>
						<td class="hideable"></td>
					</tr>
					{{- end}}
					{{ range .Folders }}
						<tr class="file">
							<td></td>
							<td>
								<a href="{{ html .Name }}">
									<svg width="1.5em" height="1em" version="1.1" viewBox="0 0 317 259"><use xlink:href="#folder"></use></svg>
									<span class="name">{{ .ReadableName }}</span>
								</a>
							</td>
							<td>&mdash;</td>
							<td class="hideable">&mdash;</td>
							<td class="hideable"></td>
						</tr>
					{{ end }}
					{{ range .Files }}
						{{ if ne .Name ""}}
							<tr class="file">
								<td></td>
								<td>
									<a href="./{{ html .Name }}">
										<svg width="1.5em" height="1em" version="1.1" viewBox="0 0 265 323"><use xlink:href="#file"></use></svg>
										<span class="name">{{html .Name}}</span>
									</a>
								</td>
								<td>{{.HumanSize}}</td>
								<td class="hideable"><time datetime="{{.HumanModTime "2006-01-02T15:04:05Z"}}">{{.HumanModTime "01/02/2006 03:04:05 PM -07:00"}}</time></td>
								<td class="hideable"></td>
							</tr>
						{{- end}}
					{{- end}}
					</tbody>
				</table>
			</div>
		</main>
		<footer>
			Served with <a rel="noopener noreferrer" href="https://caddyserver.com">Caddy</a>
		</footer>
	</body>
</html>`
