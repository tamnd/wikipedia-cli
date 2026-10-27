// Package parse reads wikitext into a tree with byte spans.
//
// The parser is not MediaWiki. It does not expand templates and it does not
// render HTML. It gives the structure of one page: sections, headings,
// paragraphs, lists, tables, templates with their parameters, links, tags and
// text. Each node has the byte offsets of its source, so a caller can point
// back to the exact place in the wikitext. The parser does not fail on bad
// wikitext. A construct that does not close is kept as text.
package parse

import "strings"

// Kind is the type of a node.
type Kind uint8

// The kinds of nodes.
const (
	KindDocument Kind = iota
	KindSection
	KindHeading
	KindParagraph
	KindPre
	KindList
	KindListItem
	KindRule
	KindTable
	KindCaption
	KindRow
	KindCell
	KindText
	KindComment
	KindTemplate
	KindArgument
	KindLink
	KindExtLink
	KindTag
	KindBold
	KindItalic
)

var kindNames = [...]string{
	"document", "section", "heading", "paragraph", "pre", "list", "item", "rule",
	"table", "caption", "row", "cell", "text", "comment", "template", "argument",
	"link", "extlink", "tag", "bold", "italic",
}

func (k Kind) String() string {
	if int(k) < len(kindNames) {
		return kindNames[k]
	}
	return "unknown"
}

// Node is one node of the tree. Start and End are byte offsets in the source
// of the document. The fields that a kind does not use are empty.
type Node struct {
	Kind  Kind
	Start int
	End   int

	// Name is the template name, the tag name, the link target, the URL of an
	// external link, or the list marker of a list item ("*", "#", ";", ":").
	Name string
	// Level is the heading level (2 to 6, and 1 for "= x ="), the section
	// level, or the depth of a list item.
	Level int
	// Header is true for a table header cell.
	Header bool
	// Attrs holds the attributes of a tag, a table, a row or a cell.
	Attrs map[string]string
	// Params holds the parameters of a template or a parser function. For a
	// parser function such as {{#if:x|y}}, the first parameter is the text
	// after the colon.
	Params []*Param
	// Children are the child nodes. For a link they are the label. For a tag
	// whose content is parsed (ref, sup, span and the others), they are the
	// content. For an opaque tag (math, nowiki, pre and the others), the
	// content is in Inner and Children is empty.
	Children []*Node
	// Inner is the span of the content of a tag between its open and close
	// tags. It is [0,0] for a tag that closes itself.
	InnerStart, InnerEnd int
	// SelfClosing is true for a tag such as <ref name="x"/> or <br/>.
	SelfClosing bool
	// Trail is the link trail, the letters after "]]" that MediaWiki adds to
	// the label.
	Trail string
}

// Param is one parameter of a template. A positional parameter has an empty
// Name and its Index (1, 2, ...). Value holds the parsed value.
type Param struct {
	Name       string
	Index      int
	Start, End int
	// ValueStart is the offset of the value, after the "=" of a named
	// parameter.
	ValueStart int
	Value      []*Node
}

// Document is a parsed page.
type Document struct {
	Src  string
	Root *Node
}

// Raw returns the source of a node.
func (d *Document) Raw(n *Node) string { return d.Src[n.Start:n.End] }

// Inner returns the content of a tag.
func (d *Document) Inner(n *Node) string { return d.Src[n.InnerStart:n.InnerEnd] }

// ParamValue returns the trimmed source of the value of a parameter.
func (d *Document) ParamValue(p *Param) string {
	return strings.TrimSpace(d.Src[p.ValueStart:p.End])
}

// Param returns the parameter of a template with the given name, or the
// positional parameter with the given number when name is "1", "2", and so on.
// It returns nil when the template has no such parameter. When a parameter is
// given twice, the last one wins, as in MediaWiki.
func (n *Node) Param(name string) *Param {
	var out *Param
	for _, p := range n.Params {
		if p.Name == name || (p.Name == "" && itoa(p.Index) == name) {
			out = p
		}
	}
	return out
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var b [20]byte
	n := len(b)
	for i > 0 {
		n--
		b[n] = byte('0' + i%10)
		i /= 10
	}
	return string(b[n:])
}

// Walk calls fn for n and each node below it, in document order. Template
// parameter values are visited after the template node. When fn returns
// false, Walk does not visit the nodes below that node.
func Walk(n *Node, fn func(*Node) bool) {
	if !fn(n) {
		return
	}
	for _, p := range n.Params {
		for _, c := range p.Value {
			Walk(c, fn)
		}
	}
	for _, c := range n.Children {
		Walk(c, fn)
	}
}

// Find returns each node below n, n included, of the given kind.
func Find(n *Node, kind Kind) []*Node {
	var out []*Node
	Walk(n, func(m *Node) bool {
		if m.Kind == kind {
			out = append(out, m)
		}
		return true
	})
	return out
}

// TemplateKey gives the key of a template name for a table lookup: the
// "Template:" prefix is removed, underscores become spaces, runs of spaces
// become one space, and the text is lower case.
func TemplateKey(name string) string {
	s := strings.TrimSpace(strings.ReplaceAll(name, "_", " "))
	if i := strings.IndexByte(s, ':'); i > 0 && strings.EqualFold(strings.TrimSpace(s[:i]), "template") {
		s = strings.TrimSpace(s[i+1:])
	}
	return strings.ToLower(strings.Join(strings.Fields(s), " "))
}
