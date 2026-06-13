package wiki

import (
	"context"
	"strconv"
	"strings"
)

// Revision is one entry in a page's revision history. It keeps everything the
// revisions query returns under the widened rvprop set: the editor's user id,
// the content sha1, the html-parsed edit summary, and the change tags as a
// list rather than a flattened string.
type Revision struct {
	RevID         int      `json:"revid"`
	ParentID      int      `json:"parentid"`
	Timestamp     string   `json:"timestamp"`
	User          string   `json:"user"`
	UserID        int      `json:"userid,omitempty"`
	Size          int      `json:"size"`
	Comment       string   `json:"comment"`
	ParsedComment string   `json:"parsedcomment,omitempty"`
	Sha1          string   `json:"sha1,omitempty"`
	Minor         bool     `json:"minor"`
	Tags          []string `json:"tags,omitempty"`
}

// TagsLine joins the change tags for the table view.
func (r Revision) TagsLine() string { return strings.Join(r.Tags, ",") }

// Revisions returns the revision history of a page, newest first.
func (c *Client) Revisions(ctx context.Context, title string, limit int, user string) ([]Revision, error) {
	v := c.actionParams()
	v.Set("action", "query")
	v.Set("prop", "revisions")
	v.Set("rvlimit", limitParam(limit))
	v.Set("rvprop", "ids|timestamp|user|userid|size|sha1|comment|parsedcomment|flags|tags")
	if user != "" {
		v.Set("rvuser", user)
	}
	v.Set("redirects", "1")
	v.Set("titles", title)
	var resp struct {
		apiError
		Query struct {
			Pages []struct {
				Missing   bool       `json:"missing"`
				Revisions []Revision `json:"revisions"`
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
	return resp.Query.Pages[0].Revisions, nil
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

// Diff compares two revisions. from/to may be numeric revision ids; the toRev
// may also be "prev"/"cur"/"next" relative to from. It returns the diff lines.
func (c *Client) Diff(ctx context.Context, fromRev int, toRev string) ([]DiffLine, error) {
	v := c.actionParams()
	v.Set("action", "compare")
	v.Set("fromrev", strconv.Itoa(fromRev))
	if n, err := strconv.Atoi(toRev); err == nil {
		v.Set("torev", strconv.Itoa(n))
	} else {
		v.Set("torelative", toRev) // prev|next|cur
	}
	v.Set("prop", "diff")
	var resp struct {
		apiError
		Compare struct {
			Body string `json:"body"`
		} `json:"compare"`
	}
	if err := c.actionJSON(ctx, v, ttlHistory, &resp); err != nil {
		return nil, err
	}
	if err := resp.err(); err != nil {
		return nil, err
	}
	return parseDiffTable(resp.Compare.Body), nil
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
