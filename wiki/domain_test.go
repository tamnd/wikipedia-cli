package wiki_test

import (
	"testing"

	"github.com/tamnd/any-cli/kit"
	"github.com/tamnd/wikipedia-cli/wiki"
)

// These tests exercise the kit driver wiring without any network: the blank
// import below registers the domain, and Mint/Resolve/Locate are pure string and
// reflection work over the registry.

func TestDomainInfo(t *testing.T) {
	info := wiki.Domain{}.Info()
	if info.Scheme != "wikipedia" {
		t.Errorf("scheme = %q, want wikipedia", info.Scheme)
	}
	if len(info.Aliases) == 0 || info.Aliases[0] != "wiki" {
		t.Errorf("aliases = %v, want [wiki]", info.Aliases)
	}
}

func TestClassify(t *testing.T) {
	d := wiki.Domain{}
	cases := []struct {
		in, typ, id string
	}{
		{"Albert Einstein", "page", "Albert Einstein"},
		{"Albert_Einstein", "page", "Albert Einstein"},
		{"https://en.wikipedia.org/wiki/Albert_Einstein", "page", "Albert Einstein"},
		{"https://en.wikipedia.org/w/index.php?title=Earth", "page", "Earth"},
		{"Category:Physics", "category", "Physics"},
		{"https://en.wikipedia.org/wiki/Category:Physics", "category", "Physics"},
	}
	for _, c := range cases {
		typ, id, err := d.Classify(c.in)
		if err != nil || typ != c.typ || id != c.id {
			t.Errorf("Classify(%q) = %q/%q/%v, want %q/%q", c.in, typ, id, err, c.typ, c.id)
		}
	}
	if _, _, err := d.Classify("  "); err == nil {
		t.Error("Classify(blank) = nil error, want error")
	}
}

func TestLocate(t *testing.T) {
	d := wiki.Domain{}
	loc, err := d.Locate("page", "Albert Einstein")
	if err != nil || loc != "https://en.wikipedia.org/wiki/Albert_Einstein" {
		t.Errorf("Locate(page) = %q/%v", loc, err)
	}
	loc, err = d.Locate("category", "Physics")
	if err != nil || loc != "https://en.wikipedia.org/wiki/Category:Physics" {
		t.Errorf("Locate(category) = %q/%v", loc, err)
	}
	if _, err := d.Locate("nonsense", "x"); err == nil {
		t.Error("Locate(nonsense) = nil error, want error")
	}
}

func TestHostMintResolve(t *testing.T) {
	h, err := kit.Open()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := h.Domain("wikipedia"); !ok {
		t.Fatal("wikipedia not mounted on host")
	}

	page := &wiki.Summary{Title: "Albert Einstein", Extract: "A physicist.", URL: "https://en.wikipedia.org/wiki/Albert_Einstein"}
	minted, err := h.Mint(page)
	if err != nil || minted.String() != "wikipedia://page/Albert%20Einstein" {
		t.Errorf("Mint(page) = %q/%v", minted.String(), err)
	}
	if body, ok := h.Body(page); !ok || body != "A physicist." {
		t.Errorf("Body = %q/%v, want the extract", body, ok)
	}

	u, err := h.ResolveOn("wikipedia", "Albert_Einstein")
	if err != nil || u.String() != "wikipedia://page/Albert%20Einstein" {
		t.Errorf("ResolveOn(bare) = %q/%v", u.String(), err)
	}
	u, err = h.Resolve("https://en.wikipedia.org/wiki/Earth")
	if err != nil || u.String() != "wikipedia://page/Earth" {
		t.Errorf("Resolve(url) = %q/%v", u.String(), err)
	}

	// The wiki alias resolves to the same canonical scheme.
	u, err = h.ResolveOn("wiki", "Earth")
	if err != nil || u.String() != "wikipedia://page/Earth" {
		t.Errorf("ResolveOn(alias) = %q/%v", u.String(), err)
	}
}
