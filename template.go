package s3browser

import (
	"html/template"
	"path"
	"strings"
)

type TemplateArgs struct {
	SiteName     string
	Dir          Directory
	SemanticSort bool
}

type Crumb struct {
	Link string
	Name string
}

func parseTemplate() (*template.Template, error) {
	funcs := template.FuncMap{
		"Breadcrumbs": breadcrumbs,
		"PathBase":    path.Base,
		"PathDir":     path.Dir,
		"PathJoin":    path.Join,
	}
	return template.New("listing").Funcs(funcs).Parse(defaultTemplate)
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
tr.current {
	background-color: #eeffee;
}
tbody tr:hover {
	background-color: #ffffec;
}
th,
td {
	text-align: left;
	padding: 10px;
	vertical-align: text-top;
}
th {
	padding: 15px 10px;
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
td.description {
	width: auto;
}
th.size,
td.size {
	text-align: right;
}
td.name svg {
	position: absolute;
	font-size: 140%;
	margin-top: -0.1ex;
}
td .name,
td .goup {
	margin-left: 2.35em;
	white-space: pre;
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
				<!-- "folder": normal looking folder -->
				<g id="folder" fill-rule="nonzero" fill="none">
					<path d="M285.22 37.55h-142.6L110.9 0H31.7C14.25 0 0 16.9 0 37.55v75.1h316.92V75.1c0-20.65-14.26-37.55-31.7-37.55z" fill="#FFA000"/>
					<path d="M285.22 36H31.7C14.25 36 0 50.28 0 67.74v158.7c0 17.47 14.26 31.75 31.7 31.75H285.2c17.44 0 31.7-14.3 31.7-31.75V67.75c0-17.47-14.26-31.75-31.7-31.75z" fill="#FFCA28"/>
				</g>
				<!-- "file": normal looking file -->
				<g id="file" stroke="#000" stroke-width="25" fill="#FFF" fill-rule="evenodd" stroke-linecap="round" stroke-linejoin="round">
					<path d="M13 24.12v274.76c0 6.16 5.87 11.12 13.17 11.12H239c7.3 0 13.17-4.96 13.17-11.12V136.15S132.6 13 128.37 13H26.17C18.87 13 13 17.96 13 24.12z"/>
					<path d="M129.37 13L129 113.9c0 10.58 7.26 19.1 16.27 19.1H249L129.37 13z"/>
				</g>
				<!-- "current": folder with a green tilde inside (meant for the current release, like 1.11.6) -->
				<g id="current" fill-rule="nonzero" fill="none">
					<path style="fill:#ffa000;stroke-width:1.00025" d="M 285.292,37.559481 H 142.656 L 110.92799,1.7359907e-6 H 31.708001 C 14.253597,1.7359907e-6 0,16.904267 0,37.559481 V 112.67844 H 317 V 75.118957 C 317,54.463744 302.73639,37.559481 285.292,37.559481 Z" />
					<path style="fill:#ffca28;stroke-width:1.00025" d="M 285.292,36.009088 H 31.708001 C 14.253597,36.009088 0,50.292693 0,67.757101 V 226.49717 c 0,17.47439 14.263599,31.75801 31.708001,31.75801 H 285.27198 c 17.44441,0 31.70801,-14.30362 31.70801,-31.75801 V 67.767106 c 0,-17.474413 -14.2636,-31.758015 -31.70801,-31.758015 z" />
					<path style="fill:#00aa00;stroke-width:12.1622" d="m 250.19279,113.55876 -97.29813,97.29812 -48.64883,-48.64929 18.24315,-18.24314 30.40568,30.4057 79.05453,-79.054991 z" fill-rule="evenodd" />
				</g>
				<!-- "working": a folder with a gear icon inside (meant for the current working branches like 1, 1.12, 1.12.0-dev, 1.12.0-rc1) -->
				<g id="working" fill-rule="nonzero" fill="none">
					<path style="fill:#ffa000;stroke-width:1.00025" d="M 285.292,37.559481 H 142.656 L 110.92799,1.7359907e-6 H 31.708001 C 14.253597,1.7359907e-6 0,16.904267 0,37.559481 V 112.67844 H 317 V 75.118957 C 317,54.463744 302.73639,37.559481 285.292,37.559481 Z" />
					<path style="fill:#ffca28;stroke-width:1.00025" d="M 285.292,36.009088 H 31.708001 C 14.253597,36.009088 0,50.292693 0,67.757101 V 226.49717 c 0,17.47439 14.263599,31.75801 31.708001,31.75801 H 285.27198 c 17.44441,0 31.70801,-14.30362 31.70801,-31.75801 V 67.767106 c 0,-17.474413 -14.2636,-31.758015 -31.70801,-31.758015 z" />
					<path style="fill:#7b7b7b;stroke-width:1.00025" d="M 235.76019,164.49959 V 139.5103 h -20.24143 c -1.42673,-6.24505 -3.91685,-12.06575 -7.22395,-17.32448 l 14.34987,-14.34903 -17.68045,-17.666867 -14.34246,14.335447 c -5.26449,-3.30628 -11.08518,-5.79515 -17.33106,-7.2219 V 77.040811 H 148.30142 V 97.28347 c -6.24711,1.42675 -12.06656,3.91562 -17.33231,7.23384 L 116.62008,90.169923 98.953224,107.83679 113.30267,122.18582 c -3.31205,5.25873 -5.80216,11.07943 -7.22972,17.32448 H 85.831113 v 24.98929 h 20.241427 c 1.42756,6.2471 3.91109,12.06739 7.22395,17.32654 l -14.343266,14.34779 17.668916,17.68045 14.34697,-14.34986 c 5.26575,3.30709 11.0852,5.7972 17.33231,7.22396 v 20.24143 h 24.98929 v -20.24143 c 6.24628,-1.42676 12.05956,-3.91687 17.32571,-7.22396 l 14.34781,14.34986 17.68045,-17.68045 -14.34987,-14.34779 c 3.3071,-5.27233 5.79722,-11.07944 7.22395,-17.32654 z m -74.96413,18.74175 c -17.25238,0 -31.23597,-13.98442 -31.23597,-31.23599 0,-17.25362 13.98359,-31.23434 31.23597,-31.23434 17.25239,0 31.23516,13.98072 31.23516,31.23434 4e-4,17.25116 -13.98236,31.23599 -31.23516,31.23599 z" />
				</g>
				<!-- "release": a dimmed folder with a dimmed green tilde inside (meant for official releases other than the latest, like 1.10.5, 1.11.0, 1.11.5) -->
				<g id="release" fill-rule="nonzero" fill="none">
					<path style="fill:#ffd48b;stroke-width:1.00025" d="M 285.292,37.559481 H 142.656 L 110.92799,1.7359907e-6 H 31.708001 C 14.253597,1.7359907e-6 0,16.904267 0,37.559481 V 112.67844 H 317 V 75.118957 C 317,54.463744 302.73639,37.559481 285.292,37.559481 Z" />
					<path style="fill:#ffe596;stroke-width:1.00025" d="M 285.292,36.009088 H 31.708001 C 14.253597,36.009088 0,50.292693 0,67.757101 V 226.49717 c 0,17.47439 14.263599,31.75801 31.708001,31.75801 H 285.27198 c 17.44441,0 31.70801,-14.30362 31.70801,-31.75801 V 67.767106 c 0,-17.474413 -14.2636,-31.758015 -31.70801,-31.758015 z" />
					<path style="fill:#8abc8a;stroke-width:12.1622" d="m 250.19279,113.55876 -97.29813,97.29812 -48.64883,-48.64929 18.24315,-18.24314 30.40568,30.4057 79.05453,-79.054991 z" />
				</g>
				<!-- "older": a grayed out folder with a gear icon inside (mean for older working branches or RC, like 1.10, 1.11, 1.11.0-rc1) -->
				<g id="older" fill-rule="nonzero" fill="none">
					<path style="fill:#808080;stroke-width:1.00025" d="M 285.292,37.559481 H 142.656 L 110.92799,1.7359907e-6 H 31.708001 C 14.253597,1.7359907e-6 0,16.904267 0,37.559481 V 112.67844 H 317 V 75.118957 C 317,54.463744 302.73639,37.559481 285.292,37.559481 Z" />
					<path style="fill:#cccccc;stroke-width:1.00025" d="M 285.292,36.009088 H 31.708001 C 14.253597,36.009088 0,50.292693 0,67.757101 V 226.49717 c 0,17.47439 14.263599,31.75801 31.708001,31.75801 H 285.27198 c 17.44441,0 31.70801,-14.30362 31.70801,-31.75801 V 67.767106 c 0,-17.474413 -14.2636,-31.758015 -31.70801,-31.758015 z" />
					<path style="fill:#808080;stroke-width:1.00025" d="M 235.76019,164.49959 V 139.5103 h -20.24143 c -1.42673,-6.24505 -3.91685,-12.06575 -7.22395,-17.32448 l 14.34987,-14.34903 -17.68045,-17.666867 -14.34246,14.335447 c -5.26449,-3.30628 -11.08518,-5.79515 -17.33106,-7.2219 V 77.040811 H 148.30142 V 97.28347 c -6.24711,1.42675 -12.06656,3.91562 -17.33231,7.23384 L 116.62008,90.169923 98.953224,107.83679 113.30267,122.18582 c -3.31205,5.25873 -5.80216,11.07943 -7.22972,17.32448 H 85.831113 v 24.98929 h 20.241427 c 1.42756,6.2471 3.91109,12.06739 7.22395,17.32654 l -14.343266,14.34779 17.668916,17.68045 14.34697,-14.34986 c 5.26575,3.30709 11.0852,5.7972 17.33231,7.22396 v 20.24143 h 24.98929 v -20.24143 c 6.24628,-1.42676 12.05956,-3.91687 17.32571,-7.22396 l 14.34781,14.34986 17.68045,-17.68045 -14.34987,-14.34779 c 3.3071,-5.27233 5.79722,-11.07944 7.22395,-17.32654 z m -74.96413,18.74175 c -17.25238,0 -31.23597,-13.98442 -31.23597,-31.23599 0,-17.25362 13.98359,-31.23434 31.23597,-31.23434 17.25239,0 31.23516,13.98072 31.23516,31.23434 4e-4,17.25116 -13.98236,31.23599 -31.23516,31.23599 z" />
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
						<th class="hideable"></th>
						<th class="name">Name</th>
						<th class="description">Description</th>
						<th class="size">Size</th>
						<th class="date hideable">Modified</th>
						<th class="hideable"></th>
					</tr>
					</thead>
					<tbody>
					{{ if ne .Dir.Path "/" }}
					<tr>
						<td class="hideable"></td>
						<td class="goup">
							<a href="{{ html (PathDir .Dir.Path) }}">
								Go up
							</a>
						</td>
						<td class="description">&mdash;</td>
						<td class="size">&mdash;</td>
						<td class="date hideable">&mdash;</td>
						<td class="hideable"></td>
					</tr>
					{{- end}}
					{{ range $dir := .Dir.RenderedDirs }}
						<tr class="file {{ $dir.Class }}">
							<td class="hideable"></td>
							<td class="name">
								{{- /* Prevent spaces from being rendered due to white-space: pre */ -}}
								<a href="{{ html (PathJoin $.Dir.Path $dir.Name) }}"{{ if ne $dir.Description "" }} title="{{ $dir.Description }}{{- end}}">
								{{- /* */ -}}
								<svg width="1.5em" height="1em" version="1.1" viewBox="0 0 317 259"><use xlink:href="#{{$dir.Icon}}"></use></svg>
								{{- /* */ -}}
								<span class="name">{{ html $dir.Name }}</span>
								{{- /* */ -}}
								</a>
								{{- /* */ -}}
							</td>
							<td class="description">{{ $dir.Description }}</td>
							<td class="size">&mdash;</td>
							<td class="date hideable">&mdash;</td>
							<td class="hideable"></td>
						</tr>
					{{ end }}
					{{ range $file := .Dir.RenderedFiles }}
						<tr class="file {{ $file.Class }}">
							<td class="hideable"></td>
							<td class="name">
								{{- /* Prevent spaces from being rendered due to white-space: pre */ -}}
								<a href="{{ html (PathJoin $.Dir.Path $file.Name) }}">
									{{- /* */ -}}
									<svg width="1.5em" height="1em" version="1.1" viewBox="0 0 265 323"><use xlink:href="#{{ $file.Icon }}"></use></svg>
									{{- /* */ -}}
									<span class="name">{{html $file.Name}}</span>
									{{- /* */ -}}
								</a>
								{{- /* */ -}}
							</td>
							<td class="description">{{ $file.Description }}</td>
							<td class="size">{{ $file.HumanSize }}</td>
							<td class="date hideable"><time datetime="{{ $file.HumanModTime "2006-01-02T15:04:05Z" }}">{{ $file.HumanModTime "01/02/2006 03:04:05 PM -07:00" }}</time></td>
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
