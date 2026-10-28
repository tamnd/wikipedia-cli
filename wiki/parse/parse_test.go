package parse

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var update = flag.Bool("update", false, "write the golden files of set T")

// TestSetT runs each file of set T and compares the tree with its golden
// file. Run "go test ./wiki/parse -update" to write the golden files after a
// change that a person checked.
func TestSetT(t *testing.T) {
	files, _ := filepath.Glob("testdata/T/*.wiki")
	if len(files) == 0 {
		t.Fatal("no files in set T")
	}
	for _, f := range files {
		t.Run(filepath.Base(f), func(t *testing.T) {
			src, err := os.ReadFile(f)
			if err != nil {
				t.Fatal(err)
			}
			doc, err := Parse(string(src))
			if err != nil {
				t.Fatal(err)
			}
			checkSpans(t, doc.Root, 0, len(src))
			got := doc.Dump()
			golden := strings.TrimSuffix(f, ".wiki") + ".tree"
			if *update {
				if err := os.WriteFile(golden, []byte(got), 0o644); err != nil {
					t.Fatal(err)
				}
				return
			}
			want, err := os.ReadFile(golden)
			if err != nil {
				t.Fatalf("%v (run with -update)", err)
			}
			if got != string(want) {
				t.Errorf("tree differs from %s:\n%s", golden, got)
			}
		})
	}
}

// checkSpans checks that each node is inside its parent and that children
// are in order.
func checkSpans(t *testing.T, n *Node, lo, hi int) {
	t.Helper()
	if n.Start < lo || n.End > hi || n.Start > n.End {
		t.Errorf("%s %d-%d is outside %d-%d", n.Kind, n.Start, n.End, lo, hi)
	}
	prev := n.Start
	for _, c := range n.Children {
		if c.Start < prev {
			t.Errorf("%s %d-%d starts before %d", c.Kind, c.Start, c.End, prev)
		}
		checkSpans(t, c, n.Start, n.End)
		prev = c.End
	}
	for _, p := range n.Params {
		for _, c := range p.Value {
			checkSpans(t, c, p.Start, p.End)
		}
	}
}

func parseT(t *testing.T, src string) *Document {
	t.Helper()
	doc, err := Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	return doc
}

func TestTemplateParams(t *testing.T) {
	doc := parseT(t, "{{math theorem|name=Lagrange's theorem|math_statement=If {{mvar|H}} is a [[subgroup]] | x |note= a=b }}")
	tpl := Find(doc.Root, KindTemplate)[0]
	if TemplateKey(tpl.Name) != "math theorem" {
		t.Errorf("name %q", tpl.Name)
	}
	if got := doc.ParamValue(tpl.Param("name")); got != "Lagrange's theorem" {
		t.Errorf("name param %q", got)
	}
	if got := doc.ParamValue(tpl.Param("math_statement")); got != "If {{mvar|H}} is a [[subgroup]]" {
		t.Errorf("statement %q", got)
	}
	if got := doc.ParamValue(tpl.Param("1")); got != "x" {
		t.Errorf("positional %q", got)
	}
	if got := doc.ParamValue(tpl.Param("note")); got != "a=b" {
		t.Errorf("note %q", got)
	}
	inner := Find(&Node{Children: tpl.Param("math_statement").Value}, KindTemplate)
	if len(inner) != 1 || inner[0].Name != "mvar" {
		t.Errorf("nested template %+v", inner)
	}
}

func TestPlainAndAnchor(t *testing.T) {
	doc := parseT(t, "== Proof of [[Lagrange's theorem|the theorem]] for <math>G</math>s<ref>x</ref> ==\n")
	h := doc.HeadingText(doc.Root.Children[1])
	if h != "Proof of the theorem for Gs" {
		t.Errorf("heading %q", h)
	}
	if a := Anchor(h); a != "Proof_of_the_theorem_for_Gs" {
		t.Errorf("anchor %q", a)
	}
}

func TestTemplateKey(t *testing.T) {
	for in, want := range map[string]string{
		"Math_theorem": "math theorem", "Template:Math  proof": "math proof", " mvar ": "mvar",
	} {
		if got := TemplateKey(in); got != want {
			t.Errorf("%q: got %q", in, got)
		}
	}
}

func TestLimits(t *testing.T) {
	if _, err := ParseOptions(strings.Repeat("a", 100), Options{MaxBytes: 10}); err != ErrTooLarge {
		t.Errorf("want ErrTooLarge, got %v", err)
	}
	// Many unclosed constructs must not make the parse slow.
	src := strings.Repeat("{{a|[[b|<ref>c ", 20000)
	start := time.Now()
	if _, err := Parse(src); err != nil {
		t.Fatal(err)
	}
	if d := time.Since(start); d > 3*time.Second {
		t.Errorf("parse took %v", d)
	}
}

// TestTagOffsetsAfterUnicode checks the tags after a letter whose lower case
// has another byte length, as İ. The offsets of the tags must not move.
func TestTagOffsetsAfterUnicode(t *testing.T) {
	src := strings.Repeat("İstanbul ", 20) + "a<ref>b</ref> c<REF name=x>d</REF>"
	doc, err := Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	checkSpans(t, doc.Root, 0, len(src))
	var refs []string
	Walk(doc.Root, func(n *Node) bool {
		if n.Kind == KindTag && n.Name == "ref" {
			refs = append(refs, src[n.InnerStart:n.InnerEnd])
		}
		return true
	})
	if strings.Join(refs, ",") != "b,d" {
		t.Errorf("refs %q", refs)
	}
}

func FuzzParse(f *testing.F) {
	files, _ := filepath.Glob("testdata/T/*.wiki")
	for _, p := range files {
		b, _ := os.ReadFile(p)
		f.Add(string(b))
	}
	f.Fuzz(func(t *testing.T, src string) {
		doc, err := ParseOptions(src, Options{Timeout: -1})
		if err != nil {
			t.Fatal(err)
		}
		checkSpans(t, doc.Root, 0, len(src))
	})
}
