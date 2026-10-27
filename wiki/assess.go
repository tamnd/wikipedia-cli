package wiki

import (
	"context"
	"net/url"
	"slices"
	"strings"
)

// ProjectPage is one page that a WikiProject tags, with its assessment.
type ProjectPage struct {
	Project    string `json:"project"`
	PageID     int64  `json:"pageid"`
	NS         int    `json:"ns"`
	Title      string `json:"title"`
	Class      string `json:"class,omitempty"`
	Importance string `json:"importance,omitempty"`
}

// ProjectPages calls fn for each page that one of the projects tags. It
// follows the continuation of list=projectpages to the end, or to limit pages
// when limit is positive. The API gives the subject page of each tagged talk
// page. The API has no namespace filter, so ProjectPages keeps only the pages
// in namespace ns. A negative ns keeps all namespaces.
func (c *Client) ProjectPages(ctx context.Context, projects []string, ns, limit int, fn func(ProjectPage) error) error {
	return projectPages(ctx, func(v url.Values, out any) error {
		return c.actionJSON(ctx, v, ttlContent, out)
	}, c.actionParams(), projects, ns, limit, fn)
}

type projectPagesResp struct {
	apiError
	Continue map[string]string `json:"continue"`
	Query    struct {
		Projects map[string][]struct {
			PageID     int64  `json:"pageid"`
			NS         int    `json:"ns"`
			Title      string `json:"title"`
			Assessment struct {
				Class      string `json:"class"`
				Importance string `json:"importance"`
			} `json:"assessment"`
		} `json:"projects"`
	} `json:"query"`
}

func projectPages(ctx context.Context, get func(url.Values, any) error, v url.Values, projects []string, ns, limit int, fn func(ProjectPage) error) error {
	v.Set("action", "query")
	v.Set("list", "projectpages")
	v.Set("wppprojects", strings.Join(projects, "|"))
	v.Set("wppassessments", "1")
	v.Set("wpplimit", "max")
	n := 0
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		var resp projectPagesResp
		if err := get(v, &resp); err != nil {
			return err
		}
		if err := resp.err(); err != nil {
			return err
		}
		names := make([]string, 0, len(resp.Query.Projects))
		for p := range resp.Query.Projects {
			names = append(names, p)
		}
		slices.Sort(names)
		for _, p := range names {
			for _, r := range resp.Query.Projects[p] {
				if ns >= 0 && r.NS != ns {
					continue
				}
				pp := ProjectPage{Project: p, PageID: r.PageID, NS: r.NS, Title: r.Title,
					Class: r.Assessment.Class, Importance: r.Assessment.Importance}
				if err := fn(pp); err != nil {
					return err
				}
				n++
				if limit > 0 && n >= limit {
					return nil
				}
			}
		}
		if len(resp.Continue) == 0 {
			return nil
		}
		for k, val := range resp.Continue {
			v.Set(k, val)
		}
	}
}
