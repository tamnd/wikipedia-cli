package wiki

import (
	"strings"
	"testing"
)

const sampleWikitext = `<!-- a comment -->
{{Infobox person
| name = Ada
| note = {{nested|{{deep}}}}
}}
'''Ada Lovelace''' was an ''English'' [[mathematician]] known for the
[[Analytical Engine|Engine]].<ref name=a>cite</ref>

== Life ==

She worked with [[Charles Babbage]].<ref/> See [https://example.com Example].

=== Notes ===
* one
* two
** nested
# first
# second

{| class="wikitable"
|-
! Header
|-
| Cell
|}

<syntaxhighlight lang="python">
def hello():
    print("world")
</syntaxhighlight>

[[File:Ada.jpg|thumb|portrait]]
[[Category:Mathematicians]]
[[fr:Ada Lovelace]]
`

func TestWikitextToMarkdown(t *testing.T) {
	got := WikitextToMarkdown(sampleWikitext, "en")
	wants := []string{
		"**Ada Lovelace** was an *English* [mathematician](https://en.wikipedia.org/wiki/mathematician)",
		"Engine](https://en.wikipedia.org/wiki/Analytical_Engine)",
		"## Life",
		"### Notes",
		"- one",
		"  - nested",
		"1. first",
		"[Example](https://example.com)",
		"```python",
		`print("world")`,
	}
	for _, w := range wants {
		if !strings.Contains(got, w) {
			t.Errorf("markdown missing %q in:\n%s", w, got)
		}
	}
	for _, bad := range []string{"Infobox", "<ref", "File:", "Category:", "fr:", "{{", "}}", "{|", "|}", "wikitable"} {
		if strings.Contains(got, bad) {
			t.Errorf("markdown should not contain %q in:\n%s", bad, got)
		}
	}
}

func TestWikitextToText(t *testing.T) {
	got := WikitextToText("'''Bold''' and ''italic'' and [[link|label]] and [https://x.com Site].")
	if want := "Bold and italic and label and Site.\n"; got != want {
		t.Errorf("text = %q, want %q", got, want)
	}
}

func TestWikitextCodeBlockText(t *testing.T) {
	got := WikitextToText("intro\n<syntaxhighlight lang=\"go\">\nfmt.Println(1)\n</syntaxhighlight>\nend")
	if !strings.Contains(got, "fmt.Println(1)") {
		t.Errorf("code body lost: %q", got)
	}
	if strings.Contains(got, "syntaxhighlight") || strings.Contains(got, "```") {
		t.Errorf("text should not keep code markup: %q", got)
	}
}

func TestWikitextLinkInsideEmphasis(t *testing.T) {
	got := WikitextToMarkdown("the ''[[Journal|Journal]]'' of record", "en")
	want := "*[Journal](https://en.wikipedia.org/wiki/Journal)*"
	if !strings.Contains(got, want) {
		t.Errorf("link inside italic not converted: got %q", got)
	}
	if strings.Contains(got, "[[") {
		t.Errorf("raw brackets leaked: %q", got)
	}
}

func TestStripCommonNested(t *testing.T) {
	got := stripCommon("a {{x|{{y|{{z}}}}}} b")
	if strings.Contains(got, "{{") || strings.Contains(got, "}}") {
		t.Errorf("nested templates not removed: %q", got)
	}
	if !strings.Contains(got, "a ") || !strings.Contains(got, " b") {
		t.Errorf("surrounding text lost: %q", got)
	}
}

func TestStripCommonUnclosed(t *testing.T) {
	if got := stripCommon("before {{ unclosed"); !strings.Contains(got, "{{") {
		t.Errorf("unclosed template should pass through: %q", got)
	}
}

func TestIsStripLinkPrefix(t *testing.T) {
	for _, p := range []string{"file", "image", "category", "fichier", "datei", "fr", "ja", "zh"} {
		if !isStripLinkPrefix(p) {
			t.Errorf("expected %q to strip", p)
		}
	}
	for _, p := range []string{"a", "abcdefghijk"} {
		if isStripLinkPrefix(p) {
			t.Errorf("expected %q not to strip", p)
		}
	}
}
