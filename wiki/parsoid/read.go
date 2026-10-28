package parsoid

import (
	"encoding/json"
	"io"
	"slices"
	"strconv"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// Reading is what the Parsoid HTML of one revision gives.
type Reading struct {
	Page     int    // the page id, from mw:pageId
	Revision int    // the revision id, from the about attribute of <html>
	SHA1     string // the sha1 of the revision, from mw:revisionSHA1

	Math          []Math
	Transclusions []Transclusion
	Sections      []Section
	// Anchors are the ids of the headings and of the anchor spans, after the
	// expansion of the templates, in the order of the page.
	Anchors []string
	Refs    int // the number of reference marks
	// TeXHTML is the number of formulas in HTML, such as the output of the
	// templates math and mvar: the spans of the class texhtml. A span that
	// the body of a template made does not count.
	TeXHTML int
}

// Math is one formula of <math>, <chem> or <ce>.
type Math struct {
	TeX     string // the source of the tag, or the alttext without \displaystyle
	AltText string // the TeX that MediaWiki rendered
	Block   bool
	// Transcluded is true when the body of a template made the formula, so
	// the wikitext of the page does not have it. A formula in a parameter of
	// a template, or in the text between the templates of one transclusion,
	// is in the wikitext of the page, so it is not transcluded.
	Transcluded bool
}

// Transclusion is one template call of the page: the parts of one data-mw.
// A transclusion can join several templates, such as {{a}}{{b}} with no
// text between them.
type Transclusion struct {
	Templates []Template
}

// Template is one template of a transclusion.
type Template struct {
	Name   string // the name as written, such as "Math theorem"
	Target string // the page of the template, such as "Template:Math_theorem"
	Params map[string]string
}

// Section is one section of the page. The section 0 is the lead.
type Section struct {
	ID      int
	Level   int // 2 for ==, and 0 for the lead
	Heading string
	Anchor  string
}

type dataMW struct {
	Name string `json:"name"`
	Body *struct {
		Extsrc string `json:"extsrc"`
	} `json:"body"`
	Parts []json.RawMessage `json:"parts"`
}

type part struct {
	Template *struct {
		Target struct {
			WT   string `json:"wt"`
			Href string `json:"href"`
		} `json:"target"`
		Params map[string]struct {
			WT string `json:"wt"`
		} `json:"params"`
	} `json:"template"`
}

// Read reads the Parsoid HTML of one revision.
func Read(r io.Reader) (*Reading, error) {
	doc, err := html.Parse(r)
	if err != nil {
		return nil, err
	}
	out := &Reading{}
	// abouts gives the wikitext of the parameters of each transclusion, by
	// its about id. Parsoid gives each node of the output of a template the
	// about id of the transclusion, and the first node, which has the
	// data-mw, comes first in the order of the document.
	abouts := map[string]*string{}
	var walk func(n *html.Node, params *string)
	walk = func(n *html.Node, params *string) {
		if n.Type == html.ElementNode {
			if slices.Contains(strings.Fields(attr(n, "typeof")), "mw:Transclusion") {
				if a := attr(n, "about"); a != "" && abouts[a] == nil {
					wt := paramsText(attr(n, "data-mw"))
					abouts[a] = &wt
				}
			}
			if p := abouts[attr(n, "about")]; params == nil && p != nil {
				params = p
			}
			out.element(n, params)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c, params)
		}
	}
	walk(doc, nil)
	return out, nil
}

// element reads one element. params is the wikitext of the parameters of
// the transclusion when the element is in the output of a template.
func (out *Reading) element(n *html.Node, params *string) {
	switch n.DataAtom {
	case atom.Html:
		if a := attr(n, "about"); a != "" {
			out.Revision, _ = strconv.Atoi(a[strings.LastIndexByte(a, '/')+1:])
		}
	case atom.Meta:
		switch attr(n, "property") {
		case "mw:pageId":
			out.Page, _ = strconv.Atoi(attr(n, "content"))
		case "mw:revisionSHA1":
			out.SHA1 = attr(n, "content")
		}
	case atom.Section:
		if id, err := strconv.Atoi(attr(n, "data-mw-section-id")); err == nil && id >= 0 {
			s := Section{ID: id}
			if h := firstHeading(n); h != nil {
				s.Level = int(h.Data[1] - '0')
				s.Heading = strings.TrimSpace(text(h))
				s.Anchor = attr(h, "id")
			}
			out.Sections = append(out.Sections, s)
		}
	case atom.H1, atom.H2, atom.H3, atom.H4, atom.H5, atom.H6:
		if id := attr(n, "id"); id != "" {
			out.Anchors = append(out.Anchors, id)
		}
	}
	types := strings.Fields(attr(n, "typeof"))
	classes := strings.Fields(attr(n, "class"))
	if slices.Contains(classes, "texhtml") && (params == nil || slices.Contains(types, "mw:Transclusion") || strings.Contains(*params, "{{")) {
		out.TeXHTML++
	}
	if slices.Contains(classes, "anchor") {
		if id := attr(n, "id"); id != "" {
			out.Anchors = append(out.Anchors, id)
		}
	}
	if len(types) == 0 {
		return
	}
	var mw dataMW
	if d := attr(n, "data-mw"); d != "" {
		_ = json.Unmarshal([]byte(d), &mw)
	}
	for _, t := range types {
		switch t {
		case "mw:Extension/math", "mw:Extension/chem", "mw:Extension/ce":
			m := Math{Block: slices.Contains(strings.Fields(attr(n, "class")), "mwe-math-element-block")}
			if e := find(n, atom.Math); e != nil {
				m.AltText = attr(e, "alttext")
				m.Block = m.Block || attr(e, "display") == "block"
			}
			if mw.Body != nil {
				m.TeX = mw.Body.Extsrc
			} else {
				m.TeX = stripDisplay(m.AltText)
			}
			m.Transcluded = params != nil && !strings.Contains(*params, m.TeX)
			out.Math = append(out.Math, m)
		case "mw:Extension/ref":
			out.Refs++
		case "mw:Transclusion":
			// Each node of a transclusion has the typeof, and only the first
			// node has the data-mw.
			if len(mw.Parts) == 0 {
				continue
			}
			var tr Transclusion
			for _, raw := range mw.Parts {
				var p part
				if json.Unmarshal(raw, &p) != nil || p.Template == nil {
					continue
				}
				t := Template{Name: strings.TrimSpace(p.Template.Target.WT), Target: strings.TrimPrefix(p.Template.Target.Href, "./"), Params: map[string]string{}}
				for k, v := range p.Template.Params {
					t.Params[k] = v.WT
				}
				tr.Templates = append(tr.Templates, t)
			}
			if len(tr.Templates) > 0 {
				out.Transclusions = append(out.Transclusions, tr)
			}
		}
	}
}

// paramsText gives the wikitext of the page in a data-mw, joined: the
// parameters of the templates, and the text between the templates of a
// transclusion with several parts.
func paramsText(d string) string {
	var mw dataMW
	if json.Unmarshal([]byte(d), &mw) != nil {
		return ""
	}
	var b strings.Builder
	for _, raw := range mw.Parts {
		var text string
		if json.Unmarshal(raw, &text) == nil {
			b.WriteString(text)
			b.WriteByte('\n')
			continue
		}
		var p part
		if json.Unmarshal(raw, &p) != nil || p.Template == nil {
			continue
		}
		for _, v := range p.Template.Params {
			b.WriteString(v.WT)
			b.WriteByte('\n')
		}
	}
	return b.String()
}

// stripDisplay removes the {\displaystyle …} that MediaWiki puts around the
// TeX of the alttext.
func stripDisplay(s string) string {
	s = strings.TrimSpace(s)
	for _, p := range []string{`{\displaystyle `, `{\textstyle `} {
		if strings.HasPrefix(s, p) && strings.HasSuffix(s, "}") {
			return strings.TrimSpace(s[len(p) : len(s)-1])
		}
	}
	return s
}

func attr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

// firstHeading gives the heading of a section: the first child element of
// the section, when it is a heading. Parsoid can put the heading in a div of
// the class mw-heading.
func firstHeading(s *html.Node) *html.Node {
	for c := s.FirstChild; c != nil; c = c.NextSibling {
		if c.Type != html.ElementNode {
			continue
		}
		switch c.DataAtom {
		case atom.H1, atom.H2, atom.H3, atom.H4, atom.H5, atom.H6:
			return c
		case atom.Div:
			for h := c.FirstChild; h != nil; h = h.NextSibling {
				if h.Type == html.ElementNode && len(h.Data) == 2 && h.Data[0] == 'h' && h.Data[1] >= '1' && h.Data[1] <= '6' {
					return h
				}
			}
		}
		return nil
	}
	return nil
}

func find(n *html.Node, a atom.Atom) *html.Node {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.DataAtom == a {
			return c
		}
		if f := find(c, a); f != nil {
			return f
		}
	}
	return nil
}

func text(n *html.Node) string {
	var b strings.Builder
	var walk func(n *html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			b.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return b.String()
}
