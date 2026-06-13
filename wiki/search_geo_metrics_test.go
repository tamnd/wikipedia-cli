package wiki

import (
	"encoding/json"
	"testing"
)

func TestSearchResultDecodeKeepsThumbnailAndKey(t *testing.T) {
	const blob = `{
      "id": 12345,
      "key": "Alan_Turing",
      "title": "Alan Turing",
      "excerpt": "an <span>English</span> mathematician",
      "matched_title": null,
      "description": "English mathematician",
      "thumbnail": {"mimetype": "image/jpeg", "width": 200, "height": 250, "duration": null, "url": "//example.org/t.jpg"}
    }`
	var p struct {
		ID           int              `json:"id"`
		Key          string           `json:"key"`
		Title        string           `json:"title"`
		Excerpt      string           `json:"excerpt"`
		MatchedTitle string           `json:"matched_title"`
		Description  string           `json:"description"`
		Thumbnail    *SearchThumbnail `json:"thumbnail"`
	}
	if err := json.Unmarshal([]byte(blob), &p); err != nil {
		t.Fatalf("decode: %v", err)
	}
	r := SearchResult{Title: p.Title, Pageid: p.ID, Key: p.Key, Description: p.Description, Snippet: stripHTML(p.Excerpt), Thumbnail: p.Thumbnail}
	if r.Pageid != 12345 || r.Key != "Alan_Turing" {
		t.Errorf("id/key dropped: %d %q", r.Pageid, r.Key)
	}
	if r.Title != "Alan Turing" || r.Description != "English mathematician" || r.Snippet != "an English mathematician" {
		t.Errorf("title/description/snippet dropped: %+v", r)
	}
	if r.Thumbnail == nil || r.Thumbnail.Width != 200 || r.Thumbnail.URL == "" {
		t.Errorf("thumbnail dropped: %+v", r.Thumbnail)
	}
}

func TestGeoResultDecodePreservesProps(t *testing.T) {
	const blob = `{"pageid": 26999, "ns": 0, "title": "Eiffel Tower", "lat": 48.8584, "lon": 2.2945, "dist": 12.3, "primary": "", "type": "landmark", "name": "Tour Eiffel", "dim": "1000", "country": "FR", "region": "75", "globe": "earth"}`
	var g struct {
		Pageid  int             `json:"pageid"`
		NS      int             `json:"ns"`
		Title   string          `json:"title"`
		Lat     float64         `json:"lat"`
		Lon     float64         `json:"lon"`
		Dist    float64         `json:"dist"`
		Primary json.RawMessage `json:"primary"`
		Type    string          `json:"type"`
		Name    string          `json:"name"`
		Dim     json.RawMessage `json:"dim"`
		Country string          `json:"country"`
		Region  string          `json:"region"`
		Globe   string          `json:"globe"`
	}
	if err := json.Unmarshal([]byte(blob), &g); err != nil {
		t.Fatalf("decode: %v", err)
	}
	r := GeoResult{Pageid: g.Pageid, NS: g.NS, Title: g.Title, Lat: g.Lat, Lon: g.Lon, Dist: g.Dist, Primary: g.Primary != nil, Type: g.Type, Name: g.Name, Dim: g.Dim, Country: g.Country, Region: g.Region, Globe: g.Globe}
	if !r.Primary {
		t.Errorf("primary presence not detected")
	}
	if r.Pageid != 26999 || r.Type != "landmark" || r.Country != "FR" || string(r.Dim) != `"1000"` {
		t.Errorf("geo props dropped: %+v", r)
	}
}

func TestRevisionDecodeKeepsUserIDAndTagsArray(t *testing.T) {
	const blob = `{"revid": 100, "parentid": 99, "timestamp": "2024-01-02T03:04:05Z", "user": "Alice", "userid": 7, "size": 1234, "sha1": "abcdef", "comment": "fix", "parsedcomment": "<i>fix</i>", "minor": true, "tags": ["mobile edit", "visualeditor"]}`
	var r Revision
	if err := json.Unmarshal([]byte(blob), &r); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if r.UserID != 7 || r.Sha1 != "abcdef" || r.ParsedComment == "" {
		t.Errorf("userid/sha1/parsedcomment dropped: %+v", r)
	}
	if len(r.Tags) != 2 || r.Tags[0] != "mobile edit" {
		t.Errorf("tags array dropped: %+v", r.Tags)
	}
	if r.TagsLine() != "mobile edit,visualeditor" {
		t.Errorf("TagsLine() = %q", r.TagsLine())
	}
}

func TestCategoryDecodeKeepsSortKeyAndHidden(t *testing.T) {
	const blob = `{"ns": 14, "title": "Category:Hidden categories", "sortkey": "abc123", "sortkeyprefix": "Turing", "timestamp": "2024-01-02T03:04:05Z", "hidden": ""}`
	var cat struct {
		NS            int             `json:"ns"`
		Title         string          `json:"title"`
		SortKey       string          `json:"sortkey"`
		SortKeyPrefix string          `json:"sortkeyprefix"`
		Timestamp     string          `json:"timestamp"`
		Hidden        json.RawMessage `json:"hidden"`
	}
	if err := json.Unmarshal([]byte(blob), &cat); err != nil {
		t.Fatalf("decode: %v", err)
	}
	c := Category{Title: cat.Title, NS: cat.NS, SortKey: cat.SortKey, SortKeyPrefix: cat.SortKeyPrefix, Timestamp: cat.Timestamp, Hidden: cat.Hidden != nil}
	if !c.Hidden || c.SortKeyPrefix != "Turing" || c.Timestamp == "" {
		t.Errorf("category fields dropped: %+v", c)
	}
}

func TestLangLinkDecodeKeepsLangName(t *testing.T) {
	const blob = `{"lang": "de", "title": "Berlin", "autonym": "Deutsch", "langname": "German", "url": "https://de.wikipedia.org/wiki/Berlin"}`
	var l LangLink
	if err := json.Unmarshal([]byte(blob), &l); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if l.LangName != "German" || l.Autonym != "Deutsch" {
		t.Errorf("langname/autonym dropped: %+v", l)
	}
}

func TestSiteStatsDecodeWidersGeneral(t *testing.T) {
	const blob = `{
      "query": {
        "general": {"sitename": "Wikipedia", "generator": "MediaWiki 1.43", "lang": "en", "mainpage": "Main Page", "base": "https://en.wikipedia.org/wiki/Main_Page", "server": "//en.wikipedia.org", "servername": "en.wikipedia.org", "articlepath": "/wiki/$1", "scriptpath": "/w", "phpversion": "8.1", "dbtype": "mysql", "wikiid": "enwiki", "timezone": "UTC", "writeapi": ""},
        "statistics": {"pages": 60000000, "articles": 6900000, "edits": 1200000000, "images": 900000, "users": 47000000, "activeusers": 120000, "admins": 850, "jobs": 42}
      }
    }`
	var resp struct {
		Query struct {
			General struct {
				SiteName    string          `json:"sitename"`
				Base        string          `json:"base"`
				Server      string          `json:"server"`
				ServerName  string          `json:"servername"`
				ArticlePath string          `json:"articlepath"`
				ScriptPath  string          `json:"scriptpath"`
				PHPVersion  string          `json:"phpversion"`
				DBType      string          `json:"dbtype"`
				WikiID      string          `json:"wikiid"`
				TimeZone    string          `json:"timezone"`
				WriteAPI    json.RawMessage `json:"writeapi"`
			} `json:"general"`
			Statistics struct {
				Jobs int `json:"jobs"`
			} `json:"statistics"`
		} `json:"query"`
	}
	if err := json.Unmarshal([]byte(blob), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	g := resp.Query.General
	s := SiteStats{Base: g.Base, Server: g.Server, ServerName: g.ServerName, ArticlePath: g.ArticlePath, ScriptPath: g.ScriptPath, PHPVersion: g.PHPVersion, DBType: g.DBType, WikiID: g.WikiID, TimeZone: g.TimeZone, WriteAPI: g.WriteAPI != nil, Jobs: resp.Query.Statistics.Jobs}
	if s.WikiID != "enwiki" || s.DBType != "mysql" || s.ArticlePath != "/wiki/$1" || !s.WriteAPI {
		t.Errorf("general fields dropped: %+v", s)
	}
	if s.Jobs != 42 {
		t.Errorf("jobs dropped: %d", s.Jobs)
	}
}

func TestPageviewPointKeepsContext(t *testing.T) {
	const blob = `{"project": "en.wikipedia", "article": "Alan_Turing", "granularity": "daily", "timestamp": "2024010100", "access": "all-access", "agent": "all-agents", "views": 4200}`
	var it struct {
		Project     string `json:"project"`
		Article     string `json:"article"`
		Granularity string `json:"granularity"`
		Timestamp   string `json:"timestamp"`
		Access      string `json:"access"`
		Agent       string `json:"agent"`
		Views       int    `json:"views"`
	}
	if err := json.Unmarshal([]byte(blob), &it); err != nil {
		t.Fatalf("decode: %v", err)
	}
	p := PageviewPoint{Date: formatViewDate(it.Timestamp, it.Granularity), Views: it.Views, Project: it.Project, Article: it.Article, Access: it.Access, Agent: it.Agent, Granularity: it.Granularity, Timestamp: it.Timestamp}
	if p.Project != "en.wikipedia" || p.Access != "all-access" || p.Agent != "all-agents" {
		t.Errorf("pageview context dropped: %+v", p)
	}
	if p.Date != "2024-01-01" {
		t.Errorf("date = %q", p.Date)
	}
}
