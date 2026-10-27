package wiki

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestSourceResponse(t *testing.T) {
	body := `{"batchcomplete":true,"query":{"redirects":[{"from":"Sylow theorem","to":"Sylow theorems"}],"pages":[{"pageid":29243,"ns":0,"title":"Sylow theorems","revisions":[{"revid":1250000000,"parentid":1249999999,"timestamp":"2026-10-01T10:00:00Z","sha1":"0123456789abcdef0123456789abcdef01234567","size":34292,"slots":{"main":{"contentmodel":"wikitext","contentformat":"text/x-wiki","content":"In [[mathematics]], ..."}}}]}]}}`
	var r sourceResponse
	if err := json.Unmarshal([]byte(body), &r); err != nil {
		t.Fatal(err)
	}
	s, err := r.source()
	if err != nil {
		t.Fatal(err)
	}
	want := Source{PageID: 29243, Title: "Sylow theorems", RevID: 1250000000, ParentID: 1249999999, Timestamp: "2026-10-01T10:00:00Z", SHA1: "0123456789abcdef0123456789abcdef01234567", Size: 34292, Content: "In [[mathematics]], ..."}
	if *s != want {
		t.Errorf("got %+v, want %+v", *s, want)
	}
}

func TestSourceResponseMissing(t *testing.T) {
	for _, body := range []string{
		`{"query":{"pages":[{"ns":0,"title":"No such page","missing":true}]}}`,
		`{"query":{"pages":[]}}`,
	} {
		var r sourceResponse
		if err := json.Unmarshal([]byte(body), &r); err != nil {
			t.Fatal(err)
		}
		if _, err := r.source(); !errors.Is(err, ErrNotFound) {
			t.Errorf("%s: got %v, want ErrNotFound", body, err)
		}
	}
	var r sourceResponse
	_ = json.Unmarshal([]byte(`{"error":{"code":"maxlag","info":"Waiting for a database server"}}`), &r)
	var ae *APIError
	if _, err := r.source(); !errors.As(err, &ae) || ae.Code != "maxlag" {
		t.Errorf("got %v, want the API error", err)
	}
}
