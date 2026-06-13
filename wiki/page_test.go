package wiki

import (
	"encoding/json"
	"testing"
)

// A representative REST page/summary blob: the namespace, the wikibase item,
// the title set, both image variants, the desktop and mobile URLs, the revision
// stamp and the coordinates must all survive the decode.
const sampleSummary = `{
  "type": "standard",
  "title": "Berlin",
  "displaytitle": "<span>Berlin</span>",
  "normalizedtitle": "Berlin",
  "namespace": {"id": 0, "text": ""},
  "wikibase_item": "Q64",
  "titles": {"canonical": "Berlin", "normalized": "Berlin", "display": "<span>Berlin</span>"},
  "pageid": 3354,
  "thumbnail": {"source": "https://example.org/thumb.jpg", "width": 320, "height": 213},
  "originalimage": {"source": "https://example.org/orig.jpg", "width": 2048, "height": 1365},
  "lang": "en",
  "dir": "ltr",
  "revision": "123456789",
  "tid": "abc-tid",
  "timestamp": "2024-01-02T03:04:05Z",
  "description": "Capital of Germany",
  "description_source": "local",
  "content_urls": {
    "desktop": {"page": "https://en.wikipedia.org/wiki/Berlin", "revisions": "https://en.wikipedia.org/wiki/Berlin?action=history", "edit": "https://en.wikipedia.org/wiki/Berlin?action=edit", "talk": "https://en.wikipedia.org/wiki/Talk:Berlin"},
    "mobile": {"page": "https://en.m.wikipedia.org/wiki/Berlin"}
  },
  "extract": "Berlin is the capital of Germany.",
  "extract_html": "<p>Berlin is the capital of Germany.</p>",
  "coordinates": {"lat": 52.52, "lon": 13.405}
}`

func TestSummaryDecodePreservesFullStructure(t *testing.T) {
	var s Summary
	if err := json.Unmarshal([]byte(sampleSummary), &s); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if s.WikibaseItem != "Q64" || s.Pageid != 3354 {
		t.Errorf("identity dropped: %q %d", s.WikibaseItem, s.Pageid)
	}
	if s.Namespace == nil || s.Namespace.ID != 0 {
		t.Errorf("namespace dropped: %+v", s.Namespace)
	}
	if s.Titles == nil || s.Titles.Canonical != "Berlin" {
		t.Errorf("titles dropped: %+v", s.Titles)
	}
	if s.OriginalImage == nil || s.OriginalImage.Width != 2048 {
		t.Errorf("originalimage dropped: %+v", s.OriginalImage)
	}
	if s.Revision != "123456789" || s.TID != "abc-tid" || s.Timestamp == "" {
		t.Errorf("revision stamp dropped: %q %q %q", s.Revision, s.TID, s.Timestamp)
	}
	if s.ExtractHTML == "" || s.DescriptionSource != "local" {
		t.Errorf("html extract / source dropped: %q %q", s.ExtractHTML, s.DescriptionSource)
	}
	if got := s.URL(); got != "https://en.wikipedia.org/wiki/Berlin" {
		t.Errorf("URL() = %q", got)
	}
	if s.ContentURLs == nil || s.ContentURLs.Mobile == nil || s.ContentURLs.Mobile.Page == "" {
		t.Errorf("mobile url dropped: %+v", s.ContentURLs)
	}
	if s.ThumbURL() != "https://example.org/thumb.jpg" {
		t.Errorf("ThumbURL() = %q", s.ThumbURL())
	}
	if s.Lat() != 52.52 || s.Lon() != 13.405 {
		t.Errorf("coordinates dropped: %v %v", s.Lat(), s.Lon())
	}
}

// A prop=info page object with everything the enriched inprop set returns.
const sampleInfo = `{
  "pageid": 3354,
  "ns": 0,
  "title": "Berlin",
  "contentmodel": "wikitext",
  "pagelanguage": "en",
  "pagelanguagehtmlcode": "en",
  "pagelanguagedir": "ltr",
  "touched": "2024-01-02T03:04:05Z",
  "lastrevid": 123456789,
  "length": 254000,
  "watchers": 2100,
  "talkid": 3355,
  "displaytitle": "<span>Berlin</span>",
  "protection": [{"type": "edit", "level": "autoconfirmed", "expiry": "infinity"}, {"type": "move", "level": "sysop", "expiry": "infinity"}],
  "restrictiontypes": ["edit", "move"],
  "varianttitles": {"en": "Berlin"},
  "fullurl": "https://en.wikipedia.org/wiki/Berlin",
  "editurl": "https://en.wikipedia.org/w/index.php?title=Berlin&action=edit",
  "canonicalurl": "https://en.wikipedia.org/wiki/Berlin"
}`

func TestPageInfoDecodePreservesFullStructure(t *testing.T) {
	var p PageInfo
	if err := json.Unmarshal([]byte(sampleInfo), &p); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if p.NS != 0 || p.TalkID != 3355 || p.Watchers != 2100 {
		t.Errorf("ns/talkid/watchers dropped: %d %d %d", p.NS, p.TalkID, p.Watchers)
	}
	if p.LangHTMLCode != "en" || p.LangDir != "ltr" {
		t.Errorf("language variants dropped: %q %q", p.LangHTMLCode, p.LangDir)
	}
	if len(p.Protection) != 2 || p.Protection[1].Type != "move" || p.Protection[1].Level != "sysop" {
		t.Errorf("protection dropped: %+v", p.Protection)
	}
	if len(p.RestrictionTypes) != 2 {
		t.Errorf("restriction types dropped: %+v", p.RestrictionTypes)
	}
	if p.VariantTitles["en"] != "Berlin" {
		t.Errorf("variant titles dropped: %+v", p.VariantTitles)
	}
	if p.EditURL == "" || p.CanonicalURL == "" {
		t.Errorf("urls dropped: %q %q", p.EditURL, p.CanonicalURL)
	}
}

// A compare response with both endpoints and a small diff body.
const sampleCompare = `{
  "fromid": 3354,
  "fromrevid": 100,
  "fromns": 0,
  "fromtitle": "Berlin",
  "toid": 3354,
  "torevid": 200,
  "tons": 0,
  "totitle": "Berlin",
  "body": "<tr><td class=\"diff-addedline\"><div>new text</div></td></tr><tr><td class=\"diff-deletedline\"><div>old text</div></td></tr>"
}`

func TestDiffDecodePreservesEndpointsAndBody(t *testing.T) {
	var resp struct {
		Compare struct {
			FromID    int    `json:"fromid"`
			FromRevID int    `json:"fromrevid"`
			FromNS    int    `json:"fromns"`
			FromTitle string `json:"fromtitle"`
			ToID      int    `json:"toid"`
			ToRevID   int    `json:"torevid"`
			ToNS      int    `json:"tons"`
			ToTitle   string `json:"totitle"`
			Body      string `json:"body"`
		} `json:"compare"`
	}
	wrapped := `{"compare":` + sampleCompare + `}`
	if err := json.Unmarshal([]byte(wrapped), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	cm := resp.Compare
	d := &Diff{
		FromID: cm.FromID, FromRevID: cm.FromRevID, FromNS: cm.FromNS, FromTitle: cm.FromTitle,
		ToID: cm.ToID, ToRevID: cm.ToRevID, ToNS: cm.ToNS, ToTitle: cm.ToTitle,
		Body:  cm.Body,
		Lines: parseDiffTable(cm.Body),
	}
	if d.FromRevID != 100 || d.ToRevID != 200 {
		t.Errorf("endpoints dropped: %d -> %d", d.FromRevID, d.ToRevID)
	}
	if d.FromID != 3354 || d.ToID != 3354 || d.FromNS != 0 || d.ToNS != 0 {
		t.Errorf("page id/ns dropped: %+v", d)
	}
	if d.FromTitle != "Berlin" || d.ToTitle != "Berlin" {
		t.Errorf("titles dropped: %q %q", d.FromTitle, d.ToTitle)
	}
	if d.Body == "" {
		t.Errorf("raw body dropped")
	}
	if len(d.Lines) != 2 || d.Lines[0].Op != "add" || d.Lines[1].Op != "del" {
		t.Errorf("parsed lines wrong: %+v", d.Lines)
	}
}
