package wiki

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// Summary is the short REST page summary (the blob the mobile apps show).
type Summary struct {
	Title       string  `json:"title" kit:"id"`
	Description string  `json:"description,omitempty"`
	Extract     string  `json:"extract" kit:"body"`
	Type        string  `json:"type,omitempty"`
	Lang        string  `json:"lang,omitempty"`
	URL         string  `json:"url"`
	Thumbnail   string  `json:"thumbnail,omitempty"`
	Lat         float64 `json:"lat,omitempty"`
	Lon         float64 `json:"lon,omitempty"`
	Pageid      int     `json:"pageid,omitempty"`
}

// GetSummary fetches the REST summary for a title.
func (c *Client) GetSummary(ctx context.Context, title string) (*Summary, error) {
	var resp struct {
		Title        string `json:"title"`
		Displaytitle string `json:"displaytitle"`
		Description  string `json:"description"`
		Extract      string `json:"extract"`
		Type         string `json:"type"`
		Lang         string `json:"lang"`
		Pageid       int    `json:"pageid"`
		Thumbnail    struct {
			Source string `json:"source"`
		} `json:"thumbnail"`
		ContentURLs struct {
			Desktop struct {
				Page string `json:"page"`
			} `json:"desktop"`
		} `json:"content_urls"`
		Coordinates *struct {
			Lat float64 `json:"lat"`
			Lon float64 `json:"lon"`
		} `json:"coordinates"`
	}
	u := c.Site.RestV1("page/summary/" + titlePath(title))
	if err := c.HTTP.GetJSON(ctx, u, ttlContent, &resp); err != nil {
		return nil, err
	}
	s := &Summary{
		Title:       resp.Title,
		Description: resp.Description,
		Extract:     resp.Extract,
		Type:        resp.Type,
		Lang:        resp.Lang,
		URL:         resp.ContentURLs.Desktop.Page,
		Thumbnail:   resp.Thumbnail.Source,
		Pageid:      resp.Pageid,
	}
	if s.URL == "" {
		s.URL = c.Site.PageURL(title)
	}
	if resp.Coordinates != nil {
		s.Lat, s.Lon = resp.Coordinates.Lat, resp.Coordinates.Lon
	}
	return s, nil
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

// PageInfo is page metadata from prop=info.
type PageInfo struct {
	Pageid       int    `json:"pageid"`
	Title        string `json:"title"`
	Length       int    `json:"length"`
	Touched      string `json:"touched"`
	LastRevID    int    `json:"lastrevid"`
	ContentModel string `json:"contentmodel"`
	Lang         string `json:"pagelanguage"`
	Redirect     bool   `json:"redirect"`
	URL          string `json:"url"`
}

// Info returns page metadata for a title.
func (c *Client) Info(ctx context.Context, title string) (*PageInfo, error) {
	v := c.actionParams()
	v.Set("action", "query")
	v.Set("prop", "info")
	v.Set("inprop", "url|displaytitle")
	v.Set("redirects", "1")
	v.Set("titles", title)
	var resp struct {
		apiError
		Query struct {
			Pages []struct {
				PageInfo
				Missing bool   `json:"missing"`
				FullURL string `json:"fullurl"`
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
	p := resp.Query.Pages[0]
	info := p.PageInfo
	info.URL = p.FullURL
	if info.URL == "" {
		info.URL = c.Site.PageURL(title)
	}
	return &info, nil
}

// ErrNotFound is returned when a page/entity does not exist.
var ErrNotFound = fmt.Errorf("not found")
