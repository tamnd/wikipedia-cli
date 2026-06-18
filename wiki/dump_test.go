package wiki

import (
	"strings"
	"testing"
)

const sampleDump = `<mediawiki>
  <page>
    <title>Alan Turing</title>
    <ns>0</ns>
    <id>1208</id>
    <revision>
      <id>555</id>
      <timestamp>2026-01-02T03:04:05Z</timestamp>
      <text>Turing was a mathematician.</text>
    </revision>
  </page>
  <page>
    <title>Talk:Alan Turing</title>
    <ns>1</ns>
    <id>1209</id>
    <revision>
      <id>556</id>
      <timestamp>2026-01-02T03:05:05Z</timestamp>
      <text>Discussion.</text>
    </revision>
  </page>
</mediawiki>`

func TestStreamPagesReader(t *testing.T) {
	var pages []DumpPage
	err := streamPagesReader(strings.NewReader(sampleDump), 0, true, func(p DumpPage) error {
		pages = append(pages, p)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(pages) != 1 {
		t.Fatalf("namespace filter failed: got %d pages", len(pages))
	}
	p := pages[0]
	r := p.Latest()
	if p.Title != "Alan Turing" || p.ID != 1208 || r == nil || r.ID != 555 {
		t.Errorf("bad page: %+v", p)
	}
	if p.LatestText() != "Turing was a mathematician." {
		t.Errorf("bad text: %q", p.LatestText())
	}
}

func TestStreamPagesAllNamespaces(t *testing.T) {
	n := 0
	err := streamPagesReader(strings.NewReader(sampleDump), -1, false, func(p DumpPage) error {
		n++
		if p.LatestText() != "" {
			t.Errorf("withText=false but got text: %q", p.LatestText())
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Errorf("expected 2 pages, got %d", n)
	}
}

func TestStreamPagesStop(t *testing.T) {
	n := 0
	err := streamPagesReader(strings.NewReader(sampleDump), -1, false, func(p DumpPage) error {
		n++
		return errStop
	})
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Errorf("errStop did not stop: %d", n)
	}
}

// A history dump page carries every revision with its full metadata. None of
// the contributor, redirect, restriction or per-revision fields may be lost.
const historyDump = `<mediawiki>
  <page>
    <title>Quantum mechanics</title>
    <ns>0</ns>
    <id>25202</id>
    <redirect title="Quantum physics" />
    <restrictions>edit=sysop:move=sysop</restrictions>
    <revision>
      <id>100</id>
      <timestamp>2020-01-01T00:00:00Z</timestamp>
      <contributor>
        <username>Alice</username>
        <id>42</id>
      </contributor>
      <minor />
      <comment>first</comment>
      <model>wikitext</model>
      <format>text/x-wiki</format>
      <origin>100</origin>
      <sha1>abc</sha1>
      <text bytes="9">old text.</text>
    </revision>
    <revision>
      <id>101</id>
      <parentid>100</parentid>
      <timestamp>2021-02-03T04:05:06Z</timestamp>
      <contributor>
        <ip>10.0.0.1</ip>
      </contributor>
      <comment>second</comment>
      <model>wikitext</model>
      <format>text/x-wiki</format>
      <origin>101</origin>
      <sha1>def</sha1>
      <text bytes="9">new text.</text>
    </revision>
  </page>
</mediawiki>`

func TestStreamPagesFullRevisionMetadata(t *testing.T) {
	var pages []DumpPage
	err := streamPagesReader(strings.NewReader(historyDump), -1, true, func(p DumpPage) error {
		pages = append(pages, p)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(pages) != 1 {
		t.Fatalf("got %d pages", len(pages))
	}
	p := pages[0]
	if !p.Redirect || p.RedirectTitle != "Quantum physics" {
		t.Errorf("redirect dropped: %v %q", p.Redirect, p.RedirectTitle)
	}
	if p.Restrictions != "edit=sysop:move=sysop" {
		t.Errorf("restrictions dropped: %q", p.Restrictions)
	}
	if len(p.Revisions) != 2 {
		t.Fatalf("expected 2 revisions, got %d", len(p.Revisions))
	}

	first := p.Revisions[0]
	if first.ID != 100 || !first.Minor || first.Comment != "first" || first.Sha1 != "abc" {
		t.Errorf("first revision metadata dropped: %+v", first)
	}
	if first.Model != "wikitext" || first.Format != "text/x-wiki" || first.Origin != 100 || first.TextBytes != 9 {
		t.Errorf("first revision content metadata dropped: %+v", first)
	}
	if first.Contributor == nil || first.Contributor.Username != "Alice" || first.Contributor.ID != 42 {
		t.Errorf("registered contributor dropped: %+v", first.Contributor)
	}

	second := p.Revisions[1]
	if second.ParentID != 100 {
		t.Errorf("parentid dropped: %+v", second)
	}
	if second.Contributor == nil || second.Contributor.IP != "10.0.0.1" {
		t.Errorf("anonymous contributor dropped: %+v", second.Contributor)
	}

	if r := p.Latest(); r == nil || r.ID != 101 {
		t.Errorf("Latest did not return the last revision: %+v", r)
	}
	if p.LatestText() != "new text." {
		t.Errorf("LatestText = %q", p.LatestText())
	}
}
