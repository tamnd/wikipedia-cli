package wiki

import "testing"

func TestSiteResolution(t *testing.T) {
	cases := []struct {
		lang, project, host string
		wantHost            string
		wantErr             bool
	}{
		{"en", "wikipedia", "", "en.wikipedia.org", false},
		{"de", "wikipedia", "", "de.wikipedia.org", false},
		{"en", "wiktionary", "", "en.wiktionary.org", false},
		{"", "commons", "", "commons.wikimedia.org", false},
		{"", "wikidata", "", "www.wikidata.org", false},
		{"en", "nope", "", "", true},
		{"en", "wikipedia", "fr.wikipedia.org", "fr.wikipedia.org", false},
		{"en", "wikipedia", "evil.example.com", "", true},
	}
	for _, tc := range cases {
		cfg := DefaultConfig()
		cfg.Lang, cfg.Project, cfg.SiteHost = tc.lang, tc.project, tc.host
		site, err := cfg.Site()
		if tc.wantErr {
			if err == nil {
				t.Errorf("Site(%v) expected error", tc)
			}
			continue
		}
		if err != nil {
			t.Errorf("Site(%v) error: %v", tc, err)
			continue
		}
		if site.Host != tc.wantHost {
			t.Errorf("Site(%v).Host = %q, want %q", tc, site.Host, tc.wantHost)
		}
	}
}

func TestPageURL(t *testing.T) {
	cfg := DefaultConfig()
	site, _ := cfg.Site()
	if got := site.PageURL("Alan Turing"); got != "https://en.wikipedia.org/wiki/Alan_Turing" {
		t.Errorf("PageURL = %q", got)
	}
}

func TestDumpWiki(t *testing.T) {
	cfg := DefaultConfig()
	site, _ := cfg.Site()
	if got := site.DumpWiki(); got != "enwiki" {
		t.Errorf("DumpWiki = %q", got)
	}
	cfg.Lang, cfg.Project = "fr", "wiktionary"
	site, _ = cfg.Site()
	if got := site.DumpWiki(); got != "frwiktionary" {
		t.Errorf("DumpWiki = %q", got)
	}
}

func TestCoreURL(t *testing.T) {
	cfg := DefaultConfig()
	site, _ := cfg.Site()
	u, ok := site.CoreURL("search/page")
	if !ok || u != "https://api.wikimedia.org/core/v1/wikipedia/en/search/page" {
		t.Errorf("CoreURL = %q ok=%v", u, ok)
	}
	cfg.Project, cfg.Lang = "commons", ""
	site, _ = cfg.Site()
	if _, ok := site.CoreURL("search/page"); ok {
		t.Error("CoreURL should be unavailable for commons")
	}
}
