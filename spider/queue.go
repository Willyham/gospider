package spider

import (
	"fmt"
	"net/url"
	"sync"
)

// urlQueue is a structure which maintains a queue of URLs.
// it also records a list of all URLs seen and implements the Seener interface.
type urlQueue struct {
	urls []*url.URL
	seen map[string]bool
	sync.RWMutex
}

var _ Seener = new(urlQueue)

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
	fmt.Println("Check seen for ", item.String(), seen)
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
