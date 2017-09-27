package spider

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"go.uber.org/zap"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "Foo")
		assert.Equal(t, "foo", r.Header.Get("User-Agent"))
	}))
	defer server.Close()

	uri, err := url.Parse(server.URL)
	require.NoError(t, err)

	c := client{
		client:    http.DefaultClient,
		logger:    zap.NewNop(),
		userAgent: "foo",
	}
	res, err := c.Request(context.Background(), uri)
	assert.NoError(t, err)
	assert.Equal(t, []byte("Foo"), res)
}

func TestRequestNoURI(t *testing.T) {
	c := client{
		client: http.DefaultClient,
		logger: zap.NewNop(),
	}
	_, err := c.Request(context.Background(), nil)
	assert.Error(t, err)
}

func TestRequestError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer server.Close()

	uri, err := url.Parse(server.URL)
	require.NoError(t, err)

	c := client{
		client: http.DefaultClient,
		logger: zap.NewNop(),
	}
	_, err = c.Request(context.Background(), uri)
	assert.Error(t, err)
	httpErr, ok := err.(httpResponseError)
	assert.True(t, ok)
	assert.Equal(t, 500, httpErr.statusCode)
	assert.Equal(t, "http response error: 500", httpErr.Error())
}
