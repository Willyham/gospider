package reporter

import (
	"html/template"
	"io"
	"net/url"
	"sync"
)

// Interface describes a reporter.
type Interface interface {
	Add(uri *url.URL, links []*url.URL, assets []string)
	Report(io.Writer) error
}

var sitemapHTML = `
<html>
<head></head>
<body>
	{{ range $key, $value := . }}
		<div>
		 <h2><div id="{{ $key.Path }}">Page {{ $key }}</div></h2>
		 <h4>Has assets:</h4>
		 {{ range $value.Assets }}
				<li>{{ . }}</li>
		 {{ end }}
		 <h4>Links to:</h4>
		 {{ range $value.Links }}
		 		<li><a href="#{{ .Path }}">{{ . }}</a></li>
		 {{ end }}
	 </div>
	{{ end }}
</body>
</html>
`

type pageContent struct {
	Links  []*url.URL
	Assets []string
}

// HTML is a reporter that can output a html sitemap.
type HTML struct {
	sitemap  map[*url.URL]pageContent
	template *template.Template
	sync.Mutex
}

// NewHTML creates a new HTML reporter.
func NewHTML() *HTML {
	return &HTML{
		sitemap:  make(map[*url.URL]pageContent),
		template: template.Must(template.New("sitemap").Parse(sitemapHTML)),
	}
}

// Add links and assets to a URI.
func (r *HTML) Add(uri *url.URL, links []*url.URL, assets []string) {
	r.Lock()
	defer r.Unlock()
	_, ok := r.sitemap[uri]
	if ok {
		return
	}
	r.sitemap[uri] = pageContent{
		Links:  links,
		Assets: assets,
	}
}

// Report writes HTML to the given writer.
func (r *HTML) Report(w io.Writer) error {
	r.Lock()
	defer r.Unlock()
	return r.template.Execute(w, r.sitemap)
}
