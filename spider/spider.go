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

func Start(args map[string]interface{}) error {
	conf, err := NewConfig(args)
	if err != nil {
		return err
	}

	spider := NewSpider(
		WithRoot(conf.RootURL),
		WithIgnoreRobots(conf.IgnoreRobots),
	)

	return spider.Run()
}

type Option func(*Spider)

func WithRoot(root *url.URL) Option {
	return func(s *Spider) {
		s.RootURL = root
	}
}

func WithIgnoreRobots(ignore bool) Option {
	return func(s *Spider) {
		s.IgnoreRobots = ignore
	}
}

func WithDepth(depth int) Option {
	return func(s *Spider) {
		s.MaxDepth = depth
	}
}

func WithClient(c *http.Client) Option {
	return func(s *Spider) {
		s.requester = client{
			client: c,
		}
	}
}

type Spider struct {
	IgnoreRobots     bool
	FollowSubdomains bool
	MaxDepth         int
	MaxTime          time.Duration
	Concurrency      int
	RootURL          *url.URL

	requester Requester
	logger    *zap.Logger
	robots    *robotstxt.RobotsData
}

// NewSpider creates a new spider with the given options.
func NewSpider(options ...Option) *Spider {
	logger, _ := zap.NewProduction()
	spider := &Spider{
		MaxDepth:     2,
		Concurrency:  4,
		MaxTime:      time.Minute,
		IgnoreRobots: false,

		requester: client{
			client: http.DefaultClient,
		},
		logger: logger,
	}
	for _, op := range options {
		op(spider)
	}
	return spider
}

func (s Spider) Run() error {
	wg := sync.WaitGroup{}
	queue := newURLQueue()

	if s.robots == nil && !s.IgnoreRobots {
		robots, err := s.readRobotsData(s.RootURL)
		if err != nil {
			return err
		}
		s.robots = robots
	}

	// TODO: const/config this
	pollInterval := time.Second

	// Add our root to the queue to start us off.
	queue.Append(s.RootURL)
	wg.Add(1)

	pool := concurrency.NewWorkerPool(s.logger, s.Concurrency, concurrency.WorkFunc(func() error {
		if !queue.HasItems() {
			time.Sleep(pollInterval)
			return nil
		}

		next := queue.Next()
		defer wg.Done()

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
		toAdd := s.filterURLsToAdd(results.Links, queue)
		for _, link := range toAdd {
			s.logger.Info("Found url to fetch", zap.String("url", link.String()))
			wg.Add(1)
			queue.Append(link)
		}

		return nil
	}))

	go pool.Start()

	// Wait for us to exhaust the queue, then shut down the pool and wait for it to fully drain.
	wg.Wait()
	pool.StopWait()
	return nil
}

var robotsTxt, _ = url.Parse("/robots.txt")

func (s Spider) readRobotsData(root *url.URL) (*robotstxt.RobotsData, error) {
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
func (s Spider) filterURLsToAdd(urls []string, seener Seener) []*url.URL {
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
// In the case that we want to follow subdomains, we check the suffix of the host,
// otherwise, we check the exact host.
func (s Spider) isExternalURL(input *url.URL) bool {
	if s.FollowSubdomains {
		return !strings.HasSuffix(input.Hostname(), s.RootURL.Hostname())
	}
	return input.Hostname() != s.RootURL.Hostname()
}
