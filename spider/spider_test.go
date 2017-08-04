package spider

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsExternalURL(t *testing.T) {
	testURL, err := url.Parse("http://willdemaine.co.uk")
	require.NoError(t, err)

	cases := []struct {
		name      string
		uri       string
		followSub bool
		expected  bool
	}{
		{"same host", "http://willdemaine.co.uk", false, false},
		{"path", "http://willdemaine.co.uk/foo", false, false},
		{"subdomain follow", "http://foo.willdemaine.co.uk", true, false},
		{"subdomain no follow", "http://foo.willdemaine.co.uk", false, true},
		{"external", "http://foo.bar.co.uk", false, true},
		{"external follow", "http://foo.bar.co.uk", true, true},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			s := Spider{
				RootURL:          testURL,
				FollowSubdomains: test.followSub,
			}

			parsed, err := url.Parse(test.uri)
			require.NoError(t, err)
			assert.Equal(t, test.expected, s.isExternalURL(parsed))
		})
	}
}

func TestRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "Foo")
	}))
	defer server.Close()

	uri, err := url.Parse(server.URL)
	require.NoError(t, err)

	spider := NewSpider(WithClient(http.DefaultClient))
	res, err := spider.request(context.Background(), uri)
	assert.NoError(t, err)
	assert.Equal(t, []byte("Foo"), res)
}

func TestRequestNoURI(t *testing.T) {
	spider := NewSpider()
	_, err := spider.request(context.Background(), nil)
	assert.Error(t, err)
}

func TestRequestError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer server.Close()

	uri, err := url.Parse(server.URL)
	require.NoError(t, err)

	spider := NewSpider(WithClient(http.DefaultClient))
	_, err = spider.request(context.Background(), uri)
	assert.Error(t, err)
}

func TestFilterURLsToAdd(t *testing.T) {
	root, err := url.Parse("http://willdemaine.co.uk")
	require.NoError(t, err)
	path1, err := url.Parse("http://willdemaine.co.uk/foo")
	require.NoError(t, err)

	cases := []struct {
		name     string
		input    []string
		expected []*url.URL
		seener   Seener
	}{
		{"empty", []string{}, []*url.URL{}, &urlQueue{}},
		{"invalid", []string{":"}, []*url.URL{}, &urlQueue{}},
		{"valid not seen", []string{"http://willdemaine.co.uk/foo"}, []*url.URL{path1}, &urlQueue{}},
		{"valid and seen", []string{"http://willdemaine.co.uk/foo"}, []*url.URL{}, &urlQueue{
			seen: map[string]bool{
				"http://willdemaine.co.uk/foo": true,
			},
		}},
	}

	s := NewSpider(WithRoot(root))
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expected, s.filterURLsToAdd(test.input, test.seener))
		})
	}
}
