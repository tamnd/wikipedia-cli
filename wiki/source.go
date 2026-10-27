package wiki

import (
	"context"
	"net/url"
	"strconv"
)

// Source is the wikitext of one revision of a page, with the ids that a corpus needs to cite it.
type Source struct {
	PageID    int    `json:"pageid"`
	NS        int    `json:"ns"`
	Title     string `json:"title"`
	RevID     int    `json:"revid"`
	ParentID  int    `json:"parentid"`
	Timestamp string `json:"timestamp"`
	SHA1      string `json:"sha1"`
	Size      int    `json:"size"`
	Content   string `json:"content"`
}

// sourceResponse is the Action API answer to a query for one revision with its content.
type sourceResponse struct {
	apiError
	Query struct {
		Pages []struct {
			PageID    int    `json:"pageid"`
			NS        int    `json:"ns"`
			Title     string `json:"title"`
			Missing   bool   `json:"missing"`
			Revisions []struct {
				RevID     int    `json:"revid"`
				ParentID  int    `json:"parentid"`
				Timestamp string `json:"timestamp"`
				SHA1      string `json:"sha1"`
				Size      int    `json:"size"`
				Slots     struct {
					Main struct {
						Content string `json:"content"`
					} `json:"main"`
				} `json:"slots"`
			} `json:"revisions"`
		} `json:"pages"`
	} `json:"query"`
}

func (r *sourceResponse) source() (*Source, error) {
	if err := r.err(); err != nil {
		return nil, err
	}
	if len(r.Query.Pages) == 0 || r.Query.Pages[0].Missing || len(r.Query.Pages[0].Revisions) == 0 {
		return nil, ErrNotFound
	}
	p := r.Query.Pages[0]
	rev := p.Revisions[0]
	return &Source{
		PageID: p.PageID, NS: p.NS, Title: p.Title,
		RevID: rev.RevID, ParentID: rev.ParentID, Timestamp: rev.Timestamp,
		SHA1: rev.SHA1, Size: rev.Size, Content: rev.Slots.Main.Content,
	}, nil
}

// PageSource returns the current revision of a page with its wikitext, its page id, its revision id, its timestamp and its sha1. A redirect is followed, so the title of the result can differ from the title that was asked for.
func (c *Client) PageSource(ctx context.Context, title string) (*Source, error) {
	v := c.actionParams()
	v.Set("action", "query")
	v.Set("prop", "revisions")
	v.Set("rvprop", "ids|timestamp|sha1|size|content")
	v.Set("rvslots", "main")
	v.Set("redirects", "1")
	v.Set("titles", title)
	return c.source(ctx, v)
}

// RevisionSource returns one revision by its id, with the same fields as PageSource.
func (c *Client) RevisionSource(ctx context.Context, revid int) (*Source, error) {
	v := c.actionParams()
	v.Set("action", "query")
	v.Set("prop", "revisions")
	v.Set("rvprop", "ids|timestamp|sha1|size|content")
	v.Set("rvslots", "main")
	v.Set("revids", strconv.Itoa(revid))
	return c.source(ctx, v)
}

func (c *Client) source(ctx context.Context, v url.Values) (*Source, error) {
	var resp sourceResponse
	if err := c.actionJSON(ctx, v, ttlContent, &resp); err != nil {
		return nil, err
	}
	return resp.source()
}
