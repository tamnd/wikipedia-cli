package wiki

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// Summary is the REST page summary (the blob the mobile apps show), with its
// full structure preserved. The JSON encoding mirrors the page/summary
// response, so the display and normalized titles, namespace, wikibase item,
// the HTML extract, the original image, the desktop and mobile URLs, and the
// revision stamp all survive a round trip. The accessor methods read the
// flattened fields the table and plain output use.
type Summary struct {
	Type              string              `json:"type,omitempty"`
	Title             string              `json:"title"`
	DisplayTitle      string              `json:"displaytitle,omitempty"`
	NormalizedTitle   string              `json:"normalizedtitle,omitempty"`
	Namespace         *SummaryNamespace   `json:"namespace,omitempty"`
	WikibaseItem      string              `json:"wikibase_item,omitempty"`
	Titles            *SummaryTitles      `json:"titles,omitempty"`
	Pageid            int                 `json:"pageid,omitempty"`
	Thumbnail         *SummaryImage       `json:"thumbnail,omitempty"`
	OriginalImage     *SummaryImage       `json:"originalimage,omitempty"`
	Lang              string              `json:"lang,omitempty"`
	Dir               string              `json:"dir,omitempty"`
	Revision          string              `json:"revision,omitempty"`
	TID               string              `json:"tid,omitempty"`
	Timestamp         string              `json:"timestamp,omitempty"`
	Description       string              `json:"description,omitempty"`
	DescriptionSource string              `json:"description_source,omitempty"`
	ContentURLs       *SummaryContentURLs `json:"content_urls,omitempty"`
	Extract           string              `json:"extract"`
	ExtractHTML       string              `json:"extract_html,omitempty"`
	Coordinates       *SummaryCoordinates `json:"coordinates,omitempty"`

	// fallbackURL is used by URL() when the response carries no content URL.
	fallbackURL string
}

// SummaryNamespace is the namespace of a summary: its numeric id and text.
type SummaryNamespace struct {
	ID   int    `json:"id"`
	Text string `json:"text,omitempty"`
}

// SummaryTitles holds the canonical, normalized and display forms of a title.
type SummaryTitles struct {
	Canonical  string `json:"canonical,omitempty"`
	Normalized string `json:"normalized,omitempty"`
	Display    string `json:"display,omitempty"`
}

// SummaryImage is a thumbnail or original image: its source URL and size.
type SummaryImage struct {
	Source string `json:"source"`
	Width  int    `json:"width,omitempty"`
	Height int    `json:"height,omitempty"`
}

// SummaryContentURLs is the desktop and mobile URL set for a summary.
type SummaryContentURLs struct {
	Desktop *SummaryURLs `json:"desktop,omitempty"`
	Mobile  *SummaryURLs `json:"mobile,omitempty"`
}

// SummaryURLs is the set of URLs for a page on one platform.
type SummaryURLs struct {
	Page      string `json:"page,omitempty"`
	Revisions string `json:"revisions,omitempty"`
	Edit      string `json:"edit,omitempty"`
	Talk      string `json:"talk,omitempty"`
}

// SummaryCoordinates is a page's geographic location.
type SummaryCoordinates struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

// URL returns the desktop page URL, falling back to a computed page URL.
func (s Summary) URL() string {
	if s.ContentURLs != nil && s.ContentURLs.Desktop != nil && s.ContentURLs.Desktop.Page != "" {
		return s.ContentURLs.Desktop.Page
	}
	return s.fallbackURL
}

// ThumbURL returns the thumbnail source URL, or "".
func (s Summary) ThumbURL() string {
	if s.Thumbnail != nil {
		return s.Thumbnail.Source
	}
	return ""
}

// Lat returns the page latitude, or 0.
func (s Summary) Lat() float64 {
	if s.Coordinates != nil {
		return s.Coordinates.Lat
	}
	return 0
}

// Lon returns the page longitude, or 0.
func (s Summary) Lon() float64 {
	if s.Coordinates != nil {
		return s.Coordinates.Lon
	}
	return 0
}

// GetSummary fetches the REST summary for a title. The whole response is
// decoded as-is, so the structured output is a faithful copy of the API.
func (c *Client) GetSummary(ctx context.Context, title string) (*Summary, error) {
	var s Summary
	u := c.Site.RestV1("page/summary/" + titlePath(title))
	if err := c.HTTP.GetJSON(ctx, u, ttlContent, &s); err != nil {
		return nil, err
	}
	s.fallbackURL = c.Site.PageURL(title)
	return &s, nil
}

// Extract returns the plain-text extract of a page. When intro is true only the
// lead section is returned; otherwise the whole article as plain text.
func (c *Client) Extract(ctx context.Context, title string, intro bool) (string, error) {
	v := c.actionParams()
	v.Set("action", "query")
	v.Set("prop", "extracts")
	v.Set("explaintext", "1")
	v.Set("redirects", "1")
	if intro {
		v.Set("exintro", "1")
	}
	v.Set("titles", title)
	text, _, err := c.extractQuery(ctx, v)
	return text, err
}

func (c *Client) extractQuery(ctx context.Context, v url.Values) (text, canonical string, err error) {
	var resp struct {
		apiError
		Query struct {
			Pages []struct {
				Title   string `json:"title"`
				Extract string `json:"extract"`
				Missing bool   `json:"missing"`
			} `json:"pages"`
		} `json:"query"`
	}
	if err := c.actionJSON(ctx, v, ttlContent, &resp); err != nil {
		return "", "", err
	}
	if err := resp.err(); err != nil {
		return "", "", err
	}
	if len(resp.Query.Pages) == 0 {
		return "", "", ErrNotFound
	}
	p := resp.Query.Pages[0]
	if p.Missing {
		return "", "", ErrNotFound
	}
	return p.Extract, p.Title, nil
}

// HTML returns the rendered HTML of a page (REST page/html).
func (c *Client) HTML(ctx context.Context, title string) ([]byte, error) {
	u := c.Site.RestV1("page/html/" + titlePath(title))
	return c.HTTP.GetText(ctx, u, "text/html")
}

// Wikitext returns the raw wikitext source of a page's current revision.
func (c *Client) Wikitext(ctx context.Context, title string) (string, error) {
	v := c.actionParams()
	v.Set("action", "query")
	v.Set("prop", "revisions")
	v.Set("rvprop", "content")
	v.Set("rvslots", "main")
	v.Set("redirects", "1")
	v.Set("titles", title)
	var resp struct {
		apiError
		Query struct {
			Pages []struct {
				Missing   bool `json:"missing"`
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
	if len(resp.Query.Pages) == 0 || resp.Query.Pages[0].Missing || len(resp.Query.Pages[0].Revisions) == 0 {
		return "", ErrNotFound
	}
	return resp.Query.Pages[0].Revisions[0].Slots.Main.Content, nil
}

// Section is one parsed section of a page.
type Section struct {
	Index  string `json:"index"`
	Level  string `json:"level"`
	Line   string `json:"line"`
	Anchor string `json:"anchor"`
	Number string `json:"number"`
}

// Sections returns the section table of contents for a page.
func (c *Client) Sections(ctx context.Context, title string) ([]Section, error) {
	v := c.actionParams()
	v.Set("action", "parse")
	v.Set("page", title)
	v.Set("prop", "sections")
	v.Set("redirects", "1")
	var resp struct {
		apiError
		Parse struct {
			Sections []Section `json:"sections"`
		} `json:"parse"`
	}
	if err := c.actionJSON(ctx, v, ttlContent, &resp); err != nil {
		return nil, err
	}
	if err := resp.err(); err != nil {
		return nil, err
	}
	return resp.Parse.Sections, nil
}

// SectionHTML returns the rendered HTML of a single section by index, via the
// parse API. The caller converts it to text/Markdown with the render helpers.
func (c *Client) SectionHTML(ctx context.Context, title string, section int) (string, error) {
	v := c.actionParams()
	v.Set("action", "parse")
	v.Set("page", title)
	v.Set("prop", "text")
	v.Set("section", strconv.Itoa(section))
	v.Set("redirects", "1")
	var resp struct {
		apiError
		Parse struct {
			Text string `json:"text"`
		} `json:"parse"`
	}
	if err := c.actionJSON(ctx, v, ttlContent, &resp); err != nil {
		return "", err
	}
	if err := resp.err(); err != nil {
		return "", err
	}
	return resp.Parse.Text, nil
}

// PageInfo is page metadata from prop=info. It keeps everything the info query
// returns: the namespace, the page-length and revision stamps, the language and
// its variants, the protection levels, the talk-page id, and the canonical,
// edit and full URLs.
type PageInfo struct {
	Pageid             int               `json:"pageid"`
	NS                 int               `json:"ns"`
	Title              string            `json:"title"`
	ContentModel       string            `json:"contentmodel,omitempty"`
	Lang               string            `json:"pagelanguage,omitempty"`
	LangHTMLCode       string            `json:"pagelanguagehtmlcode,omitempty"`
	LangDir            string            `json:"pagelanguagedir,omitempty"`
	Touched            string            `json:"touched,omitempty"`
	LastRevID          int               `json:"lastrevid,omitempty"`
	Length             int               `json:"length,omitempty"`
	Redirect           bool              `json:"redirect,omitempty"`
	New                bool              `json:"new,omitempty"`
	Watchers           int               `json:"watchers,omitempty"`
	TalkID             int               `json:"talkid,omitempty"`
	DisplayTitle       string            `json:"displaytitle,omitempty"`
	Protection         []Protection      `json:"protection,omitempty"`
	RestrictionTypes   []string          `json:"restrictiontypes,omitempty"`
	VariantTitles      map[string]string `json:"varianttitles,omitempty"`
	FullURL            string            `json:"fullurl,omitempty"`
	EditURL            string            `json:"editurl,omitempty"`
	CanonicalURL       string            `json:"canonicalurl,omitempty"`

	// URL is the canonical web URL, kept for the table view and citations.
	URL string `json:"url"`
}

// Protection is one protection entry: the action it guards, the group level
// required, and an optional expiry.
type Protection struct {
	Type   string `json:"type"`
	Level  string `json:"level"`
	Expiry string `json:"expiry,omitempty"`
}

// Info returns page metadata for a title.
func (c *Client) Info(ctx context.Context, title string) (*PageInfo, error) {
	v := c.actionParams()
	v.Set("action", "query")
	v.Set("prop", "info")
	v.Set("inprop", "url|displaytitle|protection|talkid|varianttitles|watchers")
	v.Set("redirects", "1")
	v.Set("titles", title)
	var resp struct {
		apiError
		Query struct {
			Pages []struct {
				PageInfo
				Missing bool `json:"missing"`
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
	info := resp.Query.Pages[0].PageInfo
	info.URL = info.CanonicalURL
	if info.URL == "" {
		info.URL = info.FullURL
	}
	if info.URL == "" {
		info.URL = c.Site.PageURL(title)
	}
	return &info, nil
}

// ErrNotFound is returned when a page/entity does not exist.
var ErrNotFound = fmt.Errorf("not found")
