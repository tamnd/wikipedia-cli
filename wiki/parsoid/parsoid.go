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
	"encoding/json"
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

// DefaultAPI is the Action API of the English Wikipedia. FetchParse uses it.
const DefaultAPI = "https://en.wikipedia.org/w/api.php"

// MinInterval is the shortest interval between two requests: 200 requests a
// minute.
const MinInterval = 300 * time.Millisecond

// maxTries is the number of tries of a request that gets HTTP 429 or 503.
const maxTries = 5

// Fetcher fetches Parsoid HTML. The zero value is not ready: set UserAgent.
type Fetcher struct {
	Base      string // the REST API, DefaultBase when empty
	API       string // the Action API, DefaultAPI when empty
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

// ErrTimeout is the error of HTTP 504 from the REST API. The REST API renders
// an old revision when a client asks for it, and the gateway stops the
// render of a large article after some seconds. FetchParse can then get the
// same HTML from the Action API.
var ErrTimeout = errors.New("the REST API did not render the revision in time")

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
	u := base + "/page/html/" + url.PathEscape(strings.ReplaceAll(title, " ", "_"))
	if rev != 0 {
		u += "/" + strconv.Itoa(rev)
	}
	body, code, err := f.get(ctx, u)
	switch {
	case err != nil:
		return nil, err
	case code == http.StatusOK:
		return body, nil
	case code == http.StatusNotFound:
		return nil, fmt.Errorf("%s revision %d: %w", title, rev, ErrNotFound)
	case code == http.StatusGatewayTimeout:
		return nil, fmt.Errorf("%s revision %d: %w", title, rev, ErrTimeout)
	default:
		return nil, fmt.Errorf("%s revision %d: HTTP %d", title, rev, code)
	}
}

// FetchParse gives the Parsoid HTML of one revision from the Action API
// (action=parse with parsoid=1). The Action API gives the body of the
// document, so FetchParse puts it in a document whose html element has the
// about attribute of the revision, as the REST API does. The document has no
// mw:revisionSHA1, so Read gives no SHA1 for it.
func (f *Fetcher) FetchParse(ctx context.Context, rev int) ([]byte, error) {
	if f.UserAgent == "" {
		return nil, errors.New("parsoid: the fetcher has no User-Agent")
	}
	api := f.API
	if api == "" {
		api = DefaultAPI
	}
	q := url.Values{"action": {"parse"}, "oldid": {strconv.Itoa(rev)}, "parsoid": {"1"}, "prop": {"text|revid"}, "format": {"json"}, "formatversion": {"2"}}
	body, code, err := f.get(ctx, api+"?"+q.Encode())
	if err != nil {
		return nil, err
	}
	if code != http.StatusOK {
		return nil, fmt.Errorf("revision %d: the Action API gives HTTP %d", rev, code)
	}
	var r struct {
		Parse struct {
			RevID int    `json:"revid"`
			Text  string `json:"text"`
		} `json:"parse"`
		Error *struct {
			Code string `json:"code"`
			Info string `json:"info"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &r); err != nil {
		return nil, fmt.Errorf("revision %d: %w", rev, err)
	}
	switch {
	case r.Error != nil && (r.Error.Code == "nosuchrevid" || r.Error.Code == "missingtitle"):
		return nil, fmt.Errorf("revision %d: %w", rev, ErrNotFound)
	case r.Error != nil:
		return nil, fmt.Errorf("revision %d: the Action API gives %s: %s", rev, r.Error.Code, r.Error.Info)
	case r.Parse.RevID != rev || r.Parse.Text == "":
		return nil, fmt.Errorf("revision %d: the Action API gives the revision %d", rev, r.Parse.RevID)
	}
	doc := fmt.Sprintf("<!DOCTYPE html>\n<html about=\"//en.wikipedia.org/wiki/Special:Redirect/revision/%d\"><head><meta charset=\"utf-8\"/></head><body>", rev)
	return []byte(doc + r.Parse.Text + "</body></html>\n"), nil
}

// get makes one GET request at the rate of the fetcher, and it tries again
// after HTTP 429 or 503. It gives the body and the status code.
func (f *Fetcher) get(ctx context.Context, u string) ([]byte, int, error) {
	client := f.Client
	if client == nil {
		client = &http.Client{Timeout: 60 * time.Second}
	}
	for try := 1; ; try++ {
		if err := f.wait(ctx); err != nil {
			return nil, 0, err
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		if err != nil {
			return nil, 0, err
		}
		req.Header.Set("User-Agent", f.UserAgent)
		resp, err := client.Do(req)
		if err != nil {
			return nil, 0, err
		}
		body, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if err != nil {
			return nil, 0, err
		}
		if (resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == http.StatusServiceUnavailable) && try < maxTries {
			if err := sleep(ctx, retryAfter(resp.Header.Get("Retry-After"), try)); err != nil {
				return nil, 0, err
			}
			continue
		}
		return body, resp.StatusCode, nil
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
