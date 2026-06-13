package wiki

import (
	"context"
	"strconv"
)

// Link is a wiki link (internal) or external link target.
type Link struct {
	Title string `json:"title,omitempty"`
	NS    int    `json:"ns"`
	URL   string `json:"url"`
}

// Links returns the internal wiki links on a page. When namespace >= 0 only that
// namespace is returned.
func (c *Client) Links(ctx context.Context, title string, namespace, limit int) ([]Link, error) {
	v := c.actionParams()
	v.Set("action", "query")
	v.Set("prop", "links")
	v.Set("pllimit", limitParam(limit))
	if namespace >= 0 {
		v.Set("plnamespace", strconv.Itoa(namespace))
	}
	v.Set("redirects", "1")
	v.Set("titles", title)
	var resp struct {
		apiError
		Query struct {
			Pages []struct {
				Links []struct {
					NS    int    `json:"ns"`
					Title string `json:"title"`
				} `json:"links"`
			} `json:"pages"`
		} `json:"query"`
	}
	if err := c.actionJSON(ctx, v, ttlContent, &resp); err != nil {
		return nil, err
	}
	if err := resp.err(); err != nil {
		return nil, err
	}
	var out []Link
	for _, p := range resp.Query.Pages {
		for _, l := range p.Links {
			out = append(out, Link{Title: l.Title, NS: l.NS, URL: c.Site.PageURL(l.Title)})
			if limit > 0 && len(out) >= limit {
				return out, nil
			}
		}
	}
	return out, nil
}

// ExternalLinks returns the external URLs referenced by a page.
func (c *Client) ExternalLinks(ctx context.Context, title string, limit int) ([]Link, error) {
	v := c.actionParams()
	v.Set("action", "query")
	v.Set("prop", "extlinks")
	v.Set("ellimit", limitParam(limit))
	v.Set("redirects", "1")
	v.Set("titles", title)
	var resp struct {
		apiError
		Query struct {
			Pages []struct {
				ExtLinks []struct {
					URL string `json:"url"`
				} `json:"extlinks"`
			} `json:"pages"`
		} `json:"query"`
	}
	if err := c.actionJSON(ctx, v, ttlContent, &resp); err != nil {
		return nil, err
	}
	if err := resp.err(); err != nil {
		return nil, err
	}
	var out []Link
	for _, p := range resp.Query.Pages {
		for _, l := range p.ExtLinks {
			out = append(out, Link{URL: l.URL})
			if limit > 0 && len(out) >= limit {
				return out, nil
			}
		}
	}
	return out, nil
}

// Backlinks returns pages that link to the given title ("what links here").
func (c *Client) Backlinks(ctx context.Context, title string, namespace, limit int) ([]Link, error) {
	v := c.actionParams()
	v.Set("action", "query")
	v.Set("list", "backlinks")
	v.Set("bltitle", title)
	v.Set("bllimit", limitParam(limit))
	if namespace >= 0 {
		v.Set("blnamespace", strconv.Itoa(namespace))
	}
	var resp struct {
		apiError
		Query struct {
			Backlinks []struct {
				NS    int    `json:"ns"`
				Title string `json:"title"`
			} `json:"backlinks"`
		} `json:"query"`
	}
	if err := c.actionJSON(ctx, v, ttlContent, &resp); err != nil {
		return nil, err
	}
	if err := resp.err(); err != nil {
		return nil, err
	}
	out := make([]Link, 0, len(resp.Query.Backlinks))
	for _, b := range resp.Query.Backlinks {
		out = append(out, Link{Title: b.Title, NS: b.NS, URL: c.Site.PageURL(b.Title)})
	}
	return out, nil
}

// LangLink is an interlanguage link.
type LangLink struct {
	Lang    string `json:"lang"`
	Title   string `json:"title"`
	Autonym string `json:"autonym,omitempty"`
	URL     string `json:"url"`
}

// LangLinks returns the same article in other languages.
func (c *Client) LangLinks(ctx context.Context, title string, limit int) ([]LangLink, error) {
	v := c.actionParams()
	v.Set("action", "query")
	v.Set("prop", "langlinks")
	v.Set("lllimit", limitParam(limit))
	v.Set("llprop", "url|autonym")
	v.Set("redirects", "1")
	v.Set("titles", title)
	var resp struct {
		apiError
		Query struct {
			Pages []struct {
				LangLinks []struct {
					Lang    string `json:"lang"`
					Title   string `json:"title"`
					Autonym string `json:"autonym"`
					URL     string `json:"url"`
				} `json:"langlinks"`
			} `json:"pages"`
		} `json:"query"`
	}
	if err := c.actionJSON(ctx, v, ttlContent, &resp); err != nil {
		return nil, err
	}
	if err := resp.err(); err != nil {
		return nil, err
	}
	var out []LangLink
	for _, p := range resp.Query.Pages {
		for _, l := range p.LangLinks {
			out = append(out, LangLink{Lang: l.Lang, Title: l.Title, Autonym: l.Autonym, URL: l.URL})
		}
	}
	return out, nil
}

// limitParam renders a result limit for the Action API, where "max" asks for the
// server maximum and a positive number is capped at 500.
func limitParam(limit int) string {
	if limit <= 0 || limit > 500 {
		return "max"
	}
	return strconv.Itoa(limit)
}
