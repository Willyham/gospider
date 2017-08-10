package reporter

import (
	"io"
	"net/url"
)

// Interface describes a reporter.
type Interface interface {
	Add(uri *url.URL, links []*url.URL, assets []string)
	Report(io.Writer) error
}
