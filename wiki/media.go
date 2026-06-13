package wiki

import (
	"context"
	"encoding/json"
)

// Media is one file used on a page, with its full imageinfo preserved. The JSON
// encoding mirrors the query+imageinfo response, so the description URLs, media
// type, sha1, thumbnail, embedded metadata and the whole extmetadata block all
// survive a round trip. The convenience methods (URL, Mime, License, Author)
// read from the current revision for table output and downloads.
type Media struct {
	Title           string      `json:"title"`
	PageID          int         `json:"pageid,omitempty"`
	NS              int         `json:"ns,omitempty"`
	ImageRepository string      `json:"imagerepository,omitempty"`
	ImageInfo       []ImageInfo `json:"imageinfo,omitempty"`
}

// ImageInfo is one revision of a file's metadata as returned by the imageinfo
// query. Embedded EXIF/format metadata and the extmetadata block are kept as
// raw JSON so nothing the API reports is dropped.
type ImageInfo struct {
	Timestamp        string                  `json:"timestamp,omitempty"`
	User             string                  `json:"user,omitempty"`
	UserID           int                     `json:"userid,omitempty"`
	Size             int                     `json:"size,omitempty"`
	Width            int                     `json:"width,omitempty"`
	Height           int                     `json:"height,omitempty"`
	Comment          string                  `json:"comment,omitempty"`
	URL              string                  `json:"url,omitempty"`
	DescriptionURL   string                  `json:"descriptionurl,omitempty"`
	DescriptionShort string                  `json:"descriptionshorturl,omitempty"`
	ThumbURL         string                  `json:"thumburl,omitempty"`
	ThumbWidth       int                     `json:"thumbwidth,omitempty"`
	ThumbHeight      int                     `json:"thumbheight,omitempty"`
	Mime             string                  `json:"mime,omitempty"`
	MediaType        string                  `json:"mediatype,omitempty"`
	BitDepth         int                     `json:"bitdepth,omitempty"`
	Duration         float64                 `json:"duration,omitempty"`
	PageCount        int                     `json:"pagecount,omitempty"`
	Sha1             string                  `json:"sha1,omitempty"`
	CanonicalTitle   string                  `json:"canonicaltitle,omitempty"`
	Metadata         json.RawMessage         `json:"metadata,omitempty"`
	CommonMetadata   json.RawMessage         `json:"commonmetadata,omitempty"`
	ExtMetadata      map[string]ExtMetaValue `json:"extmetadata,omitempty"`
}

// ExtMetaValue is one entry of the extmetadata block: its value (kept raw so a
// string, number or HTML fragment all survive), the source that supplied it,
// and whether the wiki marks it hidden.
type ExtMetaValue struct {
	Value  json.RawMessage `json:"value"`
	Source string          `json:"source,omitempty"`
	Hidden string          `json:"hidden,omitempty"`
}

// Current returns the most recent file revision, or nil when none was returned.
func (m Media) Current() *ImageInfo {
	if len(m.ImageInfo) == 0 {
		return nil
	}
	return &m.ImageInfo[0]
}

// URL is the current revision's file URL, or "".
func (m Media) URL() string {
	if ii := m.Current(); ii != nil {
		return ii.URL
	}
	return ""
}

// Mime is the current revision's MIME type, or "".
func (m Media) Mime() string {
	if ii := m.Current(); ii != nil {
		return ii.Mime
	}
	return ""
}

// Width is the current revision's width in pixels, or 0.
func (m Media) Width() int {
	if ii := m.Current(); ii != nil {
		return ii.Width
	}
	return 0
}

// Height is the current revision's height in pixels, or 0.
func (m Media) Height() int {
	if ii := m.Current(); ii != nil {
		return ii.Height
	}
	return 0
}

// Size is the current revision's byte size, or 0.
func (m Media) Size() int {
	if ii := m.Current(); ii != nil {
		return ii.Size
	}
	return 0
}

// License is the short license name from extmetadata, or "".
func (m Media) License() string { return m.extMetaString("LicenseShortName") }

// Author is the artist from extmetadata with HTML stripped, or "".
func (m Media) Author() string { return stripHTML(m.extMetaString("Artist")) }

// extMetaString reads a string extmetadata value by key from the current
// revision, tolerating either a JSON string or a bare scalar.
func (m Media) extMetaString(key string) string {
	ii := m.Current()
	if ii == nil {
		return ""
	}
	ev, ok := ii.ExtMetadata[key]
	if !ok {
		return ""
	}
	var s string
	if json.Unmarshal(ev.Value, &s) == nil {
		return s
	}
	return string(ev.Value)
}

// Media returns the files used on a page with their full imageinfo: url, mime,
// dimensions, media type, sha1, embedded metadata and the complete extmetadata
// block. Nothing the API returns is dropped.
func (c *Client) Media(ctx context.Context, title string, limit int) ([]Media, error) {
	v := c.actionParams()
	v.Set("action", "query")
	v.Set("generator", "images")
	v.Set("gimlimit", limitParam(limit))
	v.Set("prop", "imageinfo")
	v.Set("iiprop", "timestamp|user|userid|comment|url|size|dimensions|sha1|mime|mediatype|bitdepth|metadata|commonmetadata|extmetadata|canonicaltitle")
	v.Set("redirects", "1")
	v.Set("titles", title)
	var resp struct {
		apiError
		Query struct {
			Pages []Media `json:"pages"`
		} `json:"query"`
	}
	if err := c.actionJSON(ctx, v, ttlContent, &resp); err != nil {
		return nil, err
	}
	if err := resp.err(); err != nil {
		return nil, err
	}
	var out []Media
	for _, m := range resp.Query.Pages {
		if m.URL() == "" {
			continue
		}
		out = append(out, m)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out, nil
}
