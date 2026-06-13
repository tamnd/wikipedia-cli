package wiki

import (
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// HTMLToText renders an HTML fragment into readable plain text. It is pragmatic,
// not pixel-perfect: headings get blank lines, list items get bullets, links
// collapse to their text, and citation/edit chrome is dropped.
func HTMLToText(htmlStr string) string {
	return renderHTML(htmlStr, false)
}

// HTMLToMarkdown renders an HTML fragment into Markdown.
func HTMLToMarkdown(htmlStr string) string {
	return renderHTML(htmlStr, true)
}

func renderHTML(htmlStr string, md bool) string {
	doc, err := html.Parse(strings.NewReader(htmlStr))
	if err != nil {
		return htmlStr
	}
	r := &renderer{md: md}
	r.walk(doc)
	return cleanBlank(r.b.String())
}

type renderer struct {
	b        strings.Builder
	md       bool
	listKind []byte // 'u' or 'o' stack
	olCount  []int
}

func (r *renderer) walk(n *html.Node) {
	switch n.Type {
	case html.TextNode:
		text := collapseWS(n.Data)
		if text != "" {
			r.b.WriteString(text)
		}
		return
	case html.ElementNode:
		if skipElement(n) {
			return
		}
		r.element(n)
		return
	default:
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			r.walk(c)
		}
	}
}

func (r *renderer) children(n *html.Node) {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		r.walk(c)
	}
}

func (r *renderer) element(n *html.Node) {
	switch n.DataAtom {
	case atom.H1, atom.H2, atom.H3, atom.H4, atom.H5, atom.H6:
		level := int(n.DataAtom-atom.H1) + 1
		r.block()
		title := strings.TrimSpace(inlineText(n))
		if title == "" {
			return
		}
		if r.md {
			r.b.WriteString(strings.Repeat("#", level) + " " + title)
		} else {
			r.b.WriteString(title)
		}
		r.block()
	case atom.P, atom.Div:
		r.block()
		r.children(n)
		r.block()
	case atom.Br:
		r.b.WriteString("\n")
	case atom.Hr:
		r.block()
		if r.md {
			r.b.WriteString("---")
		}
		r.block()
	case atom.Ul, atom.Ol:
		r.block()
		kind := byte('u')
		if n.DataAtom == atom.Ol {
			kind = 'o'
			r.olCount = append(r.olCount, 0)
		}
		r.listKind = append(r.listKind, kind)
		r.children(n)
		r.listKind = r.listKind[:len(r.listKind)-1]
		if kind == 'o' {
			r.olCount = r.olCount[:len(r.olCount)-1]
		}
		r.block()
	case atom.Li:
		r.newline()
		indent := strings.Repeat("  ", max(len(r.listKind)-1, 0))
		marker := "- "
		if len(r.listKind) > 0 && r.listKind[len(r.listKind)-1] == 'o' {
			r.olCount[len(r.olCount)-1]++
			marker = itoa(r.olCount[len(r.olCount)-1]) + ". "
		}
		r.b.WriteString(indent + marker + strings.TrimSpace(inlineText(n)))
	case atom.A:
		text := strings.TrimSpace(inlineText(n))
		if text == "" {
			return
		}
		if r.md {
			href := attr(n, "href")
			href = absHref(href)
			if href != "" && !strings.HasPrefix(href, "#") {
				r.b.WriteString("[" + text + "](" + href + ")")
				return
			}
		}
		r.b.WriteString(text)
	case atom.B, atom.Strong:
		r.wrap(n, "**")
	case atom.I, atom.Em:
		r.wrap(n, "_")
	case atom.Code, atom.Tt:
		r.wrap(n, "`")
	case atom.Pre:
		r.block()
		if r.md {
			r.b.WriteString("```\n" + inlineText(n) + "\n```")
		} else {
			r.b.WriteString(inlineText(n))
		}
		r.block()
	case atom.Blockquote:
		r.block()
		text := strings.TrimSpace(inlineText(n))
		if r.md {
			for line := range strings.SplitSeq(text, "\n") {
				r.b.WriteString("> " + line + "\n")
			}
		} else {
			r.b.WriteString(text)
		}
		r.block()
	case atom.Table:
		// Tables (incl. infoboxes) are noisy in text; emit a compact key/value
		// dump of th/td pairs rather than ASCII-art.
		r.block()
		r.renderTable(n)
		r.block()
	default:
		r.children(n)
	}
}

func (r *renderer) wrap(n *html.Node, mark string) {
	text := strings.TrimSpace(inlineText(n))
	if text == "" {
		return
	}
	if r.md {
		r.b.WriteString(mark + text + mark)
	} else {
		r.b.WriteString(text)
	}
}

func (r *renderer) renderTable(n *html.Node) {
	var rows [][2]string
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.DataAtom == atom.Tr {
			var cells []string
			for c := node.FirstChild; c != nil; c = c.NextSibling {
				if c.DataAtom == atom.Th || c.DataAtom == atom.Td {
					cells = append(cells, strings.TrimSpace(inlineText(c)))
				}
			}
			if len(cells) == 2 && (cells[0] != "" || cells[1] != "") {
				rows = append(rows, [2]string{cells[0], cells[1]})
			}
			return
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	for i, row := range rows {
		if i > 0 {
			r.b.WriteString("\n")
		}
		if row[0] != "" {
			r.b.WriteString(row[0] + ": " + row[1])
		} else {
			r.b.WriteString(row[1])
		}
	}
}

func (r *renderer) block()   { r.ensureTrailing("\n\n") }
func (r *renderer) newline() { r.ensureTrailing("\n") }

func (r *renderer) ensureTrailing(s string) {
	cur := r.b.String()
	if cur == "" {
		return
	}
	need := len(s)
	have := 0
	for have < need && have < len(cur) && cur[len(cur)-1-have] == '\n' {
		have++
	}
	for ; have < need; have++ {
		r.b.WriteByte('\n')
	}
}

// inlineText flattens an element's descendants to a single spaced string,
// dropping tags. Used for headings, list items, links, and table cells.
func inlineText(n *html.Node) string {
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.ElementNode && skipElement(node) {
			return
		}
		if node.Type == html.TextNode {
			b.WriteString(collapseWS(node.Data))
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return strings.Join(strings.Fields(b.String()), " ")
}

// skipElement reports whether an element (and its subtree) is chrome we drop:
// scripts, styles, citation superscripts, edit links, nav boxes, and the
// reference list.
func skipElement(n *html.Node) bool {
	switch n.DataAtom {
	case atom.Script, atom.Style, atom.Noscript:
		return true
	case atom.Sup:
		// A sup is almost always a [1]-style citation marker; drop it.
		return hasClass(n, "reference")
	}
	class := attr(n, "class")
	for _, drop := range []string{"mw-editsection", "reference", "navbox", "reflist", "mw-references", "noprint", "metadata"} {
		if strings.Contains(class, drop) {
			return true
		}
	}
	if attr(n, "role") == "navigation" {
		return true
	}
	return false
}

func hasClass(n *html.Node, class string) bool {
	return strings.Contains(" "+attr(n, "class")+" ", " "+class+" ")
}

func attr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

func absHref(href string) string {
	if strings.HasPrefix(href, "//") {
		return "https:" + href
	}
	if strings.HasPrefix(href, "/wiki/") {
		return "https://en.wikipedia.org" + href
	}
	return href
}

func collapseWS(s string) string {
	if strings.TrimSpace(s) == "" {
		// Preserve a single space for inter-word whitespace nodes.
		if strings.ContainsAny(s, " \t\n") {
			return " "
		}
		return ""
	}
	// Collapse internal runs of whitespace to single spaces but keep edge spaces.
	leading := s[0] == ' ' || s[0] == '\n' || s[0] == '\t'
	trailing := s[len(s)-1] == ' ' || s[len(s)-1] == '\n' || s[len(s)-1] == '\t'
	out := strings.Join(strings.Fields(s), " ")
	if leading {
		out = " " + out
	}
	if trailing {
		out += " "
	}
	return out
}

func cleanBlank(s string) string {
	lines := strings.Split(s, "\n")
	var out []string
	blanks := 0
	for _, line := range lines {
		line = strings.TrimRight(line, " \t")
		if line == "" {
			blanks++
			if blanks > 1 {
				continue
			}
		} else {
			blanks = 0
		}
		out = append(out, line)
	}
	return strings.TrimSpace(strings.Join(out, "\n")) + "\n"
}
