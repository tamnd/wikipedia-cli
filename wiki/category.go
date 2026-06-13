package wiki

import (
	"context"
	"encoding/json"
	"strings"
)

// Category is one category a page belongs to. It keeps the sort key the page
// is filed under, the timestamp it was added, and whether the category is
// hidden, so the structured output reflects the full categories query.
type Category struct {
	Title         string `json:"title"`
	NS            int    `json:"ns"`
	SortKey       string `json:"sortkey,omitempty"`
	SortKeyPrefix string `json:"sortkeyprefix,omitempty"`
	Timestamp     string `json:"timestamp,omitempty"`
	Hidden        bool   `json:"hidden,omitempty"`
	URL           string `json:"url"`
}

// Categories returns the categories a page belongs to.
func (c *Client) Categories(ctx context.Context, title string, limit int) ([]Category, error) {
	v := c.actionParams()
	v.Set("action", "query")
	v.Set("prop", "categories")
	// Fetch every category (cllimit=max) with its sort key, timestamp and hidden
	// flag, then truncate to the caller's limit ourselves. Hidden categories are
	// kept rather than filtered server-side, so the output is the full list and
	// the hidden ones are marked rather than dropped.
	v.Set("cllimit", "max")
	v.Set("clprop", "sortkey|timestamp|hidden")
	v.Set("redirects", "1")
	v.Set("titles", title)
	var resp struct {
		apiError
		Query struct {
			Pages []struct {
				Categories []struct {
					NS            int             `json:"ns"`
					Title         string          `json:"title"`
					SortKey       string          `json:"sortkey"`
					SortKeyPrefix string          `json:"sortkeyprefix"`
					Timestamp     string          `json:"timestamp"`
					Hidden        json.RawMessage `json:"hidden"`
				} `json:"categories"`
			} `json:"pages"`
		} `json:"query"`
	}
	if err := c.actionJSON(ctx, v, ttlContent, &resp); err != nil {
		return nil, err
	}
	if err := resp.err(); err != nil {
		return nil, err
	}
	var out []Category
	for _, p := range resp.Query.Pages {
		for _, cat := range p.Categories {
			out = append(out, Category{
				Title: cat.Title, NS: cat.NS,
				SortKey: cat.SortKey, SortKeyPrefix: cat.SortKeyPrefix,
				Timestamp: cat.Timestamp, Hidden: cat.Hidden != nil,
				URL: c.Site.PageURL(cat.Title),
			})
			if limit > 0 && len(out) >= limit {
				return out, nil
			}
		}
	}
	return out, nil
}

// CategoryMember is one member of a category.
type CategoryMember struct {
	Pageid    int    `json:"pageid,omitempty"`
	Title     string `json:"title"`
	NS        int    `json:"ns"`
	Type      string `json:"type"`
	Timestamp string `json:"timestamp,omitempty"`
	URL       string `json:"url"`
}

// CategoryMembers lists the members of a category. memberType is one of
// "page", "subcat", "file", or "" for all. The "Category:" prefix is added if
// missing.
func (c *Client) CategoryMembers(ctx context.Context, name, memberType string, limit int) ([]CategoryMember, error) {
	if !strings.Contains(name, ":") {
		name = "Category:" + name
	}
	v := c.actionParams()
	v.Set("action", "query")
	v.Set("list", "categorymembers")
	v.Set("cmtitle", name)
	v.Set("cmlimit", limitParam(limit))
	v.Set("cmprop", "ids|title|type|timestamp")
	if memberType != "" {
		v.Set("cmtype", memberType)
	}
	var resp struct {
		apiError
		Query struct {
			CategoryMembers []struct {
				Pageid    int    `json:"pageid"`
				NS        int    `json:"ns"`
				Title     string `json:"title"`
				Type      string `json:"type"`
				Timestamp string `json:"timestamp"`
			} `json:"categorymembers"`
		} `json:"query"`
	}
	if err := c.actionJSON(ctx, v, ttlContent, &resp); err != nil {
		return nil, err
	}
	if err := resp.err(); err != nil {
		return nil, err
	}
	out := make([]CategoryMember, 0, len(resp.Query.CategoryMembers))
	for _, m := range resp.Query.CategoryMembers {
		out = append(out, CategoryMember{
			Pageid: m.Pageid, Title: m.Title, NS: m.NS, Type: m.Type, Timestamp: m.Timestamp,
			URL: c.Site.PageURL(m.Title),
		})
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out, nil
}
