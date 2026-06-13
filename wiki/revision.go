package wiki

import (
	"context"
	"strconv"
	"strings"
)

// Revision is one entry in a page's revision history.
type Revision struct {
	RevID     int    `json:"revid"`
	ParentID  int    `json:"parentid"`
	Timestamp string `json:"timestamp"`
	User      string `json:"user"`
	Size      int    `json:"size"`
	Comment   string `json:"comment"`
	Minor     bool   `json:"minor"`
	Tags      string `json:"tags,omitempty"`
}

// Revisions returns the revision history of a page, newest first.
func (c *Client) Revisions(ctx context.Context, title string, limit int, user string) ([]Revision, error) {
	v := c.actionParams()
	v.Set("action", "query")
	v.Set("prop", "revisions")
	v.Set("rvlimit", limitParam(limit))
	v.Set("rvprop", "ids|timestamp|user|size|comment|flags|tags")
	if user != "" {
		v.Set("rvuser", user)
	}
	v.Set("redirects", "1")
	v.Set("titles", title)
	var resp struct {
		apiError
		Query struct {
			Pages []struct {
				Missing   bool `json:"missing"`
				Revisions []struct {
					RevID     int      `json:"revid"`
					ParentID  int      `json:"parentid"`
					Timestamp string   `json:"timestamp"`
					User      string   `json:"user"`
					Size      int      `json:"size"`
					Comment   string   `json:"comment"`
					Minor     bool     `json:"minor"`
					Tags      []string `json:"tags"`
				} `json:"revisions"`
			} `json:"pages"`
		} `json:"query"`
	}
	if err := c.actionJSON(ctx, v, ttlHistory, &resp); err != nil {
		return nil, err
	}
	if err := resp.err(); err != nil {
		return nil, err
	}
	if len(resp.Query.Pages) == 0 || resp.Query.Pages[0].Missing {
		return nil, ErrNotFound
	}
	var out []Revision
	for _, r := range resp.Query.Pages[0].Revisions {
		out = append(out, Revision{
			RevID: r.RevID, ParentID: r.ParentID, Timestamp: r.Timestamp,
			User: r.User, Size: r.Size, Comment: r.Comment, Minor: r.Minor,
			Tags: strings.Join(r.Tags, ","),
		})
	}
	return out, nil
}

// RevisionContent returns the wikitext of a specific revision id.
func (c *Client) RevisionContent(ctx context.Context, revid int) (string, error) {
	v := c.actionParams()
	v.Set("action", "query")
	v.Set("prop", "revisions")
	v.Set("revids", strconv.Itoa(revid))
	v.Set("rvprop", "content")
	v.Set("rvslots", "main")
	var resp struct {
		apiError
		Query struct {
			Pages []struct {
				Revisions []struct {
					Slots struct {
						Main struct {
							Content string `json:"content"`
						} `json:"main"`
					} `json:"slots"`
				} `json:"revisions"`
			} `json:"pages"`
		} `json:"query"`
	}
	if err := c.actionJSON(ctx, v, ttlContent, &resp); err != nil {
		return "", err
	}
	if err := resp.err(); err != nil {
		return "", err
	}
	if len(resp.Query.Pages) == 0 || len(resp.Query.Pages[0].Revisions) == 0 {
		return "", ErrNotFound
	}
	return resp.Query.Pages[0].Revisions[0].Slots.Main.Content, nil
}

// DiffLine is one line of a unified diff.
type DiffLine struct {
	Op   string `json:"op"` // "add", "del", "context"
	Text string `json:"text"`
}

// Diff is the result of comparing two revisions. It keeps both endpoints the
// compare API reports (their page ids, revision ids, namespaces and titles) and
// the raw HTML diff body, alongside the parsed unified-diff lines used for the
// table and text views. With the body and both endpoints preserved, the JSON is
// a faithful copy of the action=compare response.
type Diff struct {
	FromID    int        `json:"fromid,omitempty"`
	FromRevID int        `json:"fromrevid,omitempty"`
	FromNS    int        `json:"fromns,omitempty"`
	FromTitle string     `json:"fromtitle,omitempty"`
	ToID      int        `json:"toid,omitempty"`
	ToRevID   int        `json:"torevid,omitempty"`
	ToNS      int        `json:"tons,omitempty"`
	ToTitle   string     `json:"totitle,omitempty"`
	Body      string     `json:"body,omitempty"`
	Lines     []DiffLine `json:"lines"`
}

// Diff compares two revisions. from/to may be numeric revision ids; the toRev
// may also be "prev"/"cur"/"next" relative to from.
func (c *Client) Diff(ctx context.Context, fromRev int, toRev string) (*Diff, error) {
	v := c.actionParams()
	v.Set("action", "compare")
	v.Set("fromrev", strconv.Itoa(fromRev))
	if n, err := strconv.Atoi(toRev); err == nil {
		v.Set("torev", strconv.Itoa(n))
	} else {
		v.Set("torelative", toRev) // prev|next|cur
	}
	v.Set("prop", "ids|title|diff")
	var resp struct {
		apiError
		Compare struct {
			FromID    int    `json:"fromid"`
			FromRevID int    `json:"fromrevid"`
			FromNS    int    `json:"fromns"`
			FromTitle string `json:"fromtitle"`
			ToID      int    `json:"toid"`
			ToRevID   int    `json:"torevid"`
			ToNS      int    `json:"tons"`
			ToTitle   string `json:"totitle"`
			Body      string `json:"body"`
		} `json:"compare"`
	}
	if err := c.actionJSON(ctx, v, ttlHistory, &resp); err != nil {
		return nil, err
	}
	if err := resp.err(); err != nil {
		return nil, err
	}
	cm := resp.Compare
	return &Diff{
		FromID: cm.FromID, FromRevID: cm.FromRevID, FromNS: cm.FromNS, FromTitle: cm.FromTitle,
		ToID: cm.ToID, ToRevID: cm.ToRevID, ToNS: cm.ToNS, ToTitle: cm.ToTitle,
		Body:  cm.Body,
		Lines: parseDiffTable(cm.Body),
	}, nil
}

// parseDiffTable turns MediaWiki's HTML diff table into unified diff lines.
func parseDiffTable(body string) []DiffLine {
	if strings.TrimSpace(body) == "" {
		return nil
	}
	var lines []DiffLine
	// Scan the diff table's rows by their MediaWiki cell classes.
	for chunk := range strings.SplitSeq(body, "<tr>") {
		switch {
		case strings.Contains(chunk, "diff-addedline"):
			lines = append(lines, DiffLine{Op: "add", Text: stripHTML(chunk)})
		case strings.Contains(chunk, "diff-deletedline"):
			lines = append(lines, DiffLine{Op: "del", Text: stripHTML(chunk)})
		case strings.Contains(chunk, "diff-context"):
			lines = append(lines, DiffLine{Op: "context", Text: stripHTML(chunk)})
		}
	}
	return lines
}
