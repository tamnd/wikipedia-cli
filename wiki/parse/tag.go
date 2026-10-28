package parse

import (
	"sort"
	"strings"
)

// tagMode says how the parser reads the content of a tag.
type tagMode uint8

const (
	tagParsed tagMode = iota // the content is wikitext
	tagOpaque                // the content is kept as it is
	tagHidden                // the content is not shown on a page view
	tagVoid                  // the tag has no content
	tagLines                 // each line is a file name and a caption in wikitext
)

// tags maps the tags that the parser knows to their mode. Other names after
// "<" are text, as in MediaWiki.
var tags = map[string]tagMode{
	// Opaque extension tags.
	"nowiki": tagOpaque, "pre": tagOpaque, "syntaxhighlight": tagOpaque, "source": tagOpaque,
	"code": tagOpaque, "score": tagOpaque, "graph": tagOpaque, "timeline": tagOpaque,
	"gallery": tagLines, "math": tagOpaque, "chem": tagOpaque, "ce": tagOpaque,
	"templatedata": tagOpaque, "mapframe": tagOpaque, "maplink": tagOpaque, "hiero": tagOpaque,
	"inputbox": tagOpaque, "categorytree": tagOpaque, "imagemap": tagOpaque, "charinsert": tagOpaque,
	"templatestyles": tagOpaque, "langconvert": tagOpaque,
	// Page view rules for transclusion tags.
	"includeonly": tagHidden, "noinclude": tagParsed, "onlyinclude": tagParsed,
	// Tags with wikitext content.
	"ref": tagParsed, "references": tagParsed, "poem": tagParsed, "section": tagVoid,
	"sup": tagParsed, "sub": tagParsed, "span": tagParsed, "div": tagParsed,
	"b": tagParsed, "i": tagParsed, "u": tagParsed, "s": tagParsed, "small": tagParsed,
	"big": tagParsed, "blockquote": tagParsed, "center": tagParsed, "font": tagParsed,
	"strong": tagParsed, "em": tagParsed, "abbr": tagParsed, "var": tagParsed, "q": tagParsed,
	"del": tagParsed, "ins": tagParsed, "tt": tagParsed, "kbd": tagParsed, "samp": tagParsed,
	"cite": tagParsed, "dfn": tagParsed, "p": tagParsed, "ul": tagParsed, "ol": tagParsed,
	"li": tagParsed, "dl": tagParsed, "dt": tagParsed, "dd": tagParsed, "table": tagParsed,
	"tr": tagParsed, "td": tagParsed, "th": tagParsed, "caption": tagParsed, "ruby": tagParsed,
	"rb": tagParsed, "rt": tagParsed, "rp": tagParsed, "bdi": tagParsed, "mark": tagParsed,
	"data": tagParsed, "time": tagParsed, "h1": tagParsed, "h2": tagParsed, "h3": tagParsed,
	"h4": tagParsed, "h5": tagParsed, "h6": tagParsed, "strike": tagParsed,
	"br": tagVoid, "hr": tagVoid, "wbr": tagVoid,
}

func (p *parser) tag(i, end int) *Node {
	if n, ok := p.okTag[i]; ok && n.End <= end {
		return n
	}
	if e, ok := p.badTag[i]; ok && end <= e {
		return nil
	}
	n := p.tag1(i, end)
	if n == nil {
		p.badTag[i] = end
	} else {
		p.okTag[i] = n
	}
	return n
}

func (p *parser) tag1(i, end int) *Node {
	k := i + 1
	for k < end && isNameByte(p.src[k]) {
		k++
	}
	name := strings.ToLower(p.src[i+1 : k])
	mode, known := tags[name]
	if !known {
		return nil
	}
	// The open tag ends at the first ">", as in MediaWiki. A quoted value
	// cannot hold ">", so a value with no close quote does not hide it.
	gt := -1
	for j := k; j < end; j++ {
		if c := p.src[j]; c == '>' {
			gt = j
			break
		} else if c == '<' {
			break
		}
	}
	if gt < 0 || k < gt && !isSpaceByte(p.src[k]) && p.src[k] != '/' {
		return nil
	}
	n := &Node{Kind: KindTag, Start: i, Name: name}
	attrText := p.src[k:gt]
	if strings.HasSuffix(strings.TrimSpace(attrText), "/") {
		n.SelfClosing = true
		attrText = strings.TrimSuffix(strings.TrimSpace(attrText), "/")
	}
	n.Attrs = parseAttrs(attrText)
	if n.SelfClosing || mode == tagVoid {
		n.End = gt + 1
		n.InnerStart, n.InnerEnd = n.End, n.End
		return n
	}
	closeAt, closeEnd := p.findClose(name, gt+1, end, mode == tagParsed)
	if closeAt < 0 {
		// HTML formatting tags often have no close tag in articles. Keep the
		// open tag alone. An extension tag with no close is text.
		if mode == tagParsed && name != "ref" && name != "noinclude" && name != "onlyinclude" && name != "poem" && name != "references" {
			n.End = gt + 1
			n.InnerStart, n.InnerEnd = n.End, n.End
			return n
		}
		return nil
	}
	n.InnerStart, n.InnerEnd, n.End = gt+1, closeAt, closeEnd
	if mode == tagParsed || mode == tagLines {
		n.Children, _ = p.inline(n.InnerStart, n.InnerEnd, 0)
	}
	return n
}

// findClose finds the close tag of name from start. When nested is true, an
// open tag of the same name inside the content needs its own close tag. It
// returns the offset of the close tag and the offset after it.
func (p *parser) findClose(name string, start, end int, nested bool) (int, int) {
	closes := p.tagIndex("</" + name)
	opens := p.tagIndex("<" + name)
	ci := sort.SearchInts(closes, start)
	oi := sort.SearchInts(opens, start)
	depth := 0
	for ; ci < len(closes) && closes[ci] < end; ci++ {
		at := closes[ci]
		if nested {
			for oi < len(opens) && opens[oi] < at {
				o := opens[oi] + len(name) + 1
				if o < len(p.src) && (isSpaceByte(p.src[o]) || p.src[o] == '>') {
					depth++
				}
				oi++
			}
		}
		gt := strings.IndexByte(p.src[at:end], '>')
		if gt < 0 {
			return -1, -1
		}
		if strings.TrimSpace(p.src[at+len(name)+2:at+gt]) != "" {
			continue
		}
		if depth > 0 {
			depth--
			continue
		}
		return at, at + gt + 1
	}
	return -1, -1
}

// tagIndex returns the sorted offsets of a tag prefix such as "</ref" in the
// source, with no regard to case. It is computed once for each prefix.
func (p *parser) tagIndex(prefix string) []int {
	if v, ok := p.tagPos[prefix]; ok {
		return v
	}
	if p.lower == "" && p.src != "" {
		p.lower = asciiLower(p.src)
	}
	var out []int
	for i := 0; ; {
		k := strings.Index(p.lower[i:], prefix)
		if k < 0 {
			break
		}
		out = append(out, i+k)
		i += k + len(prefix)
	}
	if p.tagPos == nil {
		p.tagPos = map[string][]int{}
	}
	p.tagPos[prefix] = out
	return out
}

func isNameByte(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9'
}

func isSpaceByte(c byte) bool { return c == ' ' || c == '\t' || c == '\n' || c == '\r' }

// parseAttrs reads HTML attributes: name="value", name='value', name=value
// and name alone.
func parseAttrs(s string) map[string]string {
	var out map[string]string
	i := 0
	for i < len(s) {
		for i < len(s) && (isSpaceByte(s[i]) || s[i] == '/') {
			i++
		}
		st := i
		for i < len(s) && !isSpaceByte(s[i]) && s[i] != '=' {
			i++
		}
		name := strings.ToLower(s[st:i])
		if name == "" {
			i++
			continue
		}
		for i < len(s) && isSpaceByte(s[i]) {
			i++
		}
		val := ""
		if i < len(s) && s[i] == '=' {
			i++
			for i < len(s) && isSpaceByte(s[i]) {
				i++
			}
			if i < len(s) && (s[i] == '"' || s[i] == '\'') {
				q := s[i]
				j := strings.IndexByte(s[i+1:], q)
				if j < 0 {
					val = s[i+1:]
					i = len(s)
				} else {
					val = s[i+1 : i+1+j]
					i += j + 2
				}
			} else {
				vs := i
				for i < len(s) && !isSpaceByte(s[i]) {
					i++
				}
				val = s[vs:i]
			}
		}
		if out == nil {
			out = map[string]string{}
		}
		out[name] = val
	}
	return out
}

// asciiLower gives s with the ASCII letters in lower case. The tag names are
// ASCII. strings.ToLower can change the byte length of a letter, as for the
// letter İ, and then the offsets in the lower text do not match the source.
func asciiLower(s string) string {
	b := []byte(s)
	for i, c := range b {
		if 'A' <= c && c <= 'Z' {
			b[i] = c + 'a' - 'A'
		}
	}
	return string(b)
}
