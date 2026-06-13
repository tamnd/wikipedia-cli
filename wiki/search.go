package wiki

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
)

// SearchResult is one full-text search hit.
type SearchResult struct {
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Snippet     string `json:"snippet,omitempty"`
	Size        int    `json:"size,omitempty"`
	WordCount   int    `json:"wordcount,omitempty"`
	Timestamp   string `json:"timestamp,omitempty"`
	URL         string `json:"url"`
}

// Search runs a full-text search. It prefers the cross-wiki core endpoint for
// clean snippets and falls back to the Action API when the core endpoint is not
// available for the project.
func (c *Client) Search(ctx context.Context, query string, limit int) ([]SearchResult, error) {
	if limit <= 0 {
		limit = 10
	}
	if coreURL, ok := c.Site.CoreURL("search/page"); ok {
		v := url.Values{}
		v.Set("q", query)
		v.Set("limit", strconv.Itoa(min(limit, 100)))
		var resp struct {
			Pages []struct {
				Title       string `json:"title"`
				Excerpt     string `json:"excerpt"`
				Description string `json:"description"`
			} `json:"pages"`
		}
		if err := c.HTTP.GetJSON(ctx, coreURL+"?"+v.Encode(), ttlSearch, &resp); err == nil && len(resp.Pages) > 0 {
			out := make([]SearchResult, 0, len(resp.Pages))
			for _, p := range resp.Pages {
				out = append(out, SearchResult{
					Title:       p.Title,
					Description: p.Description,
					Snippet:     stripHTML(p.Excerpt),
					URL:         c.Site.PageURL(p.Title),
				})
			}
			return out, nil
		}
	}
	return c.searchAction(ctx, query, limit)
}

func (c *Client) searchAction(ctx context.Context, query string, limit int) ([]SearchResult, error) {
	v := c.actionParams()
	v.Set("action", "query")
	v.Set("list", "search")
	v.Set("srsearch", query)
	v.Set("srlimit", strconv.Itoa(min(limit, 500)))
	v.Set("srprop", "snippet|size|wordcount|timestamp")
	var resp struct {
		apiError
		Query struct {
			Search []struct {
				Title     string `json:"title"`
				Snippet   string `json:"snippet"`
				Size      int    `json:"size"`
				WordCount int    `json:"wordcount"`
				Timestamp string `json:"timestamp"`
			} `json:"search"`
		} `json:"query"`
	}
	if err := c.actionJSON(ctx, v, ttlSearch, &resp); err != nil {
		return nil, err
	}
	if err := resp.err(); err != nil {
		return nil, err
	}
	out := make([]SearchResult, 0, len(resp.Query.Search))
	for _, s := range resp.Query.Search {
		out = append(out, SearchResult{
			Title:     s.Title,
			Snippet:   stripHTML(s.Snippet),
			Size:      s.Size,
			WordCount: s.WordCount,
			Timestamp: s.Timestamp,
			URL:       c.Site.PageURL(s.Title),
		})
	}
	return out, nil
}

// Suggestion is one opensearch autocomplete entry.
type Suggestion struct {
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	URL         string `json:"url"`
}

// Suggest returns opensearch autocomplete results for a prefix.
func (c *Client) Suggest(ctx context.Context, prefix string, limit int) ([]Suggestion, error) {
	if limit <= 0 {
		limit = 10
	}
	v := url.Values{}
	v.Set("action", "opensearch")
	v.Set("search", prefix)
	v.Set("limit", strconv.Itoa(limit))
	v.Set("format", "json")
	// opensearch returns a positional array: [query, [titles], [descs], [urls]].
	var raw []json.RawMessage
	if err := c.HTTP.GetJSON(ctx, c.Site.APIURL(v), ttlSearch, &raw); err != nil {
		return nil, err
	}
	var titles, descs, urls []string
	if len(raw) > 1 {
		_ = json.Unmarshal(raw[1], &titles)
	}
	if len(raw) > 2 {
		_ = json.Unmarshal(raw[2], &descs)
	}
	if len(raw) > 3 {
		_ = json.Unmarshal(raw[3], &urls)
	}
	out := make([]Suggestion, 0, len(titles))
	for i, t := range titles {
		s := Suggestion{Title: t, URL: c.Site.PageURL(t)}
		if i < len(descs) {
			s.Description = descs[i]
		}
		if i < len(urls) && urls[i] != "" {
			s.URL = urls[i]
		}
		out = append(out, s)
	}
	return out, nil
}

// Random returns random page titles from the main namespace.
func (c *Client) Random(ctx context.Context, n, namespace int) ([]SearchResult, error) {
	if n <= 0 {
		n = 1
	}
	v := c.actionParams()
	v.Set("action", "query")
	v.Set("list", "random")
	v.Set("rnnamespace", strconv.Itoa(namespace))
	v.Set("rnlimit", strconv.Itoa(min(n, 500)))
	var resp struct {
		apiError
		Query struct {
			Random []struct {
				Title string `json:"title"`
			} `json:"random"`
		} `json:"query"`
	}
	// Random must never be cached.
	if err := c.HTTP.GetJSON(ctx, c.Site.APIURL(v), 0, &resp); err != nil {
		return nil, err
	}
	if err := resp.err(); err != nil {
		return nil, err
	}
	out := make([]SearchResult, 0, len(resp.Query.Random))
	for _, r := range resp.Query.Random {
		out = append(out, SearchResult{Title: r.Title, URL: c.Site.PageURL(r.Title)})
	}
	return out, nil
}

// Related returns the REST "related pages" suggestions for a title.
func (c *Client) Related(ctx context.Context, title string) ([]SearchResult, error) {
	var resp struct {
		Pages []struct {
			Title   string `json:"title"`
			Extract string `json:"extract"`
			Desc    string `json:"description"`
		} `json:"pages"`
	}
	u := c.Site.RestV1("page/related/" + titlePath(title))
	if err := c.HTTP.GetJSON(ctx, u, ttlSearch, &resp); err != nil {
		return nil, err
	}
	out := make([]SearchResult, 0, len(resp.Pages))
	for _, p := range resp.Pages {
		out = append(out, SearchResult{
			Title:       p.Title,
			Description: p.Desc,
			Snippet:     stripHTML(p.Extract),
			URL:         c.Site.PageURL(p.Title),
		})
	}
	return out, nil
}
