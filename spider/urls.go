package spider

import (
	"net/url"
	"strings"
)

// Seener is something which can check if a URL has ever been seen.
type Seener interface {
	Seen(*url.URL) bool
}

type urlPredicate func(*url.URL) bool

// Seen adapts a urlPredicate to the Seener interface
func (p urlPredicate) Seen(input *url.URL) bool {
	return p(input)
}

// filter a list of urls based on the predicate.
func filter(predicate urlPredicate, urls []*url.URL) []*url.URL {
	output := make([]*url.URL, 0, len(urls))
	for _, url := range urls {
		if predicate(url) {
			output = append(output, url)
		}
	}
	return output
}

// createIsInternalPredicate creates a predicate which tests if the url is internal.
// If we're following subdomains, we check based on the suffix of the host, otherwise
// we exact match on the Hostname.
func createIsInternalPredicate(root *url.URL, followSubdomains bool) urlPredicate {
	return func(input *url.URL) bool {
		if followSubdomains {
			return strings.HasSuffix(input.Hostname(), root.Hostname())
		}
		return input.Hostname() == root.Hostname()
	}
}

// createNotSeenPredicate creates a predicate which is true when a URL has not been
// seen before, according to the given seener.
func createNotSeenPredicate(seener Seener) urlPredicate {
	return func(input *url.URL) bool {
		return !seener.Seen(input)
	}
}

type urlTransform func(*url.URL) *url.URL

// mapURLs transforms a collection of urls with the transform.
func mapURLs(f urlTransform, urls []*url.URL) []*url.URL {
	out := make([]*url.URL, len(urls))
	for i, url := range urls {
		out[i] = f(url)
	}
	return out
}

// createAbsoluteTransformer creates a transform which resolves the url
// relative to the given root.
func createAbsoluteTransformer(root *url.URL) urlTransform {
	return func(input *url.URL) *url.URL {
		return root.ResolveReference(input)
	}
}
