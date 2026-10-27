package parse

import (
	"fmt"
	"strings"
)

// Plain returns the text of a node list as a reader sees it, with no markup.
// Links give their label, or their target when they have no label. Templates,
// comments, references and categories give nothing. Math gives its TeX
// source. It is for headings, names and short spans, not for whole pages.
func (d *Document) Plain(nodes []*Node) string {
	var b strings.Builder
	d.plain(&b, nodes)
	return strings.Join(strings.Fields(b.String()), " ")
}

func (d *Document) plain(b *strings.Builder, nodes []*Node) {
	for _, n := range nodes {
		switch n.Kind {
		case KindText:
			b.WriteString(decodeEntities(d.Src[n.Start:n.End]))
		case KindBold, KindItalic, KindParagraph, KindListItem, KindHeading, KindCell, KindCaption:
			d.plain(b, n.Children)
		case KindLink:
			if strings.HasPrefix(strings.ToLower(n.Name), "category:") || isFileLink(n.Name) {
				continue
			}
			if len(n.Children) > 0 {
				d.plain(b, n.Children)
			} else {
				b.WriteString(strings.TrimPrefix(n.Name, ":"))
			}
			b.WriteString(n.Trail)
		case KindExtLink:
			d.plain(b, n.Children)
		case KindTag:
			switch n.Name {
			case "math", "chem", "ce":
				b.WriteString(d.Inner(n))
			case "nowiki", "code", "pre":
				b.WriteString(decodeEntities(d.Inner(n)))
			case "ref", "references", "includeonly", "templatestyles":
			case "br":
				b.WriteByte(' ')
			default:
				d.plain(b, n.Children)
			}
		}
	}
}

func isFileLink(target string) bool {
	t := strings.ToLower(strings.TrimSpace(target))
	return strings.HasPrefix(t, "file:") || strings.HasPrefix(t, "image:")
}

var entities = strings.NewReplacer(
	"&nbsp;", " ", "&amp;", "&", "&lt;", "<", "&gt;", ">", "&quot;", "\"", "&#39;", "'",
	"&minus;", "−", "&ndash;", "–", "&mdash;", "—", "&times;", "×", "&thinsp;", " ", "&apos;", "'",
)

func decodeEntities(s string) string {
	if !strings.Contains(s, "&") {
		return s
	}
	return entities.Replace(s)
}

// Anchor returns the anchor that MediaWiki gives to a heading with the text
// s: spaces become underscores.
func Anchor(s string) string {
	return strings.ReplaceAll(strings.TrimSpace(s), " ", "_")
}

// HeadingText returns the plain text of the heading of a section, or "" for
// the lead.
func (d *Document) HeadingText(section *Node) string {
	if len(section.Children) == 0 || section.Children[0].Kind != KindHeading {
		return ""
	}
	return d.Plain(section.Children[0].Children)
}

// Dump prints the tree, one node on each line, with its kind, its span and
// its name. It is for tests and for "wiki parse --tree".
func (d *Document) Dump() string {
	var b strings.Builder
	d.dump(&b, d.Root, 0)
	return b.String()
}

func (d *Document) dump(b *strings.Builder, n *Node, depth int) {
	ind := strings.Repeat("  ", depth)
	fmt.Fprintf(b, "%s%s %d-%d", ind, n.Kind, n.Start, n.End)
	switch n.Kind {
	case KindText:
		fmt.Fprintf(b, " %q", d.Src[n.Start:n.End])
	case KindSection, KindHeading:
		fmt.Fprintf(b, " level=%d", n.Level)
	case KindListItem:
		fmt.Fprintf(b, " %q", n.Name)
	case KindTemplate, KindLink, KindExtLink, KindArgument:
		fmt.Fprintf(b, " %q", n.Name)
		if n.Trail != "" {
			fmt.Fprintf(b, " trail=%q", n.Trail)
		}
	case KindTag:
		fmt.Fprintf(b, " %s", n.Name)
		if n.SelfClosing {
			b.WriteString(" self")
		}
		if n.InnerEnd > n.InnerStart && len(n.Children) == 0 {
			fmt.Fprintf(b, " inner=%q", d.Inner(n))
		}
	case KindCell:
		if n.Header {
			b.WriteString(" header")
		}
	}
	if len(n.Attrs) > 0 {
		keys := make([]string, 0, len(n.Attrs))
		for k := range n.Attrs {
			keys = append(keys, k)
		}
		sortStrings(keys)
		for _, k := range keys {
			fmt.Fprintf(b, " %s=%q", k, n.Attrs[k])
		}
	}
	b.WriteByte('\n')
	for _, p := range n.Params {
		name := p.Name
		if name == "" {
			name = itoa(p.Index)
		}
		fmt.Fprintf(b, "%s  param %s %d-%d\n", ind, name, p.Start, p.End)
		for _, c := range p.Value {
			d.dump(b, c, depth+2)
		}
	}
	for _, c := range n.Children {
		d.dump(b, c, depth+1)
	}
}

func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}
