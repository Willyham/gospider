package spider

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/Willyham/gospider/spider/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type errTripper struct{}

func (c errTripper) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, assert.AnError
}

var blackholeClient = &http.Client{
	Transport: errTripper{},
}

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

var willydURL, _ = url.Parse("http://willdemaine.co.uk")
var willydRobots, _ = url.Parse("http://willdemaine.co.uk/robots.txt")

func TestReadRobotsData(t *testing.T) {
	requester := &mocks.Requester{}
	requester.On("Request", mock.Anything, willydRobots).Return([]byte(`
		User-agent: *
		Disallow: /foo/
		Disallow: /bar/
	`), nil)

	s := NewSpider(WithClient(blackholeClient))
	s.requester = requester

	data, err := s.readRobotsData(willydURL)
	assert.NoError(t, err)
	assert.True(t, data.TestAgent("/", "Agent"))
	assert.False(t, data.TestAgent("/foo/a", "Agent"))
	assert.False(t, data.TestAgent("/bar/a", "Agent"))
	assert.True(t, data.TestAgent("/foo", "Agent"))
	assert.True(t, data.TestAgent("/asdf", "Agent"))
}

func TestReadRobotsDataHTTPError(t *testing.T) {
	requester := &mocks.Requester{}
	requester.On("Request", mock.Anything, willydRobots).Return([]byte{}, httpResponseError{
		statusCode: 500,
	})

	s := NewSpider(WithClient(blackholeClient))
	s.requester = requester

	data, err := s.readRobotsData(willydURL)
	assert.NoError(t, err)
	assert.False(t, data.TestAgent("/", "Foo"))
}

func TestReadRobotsDataError(t *testing.T) {
	requester := &mocks.Requester{}
	requester.On("Request", mock.Anything, willydRobots).Return([]byte{}, assert.AnError)

	s := NewSpider(WithClient(blackholeClient))
	s.requester = requester

	_, err := s.readRobotsData(willydURL)
	assert.Error(t, err)
}

func TestReadRobotsDataMissing(t *testing.T) {
	requester := &mocks.Requester{}
	requester.On("Request", mock.Anything, willydRobots).Return([]byte{}, httpResponseError{
		statusCode: 404,
	})

	s := NewSpider()
	s.requester = requester

	data, err := s.readRobotsData(willydURL)
	assert.NoError(t, err)
	assert.True(t, data.TestAgent("/", "Foo"))
}
