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
	if p.Title != "Alan Turing" || p.ID != 1208 || p.RevID != 555 {
		t.Errorf("bad page: %+v", p)
	}
	if p.Text != "Turing was a mathematician." {
		t.Errorf("bad text: %q", p.Text)
	}
}

func TestStreamPagesAllNamespaces(t *testing.T) {
	n := 0
	err := streamPagesReader(strings.NewReader(sampleDump), -1, false, func(p DumpPage) error {
		n++
		if p.Text != "" {
			t.Errorf("withText=false but got text: %q", p.Text)
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
