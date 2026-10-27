package parse

import (
	"strings"
	"time"
)

// Options sets the limits of a parse.
type Options struct {
	// MaxBytes is the largest page that Parse reads. Zero means 8 MB.
	MaxBytes int
	// Timeout is the longest time that Parse takes. Zero means 5 seconds.
	// A negative value means no limit.
	Timeout time.Duration
}

// Parse reads a page with the default options.
func Parse(src string) (*Document, error) { return ParseOptions(src, Options{}) }

// ParseOptions reads a page. It returns ErrTooLarge or ErrTimeout when the
// page is over a limit.
func ParseOptions(src string, o Options) (*Document, error) {
	maxBytes := o.MaxBytes
	if maxBytes == 0 {
		maxBytes = 8 << 20
	}
	if len(src) > maxBytes {
		return nil, ErrTooLarge
	}
	var deadline time.Time
	switch {
	case o.Timeout == 0:
		deadline = time.Now().Add(5 * time.Second)
	case o.Timeout > 0:
		deadline = time.Now().Add(o.Timeout)
	}
	p := newParser(src, deadline)
	blocks := p.blocks(0, len(src))
	root := &Node{Kind: KindDocument, Start: 0, End: len(src), Children: sections(blocks, len(src))}
	if p.err != nil {
		return nil, p.err
	}
	return &Document{Src: src, Root: root}, nil
}

// ParseInline reads a fragment as inline wikitext, with no blocks. It is for
// a template parameter or a table cell that a caller reads again.
func ParseInline(src string) []*Node {
	p := newParser(src, time.Time{})
	nodes, _ := p.inline(0, len(src), 0)
	return nodes
}

// line is one line of the top level of a range, as nodes.
type line struct {
	start, end int // end is before the newline
	nodes      []*Node
}

func (p *parser) lines(start, end int) []line {
	nodes, _ := p.inline(start, end, 0)
	var out []line
	cur := line{start: start}
	for _, n := range nodes {
		if isNewline(n) {
			cur.end = n.Start
			out = append(out, cur)
			cur = line{start: n.End}
			continue
		}
		cur.nodes = append(cur.nodes, n)
	}
	cur.end = end
	out = append(out, cur)
	return out
}

// blocks reads src[start:end] into headings and blocks. Sections are made
// later from the headings.
func (p *parser) blocks(start, end int) []*Node {
	ls := p.lines(start, end)
	var out []*Node
	var para *Node
	var pre *Node
	var list *Node
	endPara := func() {
		para, pre = nil, nil
	}
	for i := 0; i < len(ls); i++ {
		if p.err != nil {
			return out
		}
		l := ls[i]
		raw := p.src[l.start:l.end]
		trim := strings.TrimSpace(stripComments(raw))
		if !strings.HasPrefix(raw, "*") && !strings.HasPrefix(raw, "#") && !strings.HasPrefix(raw, ":") && !strings.HasPrefix(raw, ";") {
			list = nil
		}
		switch {
		case trim == "":
			if onlyComments(l.nodes) && strings.TrimSpace(raw) != "" {
				continue
			}
			endPara()
			continue
		case isHeading(raw):
			endPara()
			out = append(out, p.heading(l))
			continue
		case strings.HasPrefix(strings.TrimLeft(raw, " \t"), "{|"):
			endPara()
			j := tableEnd(p.src, ls, i)
			out = append(out, p.table(ls[i:j+1]))
			i = j
			continue
		case strings.HasPrefix(raw, "----"):
			endPara()
			out = append(out, &Node{Kind: KindRule, Start: l.start, End: l.end})
			continue
		case raw[0] == '*' || raw[0] == '#' || raw[0] == ':' || raw[0] == ';':
			endPara()
			if list == nil {
				list = &Node{Kind: KindList, Start: l.start}
				out = append(out, list)
			}
			list.Children = append(list.Children, p.listItems(l)...)
			list.End = l.end
			continue
		case raw[0] == ' ' && !startsBlockTag(trim):
			para = nil
			if pre == nil {
				pre = &Node{Kind: KindPre, Start: l.start}
				out = append(out, pre)
			} else {
				pre.Children = append(pre.Children, &Node{Kind: KindText, Start: l.start - 1, End: l.start, Name: "\n"})
			}
			pre.Children = append(pre.Children, l.nodes...)
			pre.End = l.end
			continue
		}
		pre = nil
		if para == nil {
			para = &Node{Kind: KindParagraph, Start: l.start}
			out = append(out, para)
		} else {
			para.Children = append(para.Children, &Node{Kind: KindText, Start: l.start - 1, End: l.start, Name: "\n"})
		}
		para.Children = append(para.Children, l.nodes...)
		para.End = l.end
	}
	return out
}

// startsBlockTag reports whether a line starts with a tag or a construct that
// MediaWiki does not put in a pre block when the line starts with a space.
func startsBlockTag(s string) bool {
	return strings.HasPrefix(s, "{{") || strings.HasPrefix(s, "<") || strings.HasPrefix(s, "[[") || strings.HasPrefix(s, "{|") || strings.HasPrefix(s, "|")
}

func onlyComments(nodes []*Node) bool {
	for _, n := range nodes {
		if n.Kind == KindComment {
			continue
		}
		if n.Kind == KindText && n.Start < n.End {
			return false
		}
		if n.Kind != KindText {
			return false
		}
	}
	return true
}

func headingLevel(raw string) (int, int, int) {
	s := strings.TrimRight(stripTrailingComments(raw), " \t")
	if len(s) < 3 || s[0] != '=' || s[len(s)-1] != '=' {
		return 0, 0, 0
	}
	a, b := 0, 0
	for a < len(s) && s[a] == '=' {
		a++
	}
	for b < len(s) && s[len(s)-1-b] == '=' {
		b++
	}
	if a == len(s) {
		// A line of only "=" signs: MediaWiki takes the middle as the text.
		lv := (a - 1) / 2
		if lv < 1 {
			return 0, 0, 0
		}
		if lv > 6 {
			lv = 6
		}
		return lv, lv, len(s) - lv
	}
	lv := a
	if b < lv {
		lv = b
	}
	if lv > 6 {
		lv = 6
	}
	return lv, lv, len(s) - lv
}

func stripTrailingComments(s string) string {
	for {
		t := strings.TrimRight(s, " \t")
		if !strings.HasSuffix(t, "-->") {
			return s
		}
		i := strings.LastIndex(t, "<!--")
		if i < 0 {
			return s
		}
		s = t[:i]
	}
}

func isHeading(raw string) bool {
	lv, _, _ := headingLevel(raw)
	return lv > 0
}

func (p *parser) heading(l line) *Node {
	lv, a, b := headingLevel(p.src[l.start:l.end])
	n := &Node{Kind: KindHeading, Start: l.start, End: l.end, Level: lv}
	n.Children, _ = p.inline(l.start+a, l.start+b, 0)
	return n
}

// listItems reads one list line. A ";" item with a ":" in its text gives two
// items: the term and its definition.
func (p *parser) listItems(l line) []*Node {
	raw := p.src[l.start:l.end]
	k := 0
	for k < len(raw) && strings.IndexByte("*#:;", raw[k]) >= 0 {
		k++
	}
	prefix := raw[:k]
	cs := l.start + k
	if prefix[len(prefix)-1] == ';' {
		nodes, _ := p.inline(cs, l.end, 0)
		if c := topColon(p.src, nodes); c >= 0 {
			term := &Node{Kind: KindListItem, Start: l.start, End: c, Name: prefix, Level: len(prefix)}
			term.Children, _ = p.inline(cs, c, 0)
			def := &Node{Kind: KindListItem, Start: c, End: l.end, Name: prefix[:len(prefix)-1] + ":", Level: len(prefix)}
			def.Children, _ = p.inline(c+1, l.end, 0)
			return []*Node{term, def}
		}
	}
	it := &Node{Kind: KindListItem, Start: l.start, End: l.end, Name: prefix, Level: len(prefix)}
	it.Children, _ = p.inline(cs, l.end, 0)
	return []*Node{it}
}

// sections makes the section tree from a flat list of headings and blocks.
// The lead is a section of level 0. A section ends where the next heading of
// the same or a higher level starts.
func sections(blocks []*Node, end int) []*Node {
	lead := &Node{Kind: KindSection, Start: 0, End: end}
	i := 0
	for ; i < len(blocks) && blocks[i].Kind != KindHeading; i++ {
		lead.Children = append(lead.Children, blocks[i])
	}
	if i < len(blocks) {
		lead.End = blocks[i].Start
	}
	return append([]*Node{lead}, topSections(blocks[i:], end)...)
}

// topSections gives the top-level sections, each with its subsections
// inside. blocks starts with a heading.
func topSections(blocks []*Node, end int) []*Node {
	var top []*Node
	var stack []*Node
	for _, b := range blocks {
		if b.Kind != KindHeading {
			if len(stack) > 0 {
				s := stack[len(stack)-1]
				s.Children = append(s.Children, b)
			}
			continue
		}
		for len(stack) > 0 && stack[len(stack)-1].Level >= b.Level {
			stack[len(stack)-1].End = b.Start
			stack = stack[:len(stack)-1]
		}
		s := &Node{Kind: KindSection, Start: b.Start, End: end, Level: b.Level, Children: []*Node{b}}
		if len(stack) == 0 {
			top = append(top, s)
		} else {
			stack[len(stack)-1].Children = append(stack[len(stack)-1].Children, s)
		}
		stack = append(stack, s)
	}
	return top
}
