package parse

import (
	"errors"
	"strings"
	"time"
)

// stops are the tokens that end an inline run. The token is not consumed.
type stops uint8

const (
	stopPipe     stops = 1 << iota // "|"
	stopBraces                     // "}}"
	stopBrackets                   // "]]"
	stopBracket                    // "]"
	stopNewline                    // "\n"
)

// kindQuote is a run of two or more apostrophes before resolveQuotes turns
// the runs into bold and italic nodes.
const kindQuote Kind = 255

// ErrTimeout is returned when a page takes longer than Options.Timeout.
var ErrTimeout = errors.New("parse: time limit exceeded")

// ErrTooLarge is returned when a page is larger than Options.MaxBytes.
var ErrTooLarge = errors.New("parse: page is too large")

type parser struct {
	src      string
	deadline time.Time
	steps    int
	err      error

	// The memo tables make the parse linear. A construct that starts at an
	// offset gives the same node in each context that contains its end, and a
	// construct that does not close before an end does not close before a
	// smaller end.
	okTpl, okLink, okTag map[int]*Node
	badTpl, badLink      map[int]int
	eofLink              map[int]int // links whose label found no "]]"
	badTag               map[int]int

	// lower is the source in lower case, and tagPos holds the offsets of tag
	// prefixes in it, for findClose.
	lower  string
	tagPos map[string][]int
}

func newParser(src string, deadline time.Time) *parser {
	return &parser{
		src: src, deadline: deadline,
		okTpl: map[int]*Node{}, okLink: map[int]*Node{}, okTag: map[int]*Node{},
		badTpl: map[int]int{}, badLink: map[int]int{}, badTag: map[int]int{}, eofLink: map[int]int{},
	}
}

func (p *parser) tick() bool {
	if p.err != nil {
		return false
	}
	p.steps++
	if p.steps&1023 == 0 && !p.deadline.IsZero() && time.Now().After(p.deadline) {
		p.err = ErrTimeout
		return false
	}
	return true
}

const specials = "<{['|}]\n"

// inline parses src[start:end] until end or a stop token at the top level.
// It returns the nodes and the offset where it stopped.
func (p *parser) inline(start, end int, st stops) ([]*Node, int) {
	var out []*Node
	textStart := start
	flush := func(i int) {
		if i > textStart {
			out = appendText(out, textStart, i)
		}
	}
	i := start
	for i < end {
		if !p.tick() {
			flush(end)
			return resolveQuotes(out), end
		}
		j := strings.IndexAny(p.src[i:end], specials)
		if j < 0 {
			break
		}
		i += j
		c := p.src[i]
		rest := p.src[i:end]
		switch {
		case st&stopPipe != 0 && c == '|',
			st&stopBraces != 0 && strings.HasPrefix(rest, "}}"),
			st&stopBrackets != 0 && strings.HasPrefix(rest, "]]"),
			st&stopBracket != 0 && c == ']',
			st&stopNewline != 0 && c == '\n':
			flush(i)
			return resolveQuotes(out), i
		}
		var n *Node
		switch c {
		case '\n':
			flush(i)
			out = append(out, &Node{Kind: KindText, Start: i, End: i + 1, Name: "\n"})
			i++
			textStart = i
			continue
		case '<':
			if strings.HasPrefix(rest, "<!--") {
				n = p.comment(i, end)
			} else {
				n = p.tag(i, end)
			}
		case '{':
			if strings.HasPrefix(rest, "{{{") {
				n = p.argument(i, end)
			}
			if n == nil && strings.HasPrefix(rest, "{{") {
				n = p.template(i, end)
			}
		case '[':
			if strings.HasPrefix(rest, "[[") {
				n = p.link(i, end)
			} else {
				n = p.extLink(i, end)
			}
		case '\'':
			k := 0
			for i+k < end && p.src[i+k] == '\'' {
				k++
			}
			if k >= 2 {
				n = &Node{Kind: kindQuote, Start: i, End: i + k}
			} else {
				i++
				continue
			}
		}
		if n == nil && p.cannotClose(i, end, st) {
			// A nested construct of the same type found no close before end.
			// The construct that holds this run cannot close either, so stop
			// now. This keeps the parse linear on pages with many unclosed
			// "{{" or "[[".
			flush(end)
			return resolveQuotes(out), end
		}
		if n == nil {
			// Not a construct: the character is text. Skip a run of the same
			// opening character, so that "{{{{" is not tried four times.
			i++
			for i < end && p.src[i] == c && (c == '{' || c == '[') {
				i++
			}
			continue
		}
		flush(i)
		out = append(out, n)
		i = n.End
		textStart = i
	}
	flush(end)
	return resolveQuotes(out), end
}

// cannotClose reports whether the construct at i failed because it found no
// close token before end, in a run that looks for the same close token.
func (p *parser) cannotClose(i, end int, st stops) bool {
	rest := p.src[i:end]
	switch {
	case st&stopBraces != 0 && strings.HasPrefix(rest, "{{") && !strings.HasPrefix(rest, "{{{"):
		e, ok := p.badTpl[i]
		return ok && e >= end
	case st&stopBrackets != 0 && strings.HasPrefix(rest, "[[") && !strings.HasPrefix(rest, "[[["):
		e, ok := p.eofLink[i]
		return ok && e >= end
	}
	return false
}

func appendText(out []*Node, start, end int) []*Node {
	if k := len(out); k > 0 {
		last := out[k-1]
		if last.Kind == KindText && last.End == start && !isNewline(last) {
			last.End = end
			return out
		}
	}
	return append(out, &Node{Kind: KindText, Start: start, End: end})
}

func isNewline(n *Node) bool { return n.Kind == KindText && n.End-n.Start == 1 && n.Name == "\n" }

func (p *parser) comment(i, end int) *Node {
	j := strings.Index(p.src[i+4:end], "-->")
	if j < 0 {
		return &Node{Kind: KindComment, Start: i, End: end}
	}
	return &Node{Kind: KindComment, Start: i, End: i + 4 + j + 3}
}

// argument parses {{{name|default}}}. It is rare in articles.
func (p *parser) argument(i, end int) *Node {
	nodes, pos := p.inline(i+3, end, stopPipe|stopBraces)
	_ = nodes
	for pos < end && p.src[pos] == '|' {
		_, pos = p.inline(pos+1, end, stopBraces)
	}
	if !strings.HasPrefix(p.src[pos:end], "}}}") {
		return nil
	}
	return &Node{Kind: KindArgument, Start: i, End: pos + 3, Name: strings.TrimSpace(p.src[i+3 : nameEnd(p.src, i+3, pos)])}
}

func nameEnd(src string, start, pos int) int {
	if k := strings.IndexByte(src[start:pos], '|'); k >= 0 {
		return start + k
	}
	return pos
}

// magic words that take a value after a colon, as parser functions do.
var colonWords = map[string]bool{
	"defaultsort": true, "displaytitle": true, "lc": true, "uc": true, "lcfirst": true,
	"ucfirst": true, "urlencode": true, "anchorencode": true, "formatnum": true,
	"fullurl": true, "localurl": true, "ns": true, "padleft": true, "padright": true,
	"plural": true, "int": true, "msg": true, "subst": true, "safesubst": true,
	"defaultsortkey": true, "defaultcategorysort": true,
}

func (p *parser) template(i, end int) *Node {
	if n, ok := p.okTpl[i]; ok && n.End <= end {
		return n
	}
	if e, ok := p.badTpl[i]; ok && end <= e {
		return nil
	}
	n := p.template1(i, end)
	if n == nil {
		p.badTpl[i] = end
	} else {
		p.okTpl[i] = n
	}
	return n
}

func (p *parser) template1(i, end int) *Node {
	nameStart := i + 2
	nameNodes, pos := p.inline(nameStart, end, stopPipe|stopBraces)
	if pos >= end {
		return nil
	}
	n := &Node{Kind: KindTemplate, Start: i}
	rawName := stripComments(p.src[nameStart:pos])
	// A parser function or a magic word with a value: {{#if:x|y}},
	// {{DEFAULTSORT:x}}. The text after the colon is the first parameter.
	if colon := topColon(p.src, nameNodes); colon >= 0 {
		fn := strings.TrimSpace(stripComments(p.src[nameStart:colon]))
		if strings.HasPrefix(fn, "#") || colonWords[strings.ToLower(fn)] {
			n.Name = fn
			val, _ := p.inline(colon+1, pos, 0)
			n.Params = append(n.Params, &Param{Index: 1, Start: colon + 1, End: pos, ValueStart: colon + 1, Value: val})
		}
	}
	if n.Name == "" {
		n.Name = strings.TrimSpace(rawName)
		if n.Name == "" || !validName(p.src, nameNodes) {
			return nil
		}
	}
	index := len(n.Params)
	for {
		if strings.HasPrefix(p.src[pos:end], "}}") {
			n.End = pos + 2
			return n
		}
		if pos >= end || p.src[pos] != '|' {
			return nil
		}
		ps := pos + 1
		val, pe := p.inline(ps, end, stopPipe|stopBraces)
		if pe >= end {
			return nil
		}
		par := &Param{Start: ps, End: pe, ValueStart: ps, Value: val}
		if eq := topByte(p.src, val, '='); eq >= 0 {
			par.Name = strings.TrimSpace(stripComments(p.src[ps:eq]))
			par.ValueStart = eq + 1
			par.Value, _ = p.inline(eq+1, pe, 0)
		} else {
			index++
			par.Index = index
		}
		n.Params = append(n.Params, par)
		pos = pe
	}
}

// validName reports whether the name of a template is a possible page
// title. A name with a link, a tag, or a line break inside the text is not,
// and MediaWiki shows such a template as text.
func validName(src string, nodes []*Node) bool {
	for i, n := range nodes {
		switch n.Kind {
		case KindTemplate, KindArgument, KindComment:
		case KindText:
			t := src[n.Start:n.End]
			if isNewline(n) {
				if i > 0 && i < len(nodes)-1 && hasText(src, nodes[:i]) && hasText(src, nodes[i+1:]) {
					return false
				}
				continue
			}
			if strings.ContainsAny(t, "[]<>{}") {
				return false
			}
		default:
			return false
		}
	}
	return true
}

func hasText(src string, nodes []*Node) bool {
	for _, n := range nodes {
		if n.Kind == KindText && strings.TrimSpace(src[n.Start:n.End]) != "" || n.Kind == KindTemplate {
			return true
		}
	}
	return false
}

// topByte returns the offset of the first byte c in the text nodes of a node
// list, or -1. Bold and italic nodes are searched too, because they hold text
// of the same level.
func topByte(src string, nodes []*Node, c byte) int {
	for _, n := range nodes {
		switch n.Kind {
		case KindText:
			if k := strings.IndexByte(src[n.Start:n.End], c); k >= 0 {
				return n.Start + k
			}
		case KindBold, KindItalic:
			if k := topByte(src, n.Children, c); k >= 0 {
				return k
			}
		}
	}
	return -1
}

func topColon(src string, nodes []*Node) int { return topByte(src, nodes, ':') }

func stripComments(s string) string {
	for {
		i := strings.Index(s, "<!--")
		if i < 0 {
			return s
		}
		j := strings.Index(s[i+4:], "-->")
		if j < 0 {
			return s[:i]
		}
		s = s[:i] + s[i+4+j+3:]
	}
}

func (p *parser) link(i, end int) *Node {
	if n, ok := p.okLink[i]; ok && n.End <= end {
		return n
	}
	if e, ok := p.badLink[i]; ok && end <= e {
		return nil
	}
	n := p.link1(i, end)
	if n == nil {
		p.badLink[i] = end
	} else {
		p.okLink[i] = n
	}
	return n
}

func (p *parser) link1(i, end int) *Node {
	ts := i + 2
	k := ts
	for k < end {
		c := p.src[k]
		if c == '|' || c == '\n' || c == '[' || c == '<' && !strings.HasPrefix(p.src[k:end], "<!--") {
			break
		}
		if c == ']' {
			break
		}
		if c == '{' && strings.HasPrefix(p.src[k:end], "{{") {
			t := p.template(k, end)
			if t == nil {
				return nil
			}
			k = t.End
			continue
		}
		k++
	}
	if k >= end {
		return nil
	}
	n := &Node{Kind: KindLink, Start: i, Name: strings.TrimSpace(stripComments(p.src[ts:k]))}
	if n.Name == "" {
		return nil
	}
	switch {
	case strings.HasPrefix(p.src[k:end], "]]"):
		n.End = k + 2
	case p.src[k] == '|':
		label, pos := p.inline(k+1, end, stopBrackets)
		if pos >= end {
			p.eofLink[i] = end
			return nil
		}
		n.Children = label
		n.End = pos + 2
	default:
		return nil
	}
	if !hasNamespace(n.Name) {
		t := n.End
		for t < end && isTrailByte(p.src[t]) {
			t++
		}
		if t > n.End {
			n.Trail = p.src[n.End:t]
			n.End = t
		}
	}
	return n
}

func isTrailByte(c byte) bool { return c >= 'a' && c <= 'z' }

// hasNamespace reports whether a link target starts with a namespace or an
// interwiki prefix. Such links take no link trail.
func hasNamespace(target string) bool {
	t := strings.TrimPrefix(target, ":")
	i := strings.IndexByte(t, ':')
	if i <= 0 {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(t[:i])) {
	case "category", "file", "image", "media", "wikt", "wiktionary", "s", "wikisource", "commons", "q", "b", "v", "n", "d", "m", "w", "mw", "meta", "species", "wikipedia", "wp", "help", "template", "portal", "user", "talk", "special", "draft", "module":
		return true
	}
	return len(t[:i]) <= 3 && isLower(t[:i])
}

func isLower(s string) bool {
	for i := 0; i < len(s); i++ {
		if (s[i] < 'a' || s[i] > 'z') && s[i] != '-' {
			return false
		}
	}
	return s != ""
}

var schemes = []string{"http://", "https://", "//", "ftp://", "ftps://", "mailto:", "irc://", "ircs://", "news:", "gopher://", "sftp://", "urn:", "doi:"}

func (p *parser) extLink(i, end int) *Node {
	rest := p.src[i+1 : end]
	ok := false
	for _, s := range schemes {
		if len(rest) >= len(s) && strings.EqualFold(rest[:len(s)], s) {
			ok = true
			break
		}
	}
	if !ok {
		return nil
	}
	k := i + 1
	for k < end {
		c := p.src[k]
		if c == ' ' || c == ']' || c == '\n' || c == '\t' || c == '<' || c == '"' || c == '[' {
			break
		}
		if c == '{' && strings.HasPrefix(p.src[k:end], "{{") {
			t := p.template(k, end)
			if t == nil {
				break
			}
			k = t.End
			continue
		}
		k++
	}
	n := &Node{Kind: KindExtLink, Start: i, Name: p.src[i+1 : k]}
	for k < end && (p.src[k] == ' ' || p.src[k] == '\t') {
		k++
	}
	if k < end && p.src[k] == ']' {
		n.End = k + 1
		return n
	}
	label, pos := p.inline(k, end, stopBracket|stopNewline)
	if pos >= end || p.src[pos] != ']' {
		return nil
	}
	n.Children = label
	n.End = pos + 1
	return n
}
