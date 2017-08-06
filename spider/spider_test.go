package spider

import (
	"net/url"
	"testing"

	"github.com/Willyham/gospider/spider/internal/concurrency"
	"github.com/Willyham/gospider/spider/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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
		{"local", "/foo", true, false},
		{"local no /", "foo", true, false},
		{"same host", "http://willdemaine.co.uk", false, false},
		{"path", "http://willdemaine.co.uk/foo", false, false},
		{"subdomain follow", "http://foo.willdemaine.co.uk", true, false},
		{"subdomain no follow", "http://foo.willdemaine.co.uk", false, true},
		{"external", "http://foo.bar.co.uk", false, true},
		{"external follow", "http://foo.bar.co.uk", true, true},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			s := New(WithRoot(testURL))
			s.FollowSubdomains = test.followSub

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

	s := New(WithRoot(root))
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

	s := New(WithRequester(requester))

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

	s := New(WithRequester(requester))

	data, err := s.readRobotsData(willydURL)
	assert.NoError(t, err)
	assert.False(t, data.TestAgent("/", "Foo"))
}

func TestReadRobotsDataError(t *testing.T) {
	requester := &mocks.Requester{}
	requester.On("Request", mock.Anything, willydRobots).Return([]byte{}, assert.AnError)

	s := New(WithRequester(requester))

	_, err := s.readRobotsData(willydURL)
	assert.Error(t, err)
}

func TestReadRobotsDataMissing(t *testing.T) {
	requester := &mocks.Requester{}
	requester.On("Request", mock.Anything, willydRobots).Return([]byte{}, httpResponseError{
		statusCode: 404,
	})

	s := New(WithRequester(requester))

	data, err := s.readRobotsData(willydURL)
	assert.NoError(t, err)
	assert.True(t, data.TestAgent("/", "Foo"))
}

func TestWorkerNoItems(t *testing.T) {
	s := New(WithRoot(willydURL))
	s.wg.Add(1)
	err := s.work()
	assert.NoError(t, err)
}

func TestWorker(t *testing.T) {
	requester := &mocks.Requester{}
	requester.On("Request", mock.Anything, willydURL).Return([]byte(`
		<a href="/foo/bar"></a>
	`), nil)

	s := New(
		WithRoot(willydURL),
		WithRequester(requester),
		WithConcurrency(1),
		WithIgnoreRobots(false),
	)
	s.queue.Append(willydURL)

	s.wg.Add(1)
	err := s.work()
	assert.NoError(t, err)

	assert.Len(t, s.queue.urls, 1)
	assert.Equal(t, "http://willdemaine.co.uk/foo/bar", s.queue.urls[0].String())
}

func TestWorkerRequestError(t *testing.T) {
	requester := &mocks.Requester{}
	requester.On("Request", mock.Anything, willydURL).Return(nil, httpResponseError{
		statusCode: 500,
	})

	s := New(WithRoot(willydURL), WithRequester(requester))
	s.queue.Append(willydURL)

	s.wg.Add(1)
	err := s.work()
	assert.Error(t, err)
}

func TestRun(t *testing.T) {
	requester := &mocks.Requester{}
	requester.On("Request", mock.Anything, willydURL).Return([]byte("foo"), nil)

	s := New(
		WithRoot(willydURL),
		WithConcurrency(1),
		WithRequester(requester),
		WithIgnoreRobots(true), // So we don't request robots.txt
	)

	s.worker = concurrency.WorkFunc(func() error {
		if !s.queue.HasItems() {
			return nil
		}
		defer s.wg.Done()
		_ = s.queue.Next()
		return nil
	})
	err := s.Run()
	assert.NoError(t, err)
}

func TestRunRobots(t *testing.T) {
	requester := &mocks.Requester{}
	requester.On("Request", mock.Anything, willydRobots).Return([]byte("foo"), nil)
	requester.On("Request", mock.Anything, willydURL).Return([]byte("foo"), nil)

	s := New(
		WithRoot(willydURL),
		WithConcurrency(1),
		WithRequester(requester),
	)

	s.worker = concurrency.WorkFunc(func() error {
		if !s.queue.HasItems() {
			return nil
		}
		defer s.wg.Done()
		_ = s.queue.Next()
		return nil
	})
	err := s.Run()
	assert.NoError(t, err)
}

func TestRunRobotsError(t *testing.T) {
	requester := &mocks.Requester{}
	requester.On("Request", mock.Anything, willydRobots).Return(nil, assert.AnError)

	s := New(
		WithRoot(willydURL),
		WithConcurrency(1),
		WithRequester(requester),
	)
	err := s.Run()
	assert.Error(t, err)
}
