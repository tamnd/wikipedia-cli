package wiki

import (
	"context"
	"fmt"
	"time"
)

// FeaturedFeed is the daily featured feed for a date. Its JSON encoding mirrors
// the REST feed/featured response, so the featured article, most-read list,
// picture of the day, in-the-news stories and on-this-day highlights all keep
// their full page summaries.
type FeaturedFeed struct {
	TFA       *FeedArticle `json:"tfa,omitempty"`
	MostRead  *MostRead    `json:"mostread,omitempty"`
	Image     *FeedImage   `json:"image,omitempty"`
	News      []FeedNews   `json:"news,omitempty"`
	OnThisDay []OnThisDay  `json:"onthisday,omitempty"`
}

// MostRead is the most-read block of the featured feed: the date it covers and
// the ranked articles.
type MostRead struct {
	Date     string        `json:"date,omitempty"`
	Articles []FeedArticle `json:"articles,omitempty"`
}

// FeedArticle is a page summary as it appears in a feed, with the feed-only
// view counters kept alongside the standard summary fields. Every field the
// REST summary carries is preserved so the original record can be rebuilt.
type FeedArticle struct {
	Type              string         `json:"type,omitempty"`
	Title             string         `json:"title"`
	DisplayTitle      string         `json:"displaytitle,omitempty"`
	NormalizedTitle   string         `json:"normalizedtitle,omitempty"`
	PageID            int            `json:"pageid,omitempty"`
	Lang              string         `json:"lang,omitempty"`
	Dir               string         `json:"dir,omitempty"`
	Revision          string         `json:"revision,omitempty"`
	TID               string         `json:"tid,omitempty"`
	Timestamp         string         `json:"timestamp,omitempty"`
	Description       string         `json:"description,omitempty"`
	DescriptionSource string         `json:"description_source,omitempty"`
	WikibaseItem      string         `json:"wikibase_item,omitempty"`
	Namespace         *FeedNamespace `json:"namespace,omitempty"`
	Titles            *FeedTitles    `json:"titles,omitempty"`
	Thumbnail         *FeedImageInfo `json:"thumbnail,omitempty"`
	OriginalImage     *FeedImageInfo `json:"originalimage,omitempty"`
	ContentURLs       *ContentURLs   `json:"content_urls,omitempty"`
	Extract           string         `json:"extract,omitempty"`
	ExtractHTML       string         `json:"extract_html,omitempty"`
	Coordinates       *Coordinates   `json:"coordinates,omitempty"`

	// Feed-only fields, present on the most-read articles.
	Views       int           `json:"views,omitempty"`
	Rank        int           `json:"rank,omitempty"`
	ViewHistory []ViewHistory `json:"view_history,omitempty"`
}

// FeedNamespace is the namespace of a summary: its numeric id and text.
type FeedNamespace struct {
	ID   int    `json:"id"`
	Text string `json:"text,omitempty"`
}

// FeedTitles holds the canonical, normalized and display forms of a title.
type FeedTitles struct {
	Canonical  string `json:"canonical,omitempty"`
	Normalized string `json:"normalized,omitempty"`
	Display    string `json:"display,omitempty"`
}

// FeedImageInfo is a thumbnail or original image: its source URL and size.
type FeedImageInfo struct {
	Source string `json:"source"`
	Width  int    `json:"width,omitempty"`
	Height int    `json:"height,omitempty"`
}

// ContentURLs is the desktop and mobile URL set for a page summary.
type ContentURLs struct {
	Desktop *PageURLs `json:"desktop,omitempty"`
	Mobile  *PageURLs `json:"mobile,omitempty"`
}

// PageURLs is the set of URLs for a page on one platform.
type PageURLs struct {
	Page      string `json:"page,omitempty"`
	Revisions string `json:"revisions,omitempty"`
	Edit      string `json:"edit,omitempty"`
	Talk      string `json:"talk,omitempty"`
}

// Coordinates is a page's geographic location.
type Coordinates struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

// ViewHistory is one day of a most-read article's view count.
type ViewHistory struct {
	Date  string `json:"date"`
	Views int    `json:"views"`
}

// FeedImage is the picture of the day, with its license, credit and the full
// thumbnail and original image preserved.
type FeedImage struct {
	Title       string         `json:"title"`
	Thumbnail   *FeedImageInfo `json:"thumbnail,omitempty"`
	Image       *FeedImageInfo `json:"image,omitempty"`
	FilePage    string         `json:"file_page,omitempty"`
	Artist      *FeedCredit    `json:"artist,omitempty"`
	Credit      *FeedCredit    `json:"credit,omitempty"`
	License     *FeedLicense   `json:"license,omitempty"`
	Description *FeedText      `json:"description,omitempty"`
	WBEntityID  string         `json:"wb_entity_id,omitempty"`
}

// FeedCredit is an artist or credit block: an HTML fragment, its plain text and
// an optional name.
type FeedCredit struct {
	HTML string `json:"html,omitempty"`
	Text string `json:"text,omitempty"`
	Name string `json:"name,omitempty"`
}

// FeedLicense is the license of the picture of the day.
type FeedLicense struct {
	Type string `json:"type,omitempty"`
	Code string `json:"code,omitempty"`
	URL  string `json:"url,omitempty"`
}

// FeedText is a language-tagged text block with both plain and HTML forms.
type FeedText struct {
	Text string `json:"text,omitempty"`
	HTML string `json:"html,omitempty"`
	Lang string `json:"lang,omitempty"`
}

// FeedNews is an in-the-news story: the HTML story line and the linked article
// summaries it mentions.
type FeedNews struct {
	Story string        `json:"story"`
	Links []FeedArticle `json:"links,omitempty"`
}

// OnThisDay is a historical event: the year, the event text, and the full page
// summaries of the articles it links.
type OnThisDay struct {
	Year  int           `json:"year,omitempty"`
	Text  string        `json:"text"`
	Pages []FeedArticle `json:"pages,omitempty"`
}

// URL returns the desktop page URL of a feed article, or "".
func (a FeedArticle) URL() string {
	if a.ContentURLs != nil && a.ContentURLs.Desktop != nil {
		return a.ContentURLs.Desktop.Page
	}
	return ""
}

// URL returns the picture-of-the-day's full image source, or "".
func (i FeedImage) URL() string {
	if i.Image != nil {
		return i.Image.Source
	}
	return ""
}

// DescriptionText returns the picture-of-the-day's description as plain text.
func (i FeedImage) DescriptionText() string {
	if i.Description != nil {
		return stripHTML(i.Description.Text)
	}
	return ""
}

// StoryText returns an in-the-news story as plain text.
func (n FeedNews) StoryText() string { return stripHTML(n.Story) }

// TextPlain returns an on-this-day event's text as plain text.
func (o OnThisDay) TextPlain() string { return stripHTML(o.Text) }

// PageTitles returns the titles of an event's linked pages for table output.
func (o OnThisDay) PageTitles() []string {
	out := make([]string, 0, len(o.Pages))
	for _, p := range o.Pages {
		out = append(out, p.Title)
	}
	return out
}

// Featured fetches the daily featured feed for a date (in the wiki's language).
// The whole REST response is decoded as-is, so the structured output is a
// faithful copy of what the API returns.
func (c *Client) Featured(ctx context.Context, date time.Time) (*FeaturedFeed, error) {
	path := fmt.Sprintf("feed/featured/%04d/%02d/%02d", date.Year(), int(date.Month()), date.Day())
	var feed FeaturedFeed
	if err := c.HTTP.GetJSON(ctx, c.Site.RestV1(path), ttlFeed, &feed); err != nil {
		return nil, err
	}
	return &feed, nil
}

// OnThisDayEvents fetches historical events of a given type for a month/day.
// eventType is one of all/selected/births/deaths/holidays/events. Each event
// keeps the full page summaries of the articles it links.
func (c *Client) OnThisDayEvents(ctx context.Context, eventType string, month, day int) ([]OnThisDay, error) {
	if eventType == "" {
		eventType = "all"
	}
	path := fmt.Sprintf("feed/onthisday/%s/%02d/%02d", eventType, month, day)
	var resp map[string][]OnThisDay
	if err := c.HTTP.GetJSON(ctx, c.Site.RestV1(path), ttlFeed, &resp); err != nil {
		return nil, err
	}
	var out []OnThisDay
	order := []string{"selected", "events", "births", "deaths", "holidays"}
	if eventType == "all" {
		for _, k := range order {
			out = append(out, resp[k]...)
		}
	} else {
		out = append(out, resp[eventType]...)
	}
	return out, nil
}
