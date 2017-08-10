# gospider

`gospider` is a concurrent web spider. By default, it respects robots.txt entries.

## Usage

    glide install

`gospider` is designed to be used either from the CLI or in code.

### CLI

`gospider` can be run from the CLI by running

    gospider start -r "http://foo.bar/" > out.html

By default, gospider writes an HTML sitemap to stdout.

Use `gospider --help` for more options.

### Code

the `spider.New` function follows the functional options pattern. The only parameter which is required
is the root URL - all others will be defaulted to sensible values if not supplied.

```
uri, _ := url.Parse("http://foo.bar/")

spider := spider.New(
  spider.WithRoot(uri),
  spider.WithConcurrency(5),
  spider.WithTimeout(time.Second * 2),
)

err = spider.Run()
if err != nil {
  log.Fatal("error running spider: ", err)
}
return spider.Report(os.Stdout)
```

#### Modularity

`gospider` ships with a simple HTML reporter and uses the default HTTP client to make requests. However, any requester
or reporter can be used by supplying a struct which implements the `Requester` or `Reporter` interface. For example,
to make requests through a proxy you could do:

```
type proxyRequester struct {
  client *http.Client
}

func (r *proxyRequester) Request(ctx context.Context, uri *url.URL) ([]byte, error) {
  res, err := r.client.Get(uri.String())
  // handle err, read body, etc.
  return body, nil
}

s := spider.New(
  WithRoot(...),
  WithRequester(&proxyRequester{
    client: &http.Client{
      Transport: &http.Transport{Proxy: http.ProxyURL(...)}
    }
  })
)

```

## Concurrency

`gospider` uses a worker pool concurrency model. As URLs are found they are added to a queue. Each
worker (controlled with the concurrency parameter) will poll the queue for work. Once the queue is empty,
the worker pool is drained and the spider will stop.
