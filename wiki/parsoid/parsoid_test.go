package parsoid

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestRead(t *testing.T) {
	f, err := os.Open("testdata/widget.html")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	r, err := Read(f)
	if err != nil {
		t.Fatal(err)
	}
	if r.Page != 12345 || r.Revision != 678 || r.SHA1 != "abc123" || r.Refs != 1 || r.TeXHTML != 2 {
		t.Errorf("reading = %+v", r)
	}
	want := []Math{{TeX: "W", AltText: `{\displaystyle W}`}, {TeX: "m", AltText: `{\displaystyle m}`}, {TeX: `\Z`, AltText: `{\displaystyle \mathbb {Z} }`, Transcluded: true}, {TeX: "f(x) = x", AltText: `{\displaystyle f(x)=x}`, Block: true}, {TeX: `\mathrm {H_2O}`, AltText: `{\displaystyle \mathrm {H_2O} }`}}
	if !slices.Equal(r.Math, want) {
		t.Errorf("math = %+v", r.Math)
	}
	if len(r.Transclusions) != 3 {
		t.Fatalf("transclusions = %+v", r.Transclusions)
	}
	tr := r.Transclusions[2]
	if len(tr.Templates) != 2 || tr.Templates[0].Name != "math theorem" || tr.Templates[0].Target != "Template:Math_theorem" || tr.Templates[0].Params["name"] != "Widget lemma" || tr.Templates[1].Name != "anchor" {
		t.Errorf("transclusion = %+v", tr)
	}
	if want := []Section{{ID: 0}, {ID: 1, Level: 2, Heading: "Definition", Anchor: "Definition"}}; !slices.Equal(r.Sections, want) {
		t.Errorf("sections = %+v", r.Sections)
	}
	if want := []string{"Definition", "Widget_lemma"}; !slices.Equal(r.Anchors, want) {
		t.Errorf("anchors = %v", r.Anchors)
	}
}

func TestFetch(t *testing.T) {
	tries := 0
	var paths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.UserAgent() != "test/1.0" {
			http.Error(w, "no user agent", http.StatusForbidden)
			return
		}
		paths = append(paths, r.URL.EscapedPath())
		switch {
		case strings.HasSuffix(r.URL.Path, "/1"):
			http.NotFound(w, r)
		case tries == 0:
			tries++
			w.Header().Set("Retry-After", "0")
			http.Error(w, "slow down", http.StatusTooManyRequests)
		default:
			_, _ = w.Write([]byte("<html></html>"))
		}
	}))
	defer srv.Close()
	f := &Fetcher{Base: srv.URL, UserAgent: "test/1.0"}
	start := time.Now()
	body, err := f.Fetch(context.Background(), "Lagrange's theorem (group theory)", 678)
	if err != nil || string(body) != "<html></html>" {
		t.Fatalf("fetch = %q, %v", body, err)
	}
	if _, err := f.Fetch(context.Background(), "Widget", 1); !errors.Is(err, ErrNotFound) {
		t.Errorf("the error of a missing revision is %v", err)
	}
	// Three requests take at least two intervals.
	if d := time.Since(start); d < 2*MinInterval {
		t.Errorf("three requests took %v", d)
	}
	if n, _ := f.Rate(); n != 3 {
		t.Errorf("the fetcher counts %d requests", n)
	}
	if paths[0] != "/page/html/Lagrange%27s_theorem_%28group_theory%29/678" {
		t.Errorf("the path is %s", paths[0])
	}
	if _, err := (&Fetcher{Base: srv.URL}).Fetch(context.Background(), "Widget", 2); err == nil {
		t.Error("a fetcher with no User-Agent fetches")
	}
}

func TestStripDisplay(t *testing.T) {
	for in, want := range map[string]string{`{\displaystyle x^2}`: "x^2", `{\textstyle a}`: "a", "b": "b"} {
		if got := stripDisplay(in); got != want {
			t.Errorf("stripDisplay(%q) = %q", in, got)
		}
	}
}
