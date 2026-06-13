package wiki

import (
	"context"
	"net/url"
	"strconv"
	"time"
)

// Client bundles a resolved Site with the polite HTTP client and config. It is
// the handle every feature method hangs off. Build one with New.
type Client struct {
	Site Site
	HTTP *HTTPClient
	Cfg  Config
}

// New resolves cfg into a Client, sharing one cache and HTTP client.
func New(cfg Config, cache *Cache) (*Client, error) {
	site, err := cfg.Site()
	if err != nil {
		return nil, err
	}
	return &Client{Site: site, HTTP: NewHTTPClient(cfg, cache), Cfg: cfg}, nil
}

// actionParams seeds a url.Values for an Action API call with the common
// format/version/maxlag fields.
func (c *Client) actionParams() url.Values {
	v := url.Values{}
	v.Set("format", "json")
	v.Set("formatversion", "2")
	if c.Cfg.Maxlag > 0 {
		v.Set("maxlag", strconv.Itoa(c.Cfg.Maxlag))
	}
	return v
}

// actionJSON runs an Action API query with the given params and TTL, decoding
// the response into v.
func (c *Client) actionJSON(ctx context.Context, params url.Values, ttl time.Duration, v any) error {
	return c.HTTP.GetJSON(ctx, c.Site.APIURL(params), ttl, v)
}

// apiError mirrors the Action API's {"error": {...}} envelope.
type apiError struct {
	Error *struct {
		Code string `json:"code"`
		Info string `json:"info"`
	} `json:"error"`
}

func (e apiError) err() error {
	if e.Error == nil {
		return nil
	}
	return &APIError{Code: e.Error.Code, Info: e.Error.Info}
}

// APIError is a structured Action API error.
type APIError struct {
	Code string
	Info string
}

func (e *APIError) Error() string {
	if e.Info != "" {
		return e.Code + ": " + e.Info
	}
	return e.Code
}
