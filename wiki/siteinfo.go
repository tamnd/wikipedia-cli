package wiki

import (
	"context"
	"encoding/json"
)

// SiteStats holds general site info and statistics for a wiki. It keeps the
// wider set of general fields (the server and paths, the software versions, the
// wiki id and the server time) and the full statistics block, so the structured
// output reflects what the siteinfo query returns.
type SiteStats struct {
	SiteName    string `json:"sitename"`
	Generator   string `json:"generator"`
	Lang        string `json:"lang"`
	MainPage    string `json:"mainpage"`
	Base        string `json:"base,omitempty"`
	Server      string `json:"server,omitempty"`
	ServerName  string `json:"servername,omitempty"`
	ArticlePath string `json:"articlepath,omitempty"`
	ScriptPath  string `json:"scriptpath,omitempty"`
	PHPVersion  string `json:"phpversion,omitempty"`
	DBType      string `json:"dbtype,omitempty"`
	DBVersion   string `json:"dbversion,omitempty"`
	WikiID      string `json:"wikiid,omitempty"`
	Time        string `json:"time,omitempty"`
	TimeZone    string `json:"timezone,omitempty"`
	WriteAPI    bool   `json:"writeapi,omitempty"`

	Pages       int `json:"pages"`
	Articles    int `json:"articles"`
	Edits       int `json:"edits"`
	Images      int `json:"images"`
	Users       int `json:"users"`
	ActiveUsers int `json:"activeusers"`
	Admins      int `json:"admins"`
	Jobs        int `json:"jobs,omitempty"`
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
				SiteName    string          `json:"sitename"`
				Generator   string          `json:"generator"`
				Lang        string          `json:"lang"`
				MainPage    string          `json:"mainpage"`
				Base        string          `json:"base"`
				Server      string          `json:"server"`
				ServerName  string          `json:"servername"`
				ArticlePath string          `json:"articlepath"`
				ScriptPath  string          `json:"scriptpath"`
				PHPVersion  string          `json:"phpversion"`
				DBType      string          `json:"dbtype"`
				DBVersion   string          `json:"dbversion"`
				WikiID      string          `json:"wikiid"`
				Time        string          `json:"time"`
				TimeZone    string          `json:"timezone"`
				WriteAPI    json.RawMessage `json:"writeapi"`
			} `json:"general"`
			Statistics struct {
				Pages       int `json:"pages"`
				Articles    int `json:"articles"`
				Edits       int `json:"edits"`
				Images      int `json:"images"`
				Users       int `json:"users"`
				ActiveUsers int `json:"activeusers"`
				Admins      int `json:"admins"`
				Jobs        int `json:"jobs"`
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
		Base: g.Base, Server: g.Server, ServerName: g.ServerName,
		ArticlePath: g.ArticlePath, ScriptPath: g.ScriptPath,
		PHPVersion: g.PHPVersion, DBType: g.DBType, DBVersion: g.DBVersion,
		WikiID: g.WikiID, Time: g.Time, TimeZone: g.TimeZone, WriteAPI: g.WriteAPI != nil,
		Pages: s.Pages, Articles: s.Articles, Edits: s.Edits, Images: s.Images,
		Users: s.Users, ActiveUsers: s.ActiveUsers, Admins: s.Admins, Jobs: s.Jobs,
	}, nil
}
