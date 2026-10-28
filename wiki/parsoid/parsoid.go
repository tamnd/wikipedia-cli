// Package parsoid fetches the Parsoid HTML of a revision from the REST API
// of Wikipedia, and reads from it what the wikitext alone does not give: the
// templates with their parameters in data-mw, the TeX of each formula, the
// sections, the anchors and the references.
//
// The fetch stays at or below 200 requests a minute, which is the limit
// that Wikimedia asks of the clients of the REST API. A caller can make the
// interval longer, but not shorter.
package parsoid

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

// DefaultBase is the REST API of the English Wikipedia.
const DefaultBase = "https://en.wikipedia.org/api/rest_v1"

// MinInterval is the shortest interval between two requests: 200 requests a
// minute.
const MinInterval = 300 * time.Millisecond

// maxTries is the number of tries of a request that gets HTTP 429 or 503.
const maxTries = 5

// Fetcher fetches Parsoid HTML. The zero value is not ready: set UserAgent.
type Fetcher struct {
	Base      string // the REST API, DefaultBase when empty
	UserAgent string // names the tool, its version and a contact
	Client    *http.Client
	Interval  time.Duration // the interval between two requests, at least MinInterval

	mu       sync.Mutex
	next     time.Time
	requests int
	start    time.Time
}

// ErrNotFound is the error of a revision that the REST API does not have.
var ErrNotFound = errors.New("the REST API has no such revision")

// Fetch gives the Parsoid HTML of one revision of a page. A revision of 0
// gives the current revision.
func (f *Fetcher) Fetch(ctx context.Context, title string, rev int) ([]byte, error) {
	if f.UserAgent == "" {
		return nil, errors.New("parsoid: the fetcher has no User-Agent")
	}
	base := f.Base
	if base == "" {
		base = DefaultBase
	}
	client := f.Client
	if client == nil {
		client = &http.Client{Timeout: 60 * time.Second}
	}
	u := base + "/page/html/" + url.PathEscape(strings.ReplaceAll(title, " ", "_"))
	if rev != 0 {
		u += "/" + strconv.Itoa(rev)
	}
	for try := 1; ; try++ {
		if err := f.wait(ctx); err != nil {
			return nil, err
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", f.UserAgent)
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		body, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if err != nil {
			return nil, err
		}
		switch {
		case resp.StatusCode == http.StatusOK:
			return body, nil
		case resp.StatusCode == http.StatusNotFound:
			return nil, fmt.Errorf("%s revision %d: %w", title, rev, ErrNotFound)
		case (resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == http.StatusServiceUnavailable) && try < maxTries:
			if err := sleep(ctx, retryAfter(resp.Header.Get("Retry-After"), try)); err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("%s revision %d: HTTP %d", title, rev, resp.StatusCode)
		}
	}
}

// Rate gives the number of requests and the requests a minute since the
// first request. A run record shows it.
func (f *Fetcher) Rate() (requests int, perMinute float64) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.requests == 0 {
		return 0, 0
	}
	d := time.Since(f.start)
	if d < time.Minute {
		d = time.Minute
	}
	return f.requests, float64(f.requests) / d.Minutes()
}

// wait blocks until the next request may start.
func (f *Fetcher) wait(ctx context.Context) error {
	gap := max(f.Interval, MinInterval)
	f.mu.Lock()
	now := time.Now()
	if f.requests == 0 {
		f.start = now
	}
	f.requests++
	at := f.next
	if at.Before(now) {
		at = now
	}
	f.next = at.Add(gap)
	f.mu.Unlock()
	return sleep(ctx, time.Until(at))
}

func sleep(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return ctx.Err()
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// retryAfter gives the wait of a Retry-After header in seconds, or a
// backoff that doubles with each try.
func retryAfter(h string, try int) time.Duration {
	if s, err := strconv.Atoi(strings.TrimSpace(h)); err == nil && s >= 0 {
		return time.Duration(s) * time.Second
	}
	return time.Duration(1<<try) * time.Second
}
