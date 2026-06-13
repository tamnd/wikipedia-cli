package wiki

import (
	"encoding/json"
	"testing"
)

// A trimmed feed/featured response carrying the parts that used to be dropped:
// the TFA's thumbnail and mobile urls, a most-read article's view_history, the
// picture of the day's license and credit, an in-the-news link summary, and an
// on-this-day entry with linked page summaries. None may be lost.
const sampleFeatured = `{
  "tfa": {
    "type": "standard",
    "title": "Alan_Turing",
    "normalizedtitle": "Alan Turing",
    "pageid": 1208,
    "wikibase_item": "Q7251",
    "extract": "Alan Turing was a mathematician.",
    "extract_html": "<p>Alan Turing was a mathematician.</p>",
    "thumbnail": {"source": "https://up/thumb.jpg", "width": 240, "height": 320},
    "originalimage": {"source": "https://up/orig.jpg", "width": 800, "height": 1000},
    "content_urls": {
      "desktop": {"page": "https://en.wikipedia.org/wiki/Alan_Turing"},
      "mobile": {"page": "https://en.m.wikipedia.org/wiki/Alan_Turing"}
    },
    "timestamp": "2024-01-01T00:00:00Z"
  },
  "mostread": {
    "date": "2024-01-01Z",
    "articles": [{
      "title": "Foo", "views": 12345, "rank": 1,
      "view_history": [{"date": "2023-12-30Z", "views": 100}, {"date": "2023-12-31Z", "views": 200}],
      "content_urls": {"desktop": {"page": "https://en.wikipedia.org/wiki/Foo"}}
    }]
  },
  "image": {
    "title": "File:Sunset.jpg",
    "thumbnail": {"source": "https://up/sunset-thumb.jpg", "width": 300, "height": 200},
    "image": {"source": "https://up/sunset.jpg", "width": 3000, "height": 2000},
    "file_page": "https://commons.wikimedia.org/wiki/File:Sunset.jpg",
    "artist": {"html": "<a>Jane</a>", "text": "Jane"},
    "credit": {"html": "Own work"},
    "license": {"type": "CC BY-SA 4.0", "url": "https://creativecommons.org/licenses/by-sa/4.0"},
    "description": {"text": "A sunset", "html": "<p>A sunset</p>", "lang": "en"}
  },
  "news": [{
    "story": "<a href=\"x\">Something</a> happened.",
    "links": [{"title": "Something", "extract": "An event.", "content_urls": {"desktop": {"page": "https://en.wikipedia.org/wiki/Something"}}}]
  }],
  "onthisday": [{
    "text": "An event occurred.",
    "year": 1900,
    "pages": [{"title": "Event", "extract": "Details.", "thumbnail": {"source": "https://up/e.jpg"}}]
  }]
}`

func TestFeaturedDecodePreservesFullStructure(t *testing.T) {
	var f FeaturedFeed
	if err := json.Unmarshal([]byte(sampleFeatured), &f); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if f.TFA == nil || f.TFA.WikibaseItem != "Q7251" || f.TFA.PageID != 1208 {
		t.Errorf("tfa identity dropped: %+v", f.TFA)
	}
	if f.TFA.Thumbnail == nil || f.TFA.OriginalImage == nil || f.TFA.ExtractHTML == "" {
		t.Errorf("tfa media/extract dropped: %+v", f.TFA)
	}
	if f.TFA.ContentURLs == nil || f.TFA.ContentURLs.Mobile == nil || f.TFA.ContentURLs.Mobile.Page == "" {
		t.Errorf("tfa mobile url dropped: %+v", f.TFA.ContentURLs)
	}
	if f.TFA.URL() != "https://en.wikipedia.org/wiki/Alan_Turing" {
		t.Errorf("tfa URL() = %q", f.TFA.URL())
	}

	if f.MostRead == nil || f.MostRead.Date == "" || len(f.MostRead.Articles) != 1 {
		t.Fatalf("mostread dropped: %+v", f.MostRead)
	}
	if len(f.MostRead.Articles[0].ViewHistory) != 2 {
		t.Errorf("view_history dropped: %+v", f.MostRead.Articles[0])
	}

	if f.Image == nil || f.Image.License == nil || f.Image.License.URL == "" {
		t.Errorf("potd license dropped: %+v", f.Image)
	}
	if f.Image.Artist == nil || f.Image.Credit == nil || f.Image.FilePage == "" {
		t.Errorf("potd credit/file_page dropped: %+v", f.Image)
	}
	if f.Image.URL() != "https://up/sunset.jpg" || f.Image.DescriptionText() != "A sunset" {
		t.Errorf("potd accessors wrong: %q %q", f.Image.URL(), f.Image.DescriptionText())
	}

	if len(f.News) != 1 || len(f.News[0].Links) != 1 || f.News[0].Links[0].Extract == "" {
		t.Errorf("news link summary dropped: %+v", f.News)
	}
	if f.News[0].StoryText() != "Something happened." {
		t.Errorf("news StoryText() = %q", f.News[0].StoryText())
	}

	if len(f.OnThisDay) != 1 || len(f.OnThisDay[0].Pages) != 1 {
		t.Fatalf("onthisday pages dropped: %+v", f.OnThisDay)
	}
	if f.OnThisDay[0].Pages[0].Thumbnail == nil || f.OnThisDay[0].Year != 1900 {
		t.Errorf("onthisday page summary dropped: %+v", f.OnThisDay[0])
	}
	if got := f.OnThisDay[0].PageTitles(); len(got) != 1 || got[0] != "Event" {
		t.Errorf("PageTitles() = %v", got)
	}
}
