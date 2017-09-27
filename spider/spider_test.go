package spider

import (
	"net/url"
	"testing"
	"time"

	"github.com/Willyham/gospider/spider/internal/concurrency"
	"github.com/Willyham/gospider/spider/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var willydURL, _ = url.Parse("http://willdemaine.co.uk")
var willydRobots, _ = url.Parse("http://willdemaine.co.uk/robots.txt")

func TestReadRobotsData(t *testing.T) {
	requester := &mocks.Requester{}
	requester.On("Request", mock.Anything, willydRobots).Return([]byte(`
		User-agent: *
		Disallow: /foo/
		Disallow: /bar/
	`), nil)

	s := New(
		WithRoot(willydURL),
		WithRequester(requester),
		WithUserAgent("agent"),
	)

	data, err := s.readRobotsData(willydURL)
	assert.NoError(t, err)
	assert.True(t, data.TestAgent("/", "Agent"))
	assert.False(t, data.TestAgent("/foo/a", "Agent"))
	assert.False(t, data.TestAgent("/bar/a", "Agent"))
	assert.True(t, data.TestAgent("/foo", "Agent"))
	assert.True(t, data.TestAgent("/asdf", "Agent"))
}

func TestNoRoot(t *testing.T) {
	assert.Panics(t, func() {
		New()
	})
}

func TestReadRobotsDataHTTPError(t *testing.T) {
	requester := &mocks.Requester{}
	requester.On("Request", mock.Anything, willydRobots).Return([]byte{}, httpResponseError{
		statusCode: 500,
	})

	s := New(
		WithRoot(willydURL),
		WithRequester(requester),
	)

	data, err := s.readRobotsData(willydURL)
	assert.NoError(t, err)
	assert.False(t, data.TestAgent("/", "Foo"))
}

func TestReadRobotsDataError(t *testing.T) {
	requester := &mocks.Requester{}
	requester.On("Request", mock.Anything, willydRobots).Return([]byte{}, assert.AnError)

	s := New(
		WithRoot(willydURL),
		WithRequester(requester),
	)

	_, err := s.readRobotsData(willydURL)
	assert.Error(t, err)
}

func TestReadRobotsDataMissing(t *testing.T) {
	requester := &mocks.Requester{}
	requester.On("Request", mock.Anything, willydRobots).Return([]byte{}, httpResponseError{
		statusCode: 404,
	})

	s := New(
		WithRoot(willydURL),
		WithRequester(requester),
	)

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
		WithTimeout(time.Minute),
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
		next := s.queue.Next()
		if next == nil {
			return nil
		}
		defer s.wg.Done()
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
		next := s.queue.Next()
		if next == nil {
			return nil
		}
		defer s.wg.Done()
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
