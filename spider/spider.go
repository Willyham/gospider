package spider

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/Willyham/gospider/spider/internal/concurrency"
	"github.com/Willyham/gospider/spider/internal/parser"
	"github.com/temoto/robotstxt"
)

const (
	workerPollInterval = time.Millisecond * 100
)

var robotsTxt, _ = url.Parse("/robots.txt")

// Option is a function that configures the spider.
type Option func(*Spider)

// WithRoot sets the rootURL for the spider.
func WithRoot(root *url.URL) Option {
	return func(s *Spider) {
		s.rootURL = root
	}
}

// WithIgnoreRobots sets whether or not the spider should ignore
// the robots.txt data.
func WithIgnoreRobots(ignore bool) Option {
	return func(s *Spider) {
		s.ignoreRobots = ignore
	}
}

// WithConcurrency sets how many workers will request urls concurrently.
func WithConcurrency(con int) Option {
	return func(s *Spider) {
		s.concurrency = con
	}
}

// WithRequester sets the requester that the spider should use to make requests.
func WithRequester(req Requester) Option {
	return func(s *Spider) {
		s.requester = req
	}
}

// Spider can run requests against a URI until it sees every internal page on that site
// at least once. It can be configued with Option arguments which override defaults.
type Spider struct {
	ignoreRobots     bool
	FollowSubdomains bool
	concurrency      int
	rootURL          *url.URL

	requester Requester
	worker    concurrency.Worker
	logger    *zap.Logger
	robots    *robotstxt.RobotsData
	queue     *urlQueue
	wg        sync.WaitGroup
}

// New creates a new spider with the given options.
func New(options ...Option) *Spider {
	logger, _ := zap.NewProduction()
	spider := &Spider{
		concurrency:  4,
		ignoreRobots: false,
		requester: client{
			logger: logger,
			client: http.DefaultClient,
		},
		logger: logger,
		queue:  newURLQueue(),
	}
	// Default to spider.work, but allow this to be overridden for testing
	// by having worker as a field on the Spider struct.
	spider.worker = concurrency.WorkFunc((*spider).work)
	for _, op := range options {
		op(spider)
	}
	return spider
}

// Run the spider. Start at the root and follow all valid URLs, building a map
// of the site.
func (s *Spider) Run() error {
	if s.robots == nil && !s.ignoreRobots {
		robots, err := s.readRobotsData(s.rootURL)
		if err != nil {
			return err
		}
		s.robots = robots
	}

	// Add our root to the queue to start us off.
	s.queue.Append(s.rootURL)
	s.wg.Add(1)

	pool := concurrency.NewWorkerPool(s.logger, s.concurrency, s.worker)
	go pool.Start()

	// Wait until we're done with all work, the drain the pool too.
	s.wg.Wait()
	pool.StopWait()
	return nil
}

type workerFactory func(queue *urlQueue, wg *sync.WaitGroup) concurrency.WorkFunc

// createWorker creates the worker used in the worker pool. Each worker will poll the URL queue
// for items. If a URL is found, it will collect the links/assets for the URL and report them.
func (s *Spider) work() error {
	if !s.queue.HasItems() {
		time.Sleep(workerPollInterval)
		return nil
	}

	next := s.queue.Next()
	defer s.wg.Done()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	body, err := s.requester.Request(ctx, next)
	if err != nil {
		// TODO: Maybe make err retryable.
		return err
	}

	results, err := parser.ByToken(body)
	if err != nil {
		return err
	}

	// Add all of the links to follow to the queue.
	s.logger.Info("Found links", zap.Int("links", len(results.Links)))
	toAdd := s.filterURLsToAdd(results.Links, s.queue)
	for _, link := range toAdd {
		fullPath := s.rootURL.ResolveReference(link)
		s.logger.Info("Enqueing link to fetch", zap.String("url", fullPath.String()))
		s.queue.Append(s.rootURL.ResolveReference(fullPath))
		s.wg.Add(1)
	}

	return nil
}

// readRobotsData makes a request to the root + /robots.txt and parses the data.
// In the event of a 4XX, we assume crawling is allowed. In the event of a 5XX,
// we assume it is disallowed.
func (s *Spider) readRobotsData(root *url.URL) (*robotstxt.RobotsData, error) {
	robotsURL := root.ResolveReference(robotsTxt)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	res, err := s.requester.Request(ctx, robotsURL)
	if err != nil {
		httpErr, ok := err.(httpResponseError)
		if ok {
			return robotstxt.FromStatusAndBytes(httpErr.statusCode, res)
		}
		return nil, err
	}
	return robotstxt.FromBytes(res)
}

// filterURLsToAdd determines which URLs should be added to the queue for fetching.
func (s *Spider) filterURLsToAdd(urls []string, seener Seener) []*url.URL {
	results := make([]*url.URL, 0, len(urls))
	for _, link := range urls {
		uri, err := url.Parse(link)
		if err != nil {
			s.logger.Info("Skipping invalid url", zap.String("url", link))
			continue
		}

		if !s.isExternalURL(uri) && !seener.Seen(uri) {
			results = append(results, uri)
		}
	}
	return results
}

// isExternalURL determines if the URL should be counted as 'external'.
// If the URL has no scheme and host, it's a relative one.
// In the case that we want to follow subdomains, we check the suffix of the host,
// otherwise, we check the exact host.
func (s *Spider) isExternalURL(input *url.URL) bool {
	if input.Scheme == "" && input.Hostname() == "" {
		return false
	}
	if s.FollowSubdomains {
		return !strings.HasSuffix(input.Hostname(), s.rootURL.Hostname())
	}
	return input.Hostname() != s.rootURL.Hostname()
}
