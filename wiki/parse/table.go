package parse

import "strings"

// tableEnd returns the index of the line that closes the table that starts
// at line i. Nested tables are counted. A table with no close ends at the
// last line.
func tableEnd(src string, ls []line, i int) int {
	depth := 0
	for j := i; j < len(ls); j++ {
		t := strings.TrimLeft(src[ls[j].start:ls[j].end], " \t")
		switch {
		case strings.HasPrefix(t, "{|"):
			depth++
		case strings.HasPrefix(t, "|}"):
			depth--
			if depth == 0 {
				return j
			}
		}
	}
	return len(ls) - 1
}

// cellSpan is the content of a cell or a caption before it is parsed.
type cellSpan struct {
	node       *Node
	start, end int
}

// table reads the lines of one table, from "{|" to "|}".
func (p *parser) table(ls []line) *Node {
	first := ls[0]
	open := strings.Index(p.src[first.start:first.end], "{|")
	t := &Node{Kind: KindTable, Start: first.start, End: ls[len(ls)-1].end}
	t.Attrs = parseAttrs(p.src[first.start+open+2 : first.end])
	var row *Node
	var cells []cellSpan
	addRow := func(start int) *Node {
		r := &Node{Kind: KindRow, Start: start, End: start}
		t.Children = append(t.Children, r)
		return r
	}
	extend := func(end int) {
		if len(cells) > 0 {
			cells[len(cells)-1].end = end
			cells[len(cells)-1].node.End = end
		}
		if row != nil && end > row.End {
			row.End = end
		}
	}
	last := len(ls) - 1
	if !strings.HasPrefix(strings.TrimLeft(p.src[ls[last].start:ls[last].end], " \t"), "|}") {
		last = len(ls)
	}
	for i := 1; i < last; i++ {
		l := ls[i]
		raw := p.src[l.start:l.end]
		lead := len(raw) - len(strings.TrimLeft(raw, " \t"))
		t0 := raw[lead:]
		s := l.start + lead
		switch {
		case strings.HasPrefix(t0, "{|"):
			// A nested table is part of the content of the current cell.
			j := tableEnd(p.src, ls, i)
			extend(ls[j].end)
			i = j
		case strings.HasPrefix(t0, "|+"):
			c := &Node{Kind: KindCaption, Start: s, End: l.end}
			t.Children = append(t.Children, c)
			cells = append(cells, p.cellAttrs(c, l, s+2, l.end))
		case strings.HasPrefix(t0, "|-"):
			row = addRow(s)
			row.Attrs = parseAttrs(strings.TrimLeft(p.src[s+2:l.end], "-"))
			row.End = l.end
		case strings.HasPrefix(t0, "!") || strings.HasPrefix(t0, "|"):
			header := t0[0] == '!'
			if row == nil {
				row = addRow(s)
			}
			for _, seg := range p.cellSegments(l, s+1, l.end, header) {
				c := &Node{Kind: KindCell, Start: seg[0], End: seg[2], Header: header}
				row.Children = append(row.Children, c)
				cells = append(cells, p.cellAttrs(c, l, seg[1], seg[2]))
			}
			row.End = l.end
		default:
			extend(l.end)
		}
	}
	for _, c := range cells {
		if strings.Contains(p.src[c.start:c.end], "\n") {
			c.node.Children = p.blocks(c.start, c.end)
		} else {
			c.node.Children, _ = p.inline(c.start, c.end, 0)
		}
	}
	return t
}

// cellSegments splits a cell line at "||", and at "!!" in a header line.
// Each segment is the start of the cell, the start of its content, and its
// end.
func (p *parser) cellSegments(l line, start, end int, header bool) [][3]int {
	var cuts []int
	seps := []string{"||"}
	if header {
		seps = append(seps, "!!")
	}
	walkText(p.src, l.nodes, func(ts, te int) {
		if te <= start || ts >= end {
			return
		}
		if ts < start {
			ts = start
		}
		if te > end {
			te = end
		}
		k := ts
		for k < te-1 {
			hit := false
			for _, sep := range seps {
				if strings.HasPrefix(p.src[k:te], sep) {
					cuts = append(cuts, k)
					hit = true
				}
			}
			if hit {
				k += 2
			} else {
				k++
			}
		}
	})
	var out [][3]int
	from := start
	cellStart := start - 1
	for _, c := range cuts {
		out = append(out, [3]int{cellStart, from, c})
		cellStart, from = c, c+2
	}
	return append(out, [3]int{cellStart, from, end})
}

// cellAttrs splits "attrs | content" when the part before a single "|" is
// only text.
func (p *parser) cellAttrs(c *Node, l line, start, end int) cellSpan {
	bar := -1
	plain := true
	for _, n := range l.nodes {
		if n.End <= start || n.Start >= end {
			continue
		}
		if n.Kind != KindText {
			plain = false
			break
		}
		lo, hi := n.Start, n.End
		if lo < start {
			lo = start
		}
		if hi > end {
			hi = end
		}
		if k := strings.IndexByte(p.src[lo:hi], '|'); k >= 0 {
			bar = lo + k
			break
		}
	}
	if bar >= 0 && plain && !strings.Contains(p.src[start:bar], "[[") {
		c.Attrs = parseAttrs(p.src[start:bar])
		return cellSpan{node: c, start: bar + 1, end: end}
	}
	return cellSpan{node: c, start: start, end: end}
}

// walkText calls fn with the span of each text node, inside bold and italic
// nodes too.
func walkText(src string, nodes []*Node, fn func(start, end int)) {
	for _, n := range nodes {
		switch n.Kind {
		case KindText:
			fn(n.Start, n.End)
		case KindBold, KindItalic:
			walkText(src, n.Children, fn)
		}
	}
}
