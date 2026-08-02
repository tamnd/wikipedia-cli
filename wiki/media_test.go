package wiki

import (
	"encoding/json"
	"testing"
)

// A single page from a query+imageinfo response, trimmed but representative: a
// description URL, media type, sha1, embedded metadata, a thumbnail, and an
// extmetadata block whose entries carry source and hidden markers. All of it
// must survive the decode.
const sampleMediaPage = `{
  "pageid": 12345,
  "ns": 6,
  "title": "File:Alan Turing.jpg",
  "imagerepository": "shared",
  "imageinfo": [{
    "timestamp": "2007-03-11T09:00:00Z",
    "user": "Photographer",
    "userid": 99,
    "size": 84000,
    "width": 800,
    "height": 1000,
    "comment": "upload",
    "url": "https://upload.wikimedia.org/turing.jpg",
    "descriptionurl": "https://commons.wikimedia.org/wiki/File:Alan_Turing.jpg",
    "descriptionshorturl": "https://commons.wikimedia.org/w/index.php?curid=12345",
    "thumburl": "https://upload.wikimedia.org/thumb/turing.jpg",
    "thumbwidth": 240,
    "thumbheight": 300,
    "mime": "image/jpeg",
    "mediatype": "BITMAP",
    "bitdepth": 8,
    "sha1": "deadbeef",
    "canonicaltitle": "File:Alan Turing.jpg",
    "commonmetadata": [{"name": "ImageWidth", "value": 800}],
    "metadata": [{"name": "Make", "value": "Leica"}],
    "extmetadata": {
      "LicenseShortName": {"value": "CC BY-SA 4.0", "source": "commons-desc-page"},
      "Artist": {"value": "<a href=\"/wiki/User:X\">X</a>", "source": "commons-desc-page"},
      "DateTimeOriginal": {"value": "1951", "source": "commons-desc-page", "hidden": ""}
    }
  }]
}`

func TestMediaDecodePreservesFullStructure(t *testing.T) {
	var m Media
	if err := json.Unmarshal([]byte(sampleMediaPage), &m); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if m.PageID != 12345 || m.NS != 6 || m.ImageRepository != "shared" {
		t.Errorf("page identity dropped: %+v", m)
	}
	ii := m.Current()
	if ii == nil {
		t.Fatal("no imageinfo decoded")
	}
	if ii.DescriptionURL == "" || ii.DescriptionShort == "" {
		t.Errorf("description urls dropped: %+v", ii)
	}
	if ii.MediaType != "BITMAP" || ii.Sha1 != "deadbeef" || ii.BitDepth != 8 {
		t.Errorf("media type/sha1/bitdepth dropped: %+v", ii)
	}
	if ii.ThumbURL == "" || ii.ThumbWidth != 240 || ii.ThumbHeight != 300 {
		t.Errorf("thumbnail dropped: %+v", ii)
	}
	if ii.User != "Photographer" || ii.UserID != 99 || ii.Timestamp == "" {
		t.Errorf("uploader metadata dropped: %+v", ii)
	}
	if !json.Valid(ii.Metadata) || !json.Valid(ii.CommonMetadata) {
		t.Errorf("embedded metadata dropped")
	}

	// Every extmetadata entry, with its source and hidden flag, must remain.
	if len(ii.ExtMetadata) != 3 {
		t.Fatalf("extmetadata entries dropped: %d", len(ii.ExtMetadata))
	}
	if got, ok := ii.ExtMetadata["DateTimeOriginal"]; !ok || got.Source != "commons-desc-page" {
		t.Errorf("extmetadata source dropped: %+v", got)
	}

	// The convenience accessors read the current revision's extmetadata.
	if m.License() != "CC BY-SA 4.0" {
		t.Errorf("License() = %q", m.License())
	}
	if m.Author() != "X" {
		t.Errorf("Author() = %q (HTML should be stripped)", m.Author())
	}
	if m.URL() != "https://upload.wikimedia.org/turing.jpg" || m.Mime() != "image/jpeg" {
		t.Errorf("URL/Mime accessors wrong: %q %q", m.URL(), m.Mime())
	}
	if m.Width() != 800 || m.Height() != 1000 || m.Size() != 84000 {
		t.Errorf("dimension accessors wrong: %d %d %d", m.Width(), m.Height(), m.Size())
	}
}
