package wiki

import (
	"context"
)

// SiteStats holds general site info and statistics for a wiki.
type SiteStats struct {
	SiteName    string `json:"sitename"`
	Generator   string `json:"generator"`
	Lang        string `json:"lang"`
	MainPage    string `json:"mainpage"`
	Pages       int    `json:"pages"`
	Articles    int    `json:"articles"`
	Edits       int    `json:"edits"`
	Images      int    `json:"images"`
	Users       int    `json:"users"`
	ActiveUsers int    `json:"activeusers"`
	Admins      int    `json:"admins"`
}

// Stats returns site info and statistics for the selected wiki.
func (c *Client) Stats(ctx context.Context) (*SiteStats, error) {
	v := c.actionParams()
	v.Set("action", "query")
	v.Set("meta", "siteinfo")
	v.Set("siprop", "general|statistics")
	var resp struct {
		apiError
		Query struct {
			General struct {
				SiteName  string `json:"sitename"`
				Generator string `json:"generator"`
				Lang      string `json:"lang"`
				MainPage  string `json:"mainpage"`
			} `json:"general"`
			Statistics struct {
				Pages       int `json:"pages"`
				Articles    int `json:"articles"`
				Edits       int `json:"edits"`
				Images      int `json:"images"`
				Users       int `json:"users"`
				ActiveUsers int `json:"activeusers"`
				Admins      int `json:"admins"`
			} `json:"statistics"`
		} `json:"query"`
	}
	if err := c.actionJSON(ctx, v, ttlSiteInfo, &resp); err != nil {
		return nil, err
	}
	if err := resp.err(); err != nil {
		return nil, err
	}
	g, s := resp.Query.General, resp.Query.Statistics
	return &SiteStats{
		SiteName: g.SiteName, Generator: g.Generator, Lang: g.Lang, MainPage: g.MainPage,
		Pages: s.Pages, Articles: s.Articles, Edits: s.Edits, Images: s.Images,
		Users: s.Users, ActiveUsers: s.ActiveUsers, Admins: s.Admins,
	}, nil
}
