package parse

// resolveQuotes turns the runs of apostrophes in a node list into bold and
// italic nodes. As in MediaWiki, "”" toggles italic, "”'" toggles bold,
// "””'" toggles both, and each format closes at the end of a line.
func resolveQuotes(nodes []*Node) []*Node {
	has := false
	for _, n := range nodes {
		if n.Kind == kindQuote {
			has = true
			break
		}
	}
	if !has {
		return nodes
	}
	var root []*Node
	var stack []*Node // open bold and italic nodes, outer first
	add := func(n *Node) {
		if len(stack) == 0 {
			root = append(root, n)
			return
		}
		top := stack[len(stack)-1]
		top.Children = append(top.Children, n)
	}
	closeTo := func(i, at int) {
		for len(stack) > i {
			top := stack[len(stack)-1]
			top.End = at
			stack = stack[:len(stack)-1]
		}
	}
	find := func(k Kind) int {
		for i, s := range stack {
			if s.Kind == k {
				return i
			}
		}
		return -1
	}
	toggle := func(k Kind, start, end int) {
		if i := find(k); i >= 0 {
			// Close k and each format inside it, then open again the formats
			// that were inside, so that "''a '''b'' c'''" keeps c in bold.
			var reopen []Kind
			for _, s := range stack[i+1:] {
				reopen = append(reopen, s.Kind)
			}
			closeTo(i, end)
			for _, rk := range reopen {
				n := &Node{Kind: rk, Start: end, End: end}
				add(n)
				stack = append(stack, n)
			}
			return
		}
		n := &Node{Kind: k, Start: start, End: end}
		add(n)
		stack = append(stack, n)
	}
	for _, n := range nodes {
		if n.Kind != kindQuote {
			if isNewline(n) {
				closeTo(0, n.Start)
			}
			add(n)
			if len(stack) > 0 {
				for _, s := range stack {
					if n.End > s.End {
						s.End = n.End
					}
				}
			}
			continue
		}
		start, k := n.Start, n.End-n.Start
		// Extra apostrophes before the run are text.
		switch {
		case k == 4:
			add(&Node{Kind: KindText, Start: start, End: start + 1})
			start++
			k = 3
		case k > 5:
			add(&Node{Kind: KindText, Start: start, End: start + k - 5})
			start += k - 5
			k = 5
		}
		end := start + k
		switch k {
		case 2:
			toggle(KindItalic, start, end)
		case 3:
			toggle(KindBold, start, end)
		case 5:
			bi, ii := find(KindBold), find(KindItalic)
			switch {
			case bi >= 0 && ii >= 0:
				if bi > ii {
					toggle(KindBold, start, start+3)
					toggle(KindItalic, start+3, end)
				} else {
					toggle(KindItalic, start, start+2)
					toggle(KindBold, start+2, end)
				}
			case bi >= 0:
				toggle(KindBold, start, start+3)
				toggle(KindItalic, start+3, end)
			case ii >= 0:
				toggle(KindItalic, start, start+2)
				toggle(KindBold, start+2, end)
			default:
				toggle(KindBold, start, start+3)
				toggle(KindItalic, start+3, end)
			}
		}
	}
	if len(stack) > 0 {
		last := stack[len(stack)-1]
		closeTo(0, last.End)
	}
	return dropEmpty(root)
}

// dropEmpty removes the bold and italic nodes with no content that a reopen
// leaves at the end of a run.
func dropEmpty(nodes []*Node) []*Node {
	out := nodes[:0]
	for _, n := range nodes {
		if (n.Kind == KindBold || n.Kind == KindItalic) && len(n.Children) == 0 && n.Start == n.End {
			continue
		}
		if n.Kind == KindBold || n.Kind == KindItalic {
			n.Children = dropEmpty(n.Children)
		}
		out = append(out, n)
	}
	return out
}
