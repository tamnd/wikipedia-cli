package wiki

import (
	"context"
)

// Media is one file used on a page.
type Media struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Mime    string `json:"mime,omitempty"`
	Width   int    `json:"width,omitempty"`
	Height  int    `json:"height,omitempty"`
	Size    int    `json:"size,omitempty"`
	License string `json:"license,omitempty"`
	Author  string `json:"author,omitempty"`
}

// Media returns the files used on a page with their imageinfo (url, mime, size,
// dimensions, and license/author where available).
func (c *Client) Media(ctx context.Context, title string, limit int) ([]Media, error) {
	v := c.actionParams()
	v.Set("action", "query")
	v.Set("generator", "images")
	v.Set("gimlimit", limitParam(limit))
	v.Set("prop", "imageinfo")
	v.Set("iiprop", "url|size|mime|extmetadata")
	v.Set("redirects", "1")
	v.Set("titles", title)
	var resp struct {
		apiError
		Query struct {
			Pages []struct {
				Title     string `json:"title"`
				ImageInfo []struct {
					URL    string `json:"url"`
					Mime   string `json:"mime"`
					Width  int    `json:"width"`
					Height int    `json:"height"`
					Size   int    `json:"size"`
					Meta   struct {
						License struct {
							Value string `json:"value"`
						} `json:"LicenseShortName"`
						Artist struct {
							Value string `json:"value"`
						} `json:"Artist"`
					} `json:"extmetadata"`
				} `json:"imageinfo"`
			} `json:"pages"`
		} `json:"query"`
	}
	if err := c.actionJSON(ctx, v, ttlContent, &resp); err != nil {
		return nil, err
	}
	if err := resp.err(); err != nil {
		return nil, err
	}
	var out []Media
	for _, p := range resp.Query.Pages {
		m := Media{Title: p.Title}
		if len(p.ImageInfo) > 0 {
			ii := p.ImageInfo[0]
			m.URL = ii.URL
			m.Mime = ii.Mime
			m.Width = ii.Width
			m.Height = ii.Height
			m.Size = ii.Size
			m.License = ii.Meta.License.Value
			m.Author = stripHTML(ii.Meta.Artist.Value)
		}
		if m.URL == "" {
			continue
		}
		out = append(out, m)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out, nil
}
