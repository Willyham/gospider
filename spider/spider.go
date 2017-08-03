package spider

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
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
	pool   *concurrency.WorkerPool
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
	spider.pool = concurrency.NewWorkerPool(logger, worker{}, spider.Concurrency)
	return spider
}

func (s Spider) Run() error {
	ctx, cancel := context.WithTimeout(context.Background(), s.MaxTime)
	defer cancel()

	first, err := s.request(ctx, s.RootURL)
	if err != nil {
		return err
	}

	out, err := parser.ByToken(first)
	fmt.Println(out, err)
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
