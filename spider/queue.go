package spider

import (
	"net/url"
	"sync"
)

type Seener interface {
	Seen(*url.URL) bool
}

type urlQueue struct {
	urls []*url.URL
	seen map[string]bool
	sync.RWMutex
}

func newURLQueue() *urlQueue {
	return &urlQueue{
		seen: make(map[string]bool),
	}
}

func (q *urlQueue) HasItems() bool {
	q.RLock()
	defer q.RUnlock()
	return len(q.urls) > 0
}

func (q *urlQueue) Seen(item *url.URL) bool {
	q.RLock()
	_, seen := q.seen[item.String()]
	q.RUnlock()
	return seen
}

func (q *urlQueue) Next() *url.URL {
	q.Lock()
	var next *url.URL
	next, q.urls = q.urls[len(q.urls)-1], q.urls[:len(q.urls)-1]
	q.Unlock()
	return next
}

func (q *urlQueue) Append(item *url.URL) {
	q.Lock()
	q.urls = append(q.urls, item)
	q.seen[item.String()] = true
	q.Unlock()
}
