package wiki

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// PageviewPoint is one entry in a pageview time series. Besides the date and
// view count, it keeps the request context each item carries, so a row records
// exactly which project, article, access method, agent and granularity it came
// from and the raw timestamp.
type PageviewPoint struct {
	Date        string `json:"date"`
	Views       int    `json:"views"`
	Project     string `json:"project,omitempty"`
	Article     string `json:"article,omitempty"`
	Access      string `json:"access,omitempty"`
	Agent       string `json:"agent,omitempty"`
	Granularity string `json:"granularity,omitempty"`
	Timestamp   string `json:"timestamp,omitempty"`
}

// Pageviews returns the daily or monthly pageview series for a title between two
// dates. granularity is "daily" or "monthly"; access/agent default to "all".
func (c *Client) Pageviews(ctx context.Context, title, granularity, access, agent string, from, to time.Time) ([]PageviewPoint, error) {
	if granularity == "" {
		granularity = "daily"
	}
	if access == "" {
		access = "all-access"
	}
	if agent == "" {
		agent = "all-agents"
	}
	project := c.Site.Host
	encTitle := strings.ReplaceAll(strings.TrimSpace(title), " ", "_")
	u := fmt.Sprintf("%s/pageviews/per-article/%s/%s/%s/%s/%s/%s/%s",
		MetricsBase, project, access, agent, urlPathEscape(encTitle),
		granularity, from.Format("20060102"), to.Format("20060102"))
	var resp struct {
		Items []struct {
			Project     string `json:"project"`
			Article     string `json:"article"`
			Granularity string `json:"granularity"`
			Timestamp   string `json:"timestamp"`
			Access      string `json:"access"`
			Agent       string `json:"agent"`
			Views       int    `json:"views"`
		} `json:"items"`
	}
	if err := c.HTTP.GetJSON(ctx, u, ttlPageviews, &resp); err != nil {
		return nil, err
	}
	out := make([]PageviewPoint, 0, len(resp.Items))
	for _, it := range resp.Items {
		out = append(out, PageviewPoint{
			Date:        formatViewDate(it.Timestamp, granularity),
			Views:       it.Views,
			Project:     it.Project,
			Article:     it.Article,
			Access:      it.Access,
			Agent:       it.Agent,
			Granularity: it.Granularity,
			Timestamp:   it.Timestamp,
		})
	}
	return out, nil
}

// TopArticle is one entry in a most-viewed list. It keeps the URL-path article
// key and the project, access method and date the list was drawn from, so a row
// is self-describing.
type TopArticle struct {
	Rank    int    `json:"rank"`
	Title   string `json:"title"`
	Article string `json:"article,omitempty"`
	Views   int    `json:"views"`
	Project string `json:"project,omitempty"`
	Access  string `json:"access,omitempty"`
	Year    string `json:"year,omitempty"`
	Month   string `json:"month,omitempty"`
	Day     string `json:"day,omitempty"`
	URL     string `json:"url"`
}

// Top returns the most-viewed articles for a day, or for a month when day == 0.
func (c *Client) Top(ctx context.Context, year, month, day, limit int) ([]TopArticle, error) {
	dayStr := fmt.Sprintf("%02d", day)
	if day == 0 {
		dayStr = "all-days"
	}
	u := fmt.Sprintf("%s/pageviews/top/%s/all-access/%04d/%02d/%s",
		MetricsBase, c.Site.Host, year, month, dayStr)
	var resp struct {
		Items []struct {
			Project  string `json:"project"`
			Access   string `json:"access"`
			Year     string `json:"year"`
			Month    string `json:"month"`
			Day      string `json:"day"`
			Articles []struct {
				Article string `json:"article"`
				Views   int    `json:"views"`
				Rank    int    `json:"rank"`
			} `json:"articles"`
		} `json:"items"`
	}
	if err := c.HTTP.GetJSON(ctx, u, ttlPageviews, &resp); err != nil {
		return nil, err
	}
	var out []TopArticle
	if len(resp.Items) == 0 {
		return out, nil
	}
	it := resp.Items[0]
	for _, a := range it.Articles {
		title := strings.ReplaceAll(a.Article, "_", " ")
		out = append(out, TopArticle{
			Rank: a.Rank, Title: title, Article: a.Article, Views: a.Views,
			Project: it.Project, Access: it.Access, Year: it.Year, Month: it.Month, Day: it.Day,
			URL: c.Site.PageURL(a.Article),
		})
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out, nil
}

func formatViewDate(ts, granularity string) string {
	// timestamps are YYYYMMDDHH.
	if len(ts) >= 8 {
		if granularity == "monthly" {
			return ts[0:4] + "-" + ts[4:6]
		}
		return ts[0:4] + "-" + ts[4:6] + "-" + ts[6:8]
	}
	return ts
}
