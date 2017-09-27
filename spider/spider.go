package spider

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/Willyham/gospider/spider/internal/concurrency"
	"github.com/Willyham/gospider/spider/internal/parser"
	"github.com/Willyham/gospider/spider/reporter"
	"github.com/temoto/robotstxt"
)

const (
	workerPollInterval = time.Millisecond * 100
	userAgent          = "gospider/v1.0"
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

// WithTimeout sets the request timeout.
func WithTimeout(dur time.Duration) Option {
	return func(s *Spider) {
		s.requestTimeout = dur
	}
}

// WithUserAgent overwrites the default user agent.
func WithUserAgent(agent string) Option {
	return func(s *Spider) {
		s.userAgent = agent
	}
}

// Spider can run requests against a URI until it sees every internal page on that site
// at least once. It can be configued with Option arguments which override defaults.
type Spider struct {
	ignoreRobots     bool
	followSubdomains bool
	concurrency      int
	rootURL          *url.URL
	requestTimeout   time.Duration
	userAgent        string

	requester Requester
	reporter  reporter.Interface
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
		concurrency:    1,
		ignoreRobots:   false,
		requestTimeout: time.Second * 5,
		userAgent:      userAgent,
		requester: client{
			logger: logger,
			client: http.DefaultClient,
		},
		logger:   logger,
		queue:    newURLQueue(),
		reporter: reporter.NewHTML(),
	}
	// Default to spider.work, but allow this to be overridden for testing
	// by having worker as a field on the Spider struct.
	spider.worker = concurrency.WorkFunc((*spider).work)
	for _, op := range options {
		op(spider)
	}

	if spider.rootURL == nil {
		panic("must supply a root URL")
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

// Report writes the report to the writer.
func (s *Spider) Report(w io.Writer) error {
	return s.reporter.Report(w)
}

// work is the function used by the worker in the pool. Each worker will poll the URL queue
// for items. If a URL is found, it will collect the links/assets for the URL and report them.
func (s *Spider) work() error {
	next := s.queue.Next()
	if next == nil {
		time.Sleep(workerPollInterval)
		return nil
	}
	s.logger.Info("Items left in queue", zap.Int("number", len(s.queue.urls)))
	defer s.wg.Done()

	ctx, cancel := context.WithTimeout(context.Background(), s.requestTimeout)
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

	// TODO: Move these predicates out of the work function
	onlyInternal := createIsInternalPredicate(s.rootURL, s.followSubdomains)
	asAbsolute := createAbsoluteTransformer(s.rootURL)
	notSeen := createNotSeenPredicate(s.queue)

	absoluteLinks := mapURLs(asAbsolute, results.Links)
	internalLinks := filter(onlyInternal, absoluteLinks)

	// Report all links before we filter out the ones we need to fetch.
	s.reporter.Add(next, internalLinks, results.Assets)
	s.logger.Info("Found links", zap.Int("links", len(internalLinks)))

	toAdd := filter(notSeen, internalLinks)
	for _, link := range toAdd {
		s.logger.Info("Enqueing link to fetch", zap.String("url", link.String()))
		s.queue.Append(link)
		s.wg.Add(1)
	}

	return nil
}

// readRobotsData makes a request to the root + /robots.txt and parses the data.
// In the event of a 4XX, we assume crawling is allowed. In the event of a 5XX,
// we assume it is disallowed.
func (s *Spider) readRobotsData(root *url.URL) (*robotstxt.RobotsData, error) {
	robotsURL := root.ResolveReference(robotsTxt)
	ctx, cancel := context.WithTimeout(context.Background(), s.requestTimeout)
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
