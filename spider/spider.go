package spider

import (
	"context"
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/Willyham/gospider/spider/internal/concurrency"
	"github.com/Willyham/gospider/spider/internal/parser"
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

func WithFollowSubdomains(follow bool) Option {
	return func(s *Spider) {
		s.FollowSubdomains = follow
	}
}

func WithDepth(depth int) Option {
	return func(s *Spider) {
		s.MaxDepth = depth
	}
}

func WithClient(client *http.Client) Option {
	return func(s *Spider) {
		s.client = client
	}
}

type Spider struct {
	IgnoreRobots     bool
	FollowSubdomains bool
	MaxDepth         int
	MaxTime          time.Duration
	Concurrency      int
	RootURL          *url.URL

	client *http.Client
	logger *zap.Logger
}

// NewSpider creates a new spider with the given options.
func NewSpider(options ...Option) *Spider {
	logger, _ := zap.NewProduction()
	spider := &Spider{
		MaxDepth:         2,
		Concurrency:      4,
		MaxTime:          time.Minute,
		FollowSubdomains: false,
		IgnoreRobots:     false,

		client: http.DefaultClient,
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

	// TODO: const/config this
	pollInterval := time.Second

	// Add our root to the queue to start us off.
	queue.Append(s.RootURL)
	wg.Add(1)

	// Create parent context which sets an overall timeout.
	ctx, cancel := context.WithTimeout(context.Background(), s.MaxTime)
	defer cancel()

	pool := concurrency.NewWorkerPool(s.logger, s.Concurrency, concurrency.WorkFunc(func() error {
		if !queue.HasItems() {
			time.Sleep(pollInterval)
			return nil
		}

		next := queue.Next()
		defer wg.Done()
		body, err := s.request(ctx, next)
		if err != nil {
			// TODO: Maybe make err retryable.
			return err
		}

		results, err := parser.ByToken(body)
		if err != nil {
			return err
		}

		// Add all of the links to follow to the queue.
		for _, link := range results.Links {
			uri, err := url.Parse(link)
			if err != nil {
				s.logger.Info("Skipping invalid url", zap.String("url", link))
				continue
			}

			if !s.isExternalURL(uri) && !queue.Seen(uri) {
				s.logger.Info("Found url to fetch", zap.String("url", link))
				wg.Add(1)
				queue.Append(uri)
			}
		}

		return nil
	}))

	go pool.Start()

	// Wait for us to exhaust the queue, then shut down the pool and wait for it to fully drain.
	wg.Wait()
	pool.StopWait()
	return nil
}

func (s Spider) request(ctx context.Context, uri *url.URL) ([]byte, error) {
	if uri == nil {
		return nil, errors.New("must provide uri to request")
	}

	// Ignore this error as it's not possible to trigger with a valid URL and a constant method.
	req, _ := http.NewRequest(http.MethodGet, uri.String(), nil)

	req = req.WithContext(ctx)
	res, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != 200 {
		s.logger.Info("error fetching", zap.String("url", uri.String()))
		return nil, errors.New("unexpected status code")
	}

	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	return data, nil
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
