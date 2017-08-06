package spider

import (
	"context"
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
)

type httpResponseError struct {
	statusCode int
}

func (e httpResponseError) Error() string {
	return "http response error: " + strconv.Itoa(e.statusCode)
}

// Requester is something that can make a request.
type Requester interface {
	Request(ctx context.Context, uri *url.URL) ([]byte, error)
}

//go:generate mockery -name Requester -case underscore

type client struct {
	client *http.Client
}

func (c client) Request(ctx context.Context, uri *url.URL) ([]byte, error) {
	if uri == nil {
		return nil, errors.New("must provide uri to request")
	}

	// Ignore this error as it's not possible to trigger with a valid URL and a constant method.
	req, _ := http.NewRequest(http.MethodGet, uri.String(), nil)
	req = req.WithContext(ctx)
	res, err := c.client.Do(req)
	if err != nil {
		return nil, httpResponseError{
			statusCode: res.StatusCode,
		}
	}

	if res.StatusCode != 200 {
		return nil, httpResponseError{
			statusCode: res.StatusCode,
		}
	}

	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	return data, nil
}
