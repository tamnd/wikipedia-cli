package wiki

import (
	"strings"
	"testing"
)

func TestHTMLToText(t *testing.T) {
	in := `<h2>History</h2><p>Alan <b>Turing</b> was a <a href="/wiki/Mathematician">mathematician</a>.` +
		`<sup class="reference">[1]</sup></p><ul><li>One</li><li>Two</li></ul>`
	got := HTMLToText(in)
	if !strings.Contains(got, "History") {
		t.Errorf("missing heading: %q", got)
	}
	if !strings.Contains(got, "Alan Turing was a mathematician.") {
		t.Errorf("paragraph wrong: %q", got)
	}
	if strings.Contains(got, "[1]") {
		t.Errorf("citation marker not dropped: %q", got)
	}
	if !strings.Contains(got, "- One") || !strings.Contains(got, "- Two") {
		t.Errorf("list items wrong: %q", got)
	}
}

func TestHTMLToMarkdown(t *testing.T) {
	in := `<h2>History</h2><p>A <a href="https://example.com">link</a> and <b>bold</b>.</p>`
	got := HTMLToMarkdown(in)
	if !strings.Contains(got, "## History") {
		t.Errorf("heading not markdown: %q", got)
	}
	if !strings.Contains(got, "[link](https://example.com)") {
		t.Errorf("link not markdown: %q", got)
	}
	if !strings.Contains(got, "**bold**") {
		t.Errorf("bold not markdown: %q", got)
	}
}

func TestHTMLToMarkdownList(t *testing.T) {
	in := `<ol><li>First</li><li>Second</li></ol>`
	got := HTMLToMarkdown(in)
	if !strings.Contains(got, "1. First") || !strings.Contains(got, "2. Second") {
		t.Errorf("ordered list wrong: %q", got)
	}
}
