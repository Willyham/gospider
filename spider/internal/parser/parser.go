package parser

import (
	"bytes"
	"io"

	"golang.org/x/net/html"
)

// HTML tags we care about
const (
	TagA      = "a"
	TagLink   = "link"
	TagImg    = "img"
	TagScript = "script"
)

// Attribute types we look for,
const (
	AttrHref = "href"
	AttrSrc  = "src"
)

// ParseResults encapsulates data we want out of the parser.
type ParseResults struct {
	Assets []string
	Links  []string
}

// Parser allows for different parser implementations.
// For example, it may be possible to get a speed increase at the expense of accuracy by using regex.
type Parser interface {
	Parse([]byte) (ParseResults, error)
}

// Func describes the parser function.
type Func func([]byte) (ParseResults, error)

// Parse adapts func to the Parser interface.
func (f Func) Parse(body []byte) (ParseResults, error) {
	return f(body)
}

// ByToken iterates over tokens in the response, pulling out links and assets.
var ByToken = Func(func(body []byte) (ParseResults, error) {
	tokenizer := html.NewTokenizer(bytes.NewReader(body))
	results := ParseResults{}
	for {
		tokenType := tokenizer.Next()
		switch tokenType {

		case html.ErrorToken:
			err := tokenizer.Err()
			if err == io.EOF {
				return results, nil
			}
			return results, err

		case html.StartTagToken:
			token := tokenizer.Token()

			// Capture links by looking for "a" tags
			if isTag(token, TagA) {
				href := filterAttrByName(token, AttrHref)
				if href == nil {
					continue
				}
				results.Links = append(results.Links, *href)
				continue
			}

			// Image and script assets both share the 'src' attribute.
			if isTag(token, TagImg) || isTag(token, TagScript) {
				src := filterAttrByName(token, AttrSrc)
				if src == nil {
					continue
				}
				results.Assets = append(results.Assets, *src)
			}

			if isTag(token, TagLink) {
				href := filterAttrByName(token, AttrHref)
				if href == nil {
					continue
				}
				results.Assets = append(results.Assets, *href)
				continue
			}

		}
	}
})

// isTag returns true if the token is a [tag], false otherwise.
func isTag(token html.Token, tag string) bool {
	return token.Data == tag
}

// filterAttrByName gets the attr value which matches name, nil otherwise.
func filterAttrByName(token html.Token, name string) *string {
	for _, attrs := range token.Attr {
		if attrs.Key == name {
			return &attrs.Val
		}
	}
	return nil
}
