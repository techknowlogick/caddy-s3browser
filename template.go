package s3browser

import (
	"html/template"
	"path"
	"regexp"
	"sort"
	"strings"

	"github.com/Masterminds/semver"
)

// semVerRegex is the regular expression used to parse a partial semantic version.
// We rely on github.com/Masterminds/semver for the actual parsing, but
// we want to consider the edge cases 1.0.0 vs. 1.0 vs 1.
var semVerRegex = regexp.MustCompile(`^v?([0-9]+)(\.[0-9]+)?(\.[0-9]+)?`)

type TemplateArgs struct {
	SiteName     string
	Dir          Directory
	SemanticSort bool
}

type Crumb struct {
	Link string
	Name string
}

type collection []*semver.Version

func parseTemplate() (*template.Template, error) {
	funcs := template.FuncMap{
		"SemSort":     semSort,
		"Breadcrumbs": breadcrumbs,
		"PathBase":    path.Base,
		"PathDir":     path.Dir,
		"PathJoin":    path.Join,
	}
	return template.New("listing").Funcs(funcs).Parse(defaultTemplate)
}

func semSort(args TemplateArgs, folders []string) []string {
	// No sorting if disabled
	if !args.SemanticSort {
		return folders
	}

	// Create one list with semver named folders, and one with the others
	internal := make([]*semver.Version, 0, len(folders))
	unversioned := make([]string, 0)
	for _, folder := range folders {
		version, err := semver.NewVersion(folder)
		if err != nil {
			// Folders not matching a version number go last
			unversioned = append(unversioned, folder)
			continue
		}
		internal = append(internal, version)
	}

	sort.Sort(collection(internal))

	versioned := make([]string, len(internal))
	for i := range internal {
		versioned[i] = internal[i].Original()
	}

	return append(versioned, unversioned...)
}

func breadcrumbs(args TemplateArgs) []Crumb {
	crumbs := []Crumb{
		Crumb{Link: "/", Name: args.SiteName},
	}

	dirPath := args.Dir.Path

	if dirPath == "/" {
		return crumbs
	}

	currPath := ""
	for _, currName := range strings.Split(strings.Trim(dirPath, "/"), "/") {
		currPath += "/" + currName
		crumbs = append(crumbs, Crumb{Link: currPath, Name: currName})
	}

	return crumbs
}

func (c collection) Len() int {
	return len(c)
}

func (c collection) Less(i, j int) bool {
	// Note: this function sorts backwards;
	// we invert j with i
	if c[i].Equal(c[j]) {
		// 1.1 is less than 1.1.0
		mi := semVerRegex.FindStringSubmatch(c[i].Original())
		mj := semVerRegex.FindStringSubmatch(c[j].Original())
		if mi != nil && mj != nil {
			return len(mj[0]) < len(mi[0])
		}
	}
	return c[j].LessThan(c[i])
}

func (c collection) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}

const defaultTemplate = `<!DOCTYPE html>
<html>
	<head>
		<title>{{ if ne .Dir.Path "/" }}{{ PathBase .Dir.Path }} | {{ end }}{{ .SiteName }}</title>
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
</style>
<!-- template source from https://github.com/caddyserver/caddy/blob/a2d71bdd94c0ca51dfb3b816b61911dac799581f/caddyhttp/browse/setup.go -->
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
				{{ range $_, $crumb := Breadcrumbs $ }}
					<a href="{{ html $crumb.Link }}">{{ html $crumb.Name }}</a> /
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
					{{ if ne .Dir.Path "/" }}
					<tr>
						<td></td>
						<td>
							<a href="{{ html (PathDir .Dir.Path) }}">
								<span class="goup">Go up</span>
							</a>
						</td>
						<td>&mdash;</td>
						<td class="hideable">&mdash;</td>
						<td class="hideable"></td>
					</tr>
					{{- end}}
					{{ range $name := SemSort $ .Dir.Folders }}
						<tr class="file">
							<td></td>
							<td>
								<a href="{{ html (PathJoin $.Dir.Path $name) }}">
									<svg width="1.5em" height="1em" version="1.1" viewBox="0 0 317 259"><use xlink:href="#folder"></use></svg>
									<span class="name">{{ html $name }}</span>
								</a>
							</td>
							<td>&mdash;</td>
							<td class="hideable">&mdash;</td>
							<td class="hideable"></td>
						</tr>
					{{ end }}
					{{ range $name, $info := .Dir.Files }}
						<tr class="file">
							<td></td>
							<td>
								<a href="{{ html (PathJoin $.Dir.Path $name) }}">
									<svg width="1.5em" height="1em" version="1.1" viewBox="0 0 265 323"><use xlink:href="#file"></use></svg>
									<span class="name">{{html $name}}</span>
								</a>
							</td>
							<td>{{ $info.HumanSize }}</td>
							<td class="hideable"><time datetime="{{ $info.HumanModTime "2006-01-02T15:04:05Z" }}">{{ $info.HumanModTime "01/02/2006 03:04:05 PM -07:00" }}</time></td>
							<td class="hideable"></td>
						</tr>
					{{- end}}
					</tbody>
				</table>
			</div>
		</main>
		<footer>
			Served by S3 Browser via <a rel="noopener noreferrer" href="https://caddyserver.com">Caddy</a>
		</footer>
	</body>
</html>`
