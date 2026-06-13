package wiki

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// HTTPClient is a polite, retrying HTTP client for the Wikimedia hosts. It
// rate-limits requests, retries on 429/5xx with backoff that honors any
// Retry-After header, and caches GET-JSON responses on disk with a per-call TTL.
type HTTPClient struct {
	c         *http.Client
	download  *http.Client // no timeout, for large file bodies
	cache     *Cache
	retries   int
	delay     time.Duration
	userAgent string

	mu   sync.Mutex
	next time.Time // earliest time the next request may start
}

// NewHTTPClient builds an HTTPClient from cfg. The cache may be nil.
func NewHTTPClient(cfg Config, cache *Cache) *HTTPClient {
	ua := cfg.UserAgent
	if ua == "" {
		ua = UserAgent
	}
	return &HTTPClient{
		c:         &http.Client{Timeout: cfg.Timeout},
		download:  &http.Client{},
		cache:     cache,
		retries:   max(cfg.Retries, 0),
		delay:     cfg.Delay,
		userAgent: ua,
	}
}

// throttle blocks until the configured minimum inter-request delay has elapsed.
func (h *HTTPClient) throttle(ctx context.Context) error {
	if h.delay <= 0 {
		return nil
	}
	h.mu.Lock()
	now := time.Now()
	wait := time.Until(h.next)
	if h.next.Before(now) {
		h.next = now.Add(h.delay)
	} else {
		h.next = h.next.Add(h.delay)
	}
	h.mu.Unlock()
	if wait <= 0 {
		return nil
	}
	t := time.NewTimer(wait)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// GetJSON fetches url and unmarshals the JSON body into v. When ttl > 0 and a
// cache is configured the response is served from and stored in the cache.
func (h *HTTPClient) GetJSON(ctx context.Context, url string, ttl time.Duration, v any) error {
	if h.cache != nil && ttl > 0 {
		if data, ok := h.cache.Get(url, ttl); ok {
			return json.Unmarshal(data, v)
		}
	}
	body, err := h.GetBytes(ctx, url)
	if err != nil {
		return err
	}
	if h.cache != nil && ttl > 0 {
		h.cache.Put(url, body)
	}
	return json.Unmarshal(body, v)
}

// GetBytes fetches url and returns the whole body, retrying transient failures.
func (h *HTTPClient) GetBytes(ctx context.Context, url string) ([]byte, error) {
	resp, err := h.do(ctx, h.c, url, "", "")
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, &HTTPError{Status: resp.StatusCode, URL: url, Body: snippet(body)}
	}
	return body, nil
}

// GetText fetches url with an explicit Accept header and returns the raw body.
func (h *HTTPClient) GetText(ctx context.Context, url, accept string) ([]byte, error) {
	resp, err := h.do(ctx, h.c, url, "", accept)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, &HTTPError{Status: resp.StatusCode, URL: url, Body: snippet(body)}
	}
	return body, nil
}

// Open returns the response body for streaming downloads (no client timeout; the
// caller closes it and relies on ctx for cancellation). An optional Range header
// is sent when rangeHdr is non-empty (for resuming a download).
func (h *HTTPClient) Open(ctx context.Context, url, rangeHdr string) (*http.Response, error) {
	resp, err := h.do(ctx, h.download, url, rangeHdr, "")
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		_ = resp.Body.Close()
		return nil, &HTTPError{Status: resp.StatusCode, URL: url, Body: snippet(body)}
	}
	return resp, nil
}

func (h *HTTPClient) do(ctx context.Context, client *http.Client, url, rangeHdr, accept string) (*http.Response, error) {
	var last error
	for i := 0; i <= h.retries; i++ {
		if i > 0 {
			if err := sleep(ctx, h.delay*time.Duration(i*i+1)); err != nil {
				return nil, err
			}
		}
		if err := h.throttle(ctx); err != nil {
			return nil, err
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", h.userAgent)
		req.Header.Set("Api-User-Agent", h.userAgent)
		if accept != "" {
			req.Header.Set("Accept", accept)
		}
		if rangeHdr != "" {
			req.Header.Set("Range", rangeHdr)
		}
		resp, err := client.Do(req)
		if err != nil {
			last = err
			continue
		}
		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
			wait := retryAfter(resp.Header.Get("Retry-After"))
			_ = resp.Body.Close()
			last = &HTTPError{Status: resp.StatusCode, URL: url}
			if i < h.retries {
				if err := sleep(ctx, wait); err != nil {
					return nil, err
				}
			}
			continue
		}
		return resp, nil
	}
	if last == nil {
		last = fmt.Errorf("request to %s failed", url)
	}
	return nil, last
}

func sleep(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
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

// retryAfter parses a Retry-After header (delta-seconds) into a duration, with a
// zero fallback so the caller's backoff still applies.
func retryAfter(v string) time.Duration {
	if v == "" {
		return 0
	}
	if secs, err := strconv.Atoi(v); err == nil && secs > 0 {
		return time.Duration(secs) * time.Second
	}
	return 0
}

func snippet(b []byte) string {
	const n = 200
	s := string(b)
	if len(s) > n {
		s = s[:n]
	}
	return s
}

// HTTPError is returned for non-2xx responses.
type HTTPError struct {
	Status int
	URL    string
	Body   string
}

func (e *HTTPError) Error() string {
	if e.Body != "" {
		return fmt.Sprintf("HTTP %d from %s: %s", e.Status, e.URL, e.Body)
	}
	return fmt.Sprintf("HTTP %d from %s", e.Status, e.URL)
}

// NotFound reports whether err is a 404.
func NotFound(err error) bool {
	var he *HTTPError
	if e, ok := err.(*HTTPError); ok {
		he = e
	}
	return he != nil && he.Status == http.StatusNotFound
}
