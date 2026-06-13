package wiki

import (
	"bytes"
	"net/url"
	"regexp"
	"strings"
	"sync"
)

// WikitextToMarkdown renders MediaWiki wikitext (the source you get from a dump
// or prop=wikitext) into clean Markdown. It is pragmatic, not a MediaWiki
// reimplementation: templates, tables, references and citation chrome are
// dropped, while headings, lists, bold/italic, code blocks, internal and
// external links and paragraphs are converted. lang sets the host used for
// internal-link URLs; an empty lang collapses internal links to their label so
// the output stays wiki-agnostic.
func WikitextToMarkdown(src, lang string) string {
	if src == "" {
		return ""
	}
	return finishMarkdown(stripCommon(src), lang) + "\n"
}

// WikitextToText renders wikitext into readable plain text, dropping all markup
// (but keeping the contents of code blocks).
func WikitextToText(src string) string {
	if src == "" {
		return ""
	}
	return finishText(stripCommon(src)) + "\n"
}

// ConvertBoth renders wikitext to Markdown and plain text sharing one strip
// pass. Cheaper than calling the two converters separately.
func ConvertBoth(src, lang string) (md, txt string) {
	if src == "" {
		return "", ""
	}
	stripped := stripCommon(src)
	return finishMarkdown(stripped, lang) + "\n", finishText(stripped) + "\n"
}

// bufPool reuses buffers across conversions; a full dump runs the converter
// millions of times and the buffers dominate allocation otherwise.
var bufPool = sync.Pool{New: func() any { return new(bytes.Buffer) }}

var langAttrRe = regexp.MustCompile(`lang\s*=\s*["']?([^"'\s>]+)`)

// stripCommon removes the constructs that have no Markdown analogue in a single
// character scan: HTML comments, <nowiki> and <ref>, templates {{...}}, tables
// {|...|} (both nesting), and File/Image/Category/interwiki links. What remains
// is text plus the inline markup the finish passes understand.
func stripCommon(src string) string {
	n := len(src)
	if n == 0 {
		return ""
	}
	buf := bufPool.Get().(*bytes.Buffer)
	buf.Reset()
	buf.Grow(n / 2)

	for i := 0; i < n; {
		c := src[i]

		if c == '<' {
			// HTML comment.
			if strings.HasPrefix(src[i:], "<!--") {
				if end := strings.Index(src[i+4:], "-->"); end >= 0 {
					i += 4 + end + 3
				} else {
					i = n
				}
				continue
			}
			// <nowiki>...</nowiki>: drop the tags, keep nothing (literal markup).
			if strings.HasPrefix(src[i:], "<nowiki>") {
				if end := strings.Index(src[i+8:], "</nowiki>"); end >= 0 {
					i += 8 + end + 9
				} else {
					i = n
				}
				continue
			}
			// <ref ...>...</ref> or <ref .../>.
			if strings.HasPrefix(src[i:], "<ref") {
				next := byte(' ')
				if i+4 < n {
					next = src[i+4]
				}
				if next == ' ' || next == '>' || next == '/' || next == '\n' || next == '\t' {
					if gt := strings.IndexByte(src[i+4:], '>'); gt >= 0 {
						tagEnd := i + 4 + gt
						if src[tagEnd-1] == '/' {
							i = tagEnd + 1
						} else if end := strings.Index(src[tagEnd+1:], "</ref>"); end >= 0 {
							i = tagEnd + 1 + end + 6
						} else {
							i = tagEnd + 1
						}
						continue
					}
				}
			}
		}

		// Templates {{ ... }} with nesting.
		if c == '{' && i+1 < n && src[i+1] == '{' {
			if end := scanBalanced(src, i, '{', '}'); end > i {
				i = end
				continue
			}
			buf.WriteString("{{")
			i += 2
			continue
		}
		// Tables {| ... |} with nesting.
		if c == '{' && i+1 < n && src[i+1] == '|' {
			if end := scanBalancedPair(src, i, "{|", "|}"); end > i {
				i = end
				continue
			}
			buf.WriteString("{|")
			i += 2
			continue
		}

		// [[ ... ]] links: drop File/Image/Category/interwiki wholesale.
		if c == '[' && i+1 < n && src[i+1] == '[' {
			colon := -1
			for k := i + 2; k < n; k++ {
				ch := src[k]
				if ch == ':' {
					colon = k
					break
				}
				if ch == '|' || ch == ']' || ch == '\n' {
					break
				}
			}
			if colon > i+2 && isStripLinkPrefix(strings.ToLower(src[i+2:colon])) {
				if end := scanBalancedPair(src, i, "[[", "]]"); end > i {
					i = end
					continue
				}
			}
			buf.WriteString("[[")
			i += 2
			continue
		}

		buf.WriteByte(c)
		i++
	}
	out := buf.String()
	bufPool.Put(buf)
	return out
}

// scanBalanced returns the index just past the matching close for a doubled
// open/close (e.g. "{{"/"}}"), or i if unbalanced.
func scanBalanced(s string, i int, open, close byte) int {
	n := len(s)
	depth := 1
	j := i + 2
	for j+1 < n && depth > 0 {
		if s[j] == open && s[j+1] == open {
			depth++
			j += 2
		} else if s[j] == close && s[j+1] == close {
			depth--
			j += 2
		} else {
			j++
		}
	}
	if depth == 0 {
		return j
	}
	return i
}

// scanBalancedPair is scanBalanced for two-byte open/close tokens that differ
// ("{|"/"|}", "[["/"]]").
func scanBalancedPair(s string, i int, open, close string) int {
	n := len(s)
	depth := 1
	j := i + 2
	for j+1 < n && depth > 0 {
		if s[j] == open[0] && s[j+1] == open[1] {
			depth++
			j += 2
		} else if s[j] == close[0] && s[j+1] == close[1] {
			depth--
			j += 2
		} else {
			j++
		}
	}
	if depth == 0 {
		return j
	}
	return i
}

// isStripLinkPrefix reports whether a lowercase [[prefix:...]] should drop the
// whole link: the File/Image/Category namespaces (a handful of languages) and
// short all-letter interwiki codes like "fr" or "ja".
func isStripLinkPrefix(prefix string) bool {
	switch prefix {
	case "file", "image", "media", "fichier", "archivo", "datei",
		"файл", "ファイル", "文件", "bestand", "hình", "tập tin":
		return true
	case "category", "catégorie", "categoría", "kategorie",
		"категория", "カテゴリ", "分类", "categorie", "thể loại":
		return true
	}
	if len(prefix) >= 2 && len(prefix) <= 10 {
		for _, c := range prefix {
			if c < 'a' || c > 'z' {
				return false
			}
		}
		return true
	}
	return false
}

var mdLinkPrefixes = []string{
	"file:", "image:", "media:", "category:", "fichier:", "archivo:",
	"datei:", "категория:", "categoría:", "catégorie:", "categorie:",
}

// finishMarkdown converts the stripped wikitext to Markdown in one scan.
func finishMarkdown(s, lang string) string {
	if lang == "" {
		lang = "en"
	}
	n := len(s)
	if n == 0 {
		return ""
	}
	buf := bufPool.Get().(*bytes.Buffer)
	buf.Reset()
	buf.Grow(n * 5 / 4)

	for i := 0; i < n; {
		c := s[i]
		atLineStart := i == 0 || s[i-1] == '\n'

		// Code: <syntaxhighlight>/<source>/<pre> → fenced; <code> → inline.
		if c == '<' {
			if adv, ok := writeCodeBlockMD(buf, s, i); ok {
				i = adv
				continue
			}
			// Any other tag: strip it.
			if gt := strings.IndexByte(s[i+1:], '>'); gt >= 0 {
				i += 1 + gt + 1
				continue
			}
		}

		// Lists and indentation at line start.
		if atLineStart && (c == '*' || c == '#' || c == ':' || c == ';') {
			depth := 0
			j := i
			for j < n && (s[j] == '*' || s[j] == '#' || s[j] == ':' || s[j] == ';') {
				depth++
				j++
			}
			for j < n && s[j] == ' ' {
				j++
			}
			buf.WriteString(listPrefix(s[i:i+depth], true))
			i = j
			continue
		}

		// [[Page|Display]] → [Display](url), dropping special namespaces.
		if c == '[' && i+1 < n && s[i+1] == '[' {
			if end := strings.Index(s[i+2:], "]]"); end >= 0 {
				writeWikiLinkMD(buf, s[i+2:i+2+end], lang)
				i += 2 + end + 2
				continue
			}
		}
		// [url text] → [text](url).
		if c == '[' && i+1 < n && s[i+1] == 'h' {
			if adv, ok := writeExtLinkMD(buf, s, i); ok {
				i = adv
				continue
			}
		}

		// == Heading == → ## Heading.
		if c == '=' && atLineStart {
			if adv, ok := writeHeadingMD(buf, s, i); ok {
				i = adv
				continue
			}
		}

		// '''bold''' / ''italic'' / '''''both'''''.
		if c == '\'' && i+1 < n && s[i+1] == '\'' {
			if adv, ok := writeEmphasisMD(buf, s, i, lang); ok {
				i = adv
				continue
			}
		}

		// Collapse 3+ newlines to a paragraph break.
		if c == '\n' && i+2 < n && s[i+1] == '\n' && s[i+2] == '\n' {
			buf.WriteString("\n\n")
			for i += 2; i < n && s[i] == '\n'; i++ {
			}
			continue
		}

		buf.WriteByte(c)
		i++
	}
	out := decodeEntities(strings.TrimSpace(buf.String()))
	bufPool.Put(buf)
	return out
}

// finishText converts stripped wikitext to plain text in one scan, keeping the
// contents of code blocks but dropping every marker.
func finishText(s string) string {
	n := len(s)
	if n == 0 {
		return ""
	}
	buf := bufPool.Get().(*bytes.Buffer)
	buf.Reset()
	buf.Grow(n)

	for i := 0; i < n; {
		c := s[i]
		atLineStart := i == 0 || s[i-1] == '\n'

		if c == '<' {
			if adv, ok := writeCodeBlockText(buf, s, i); ok {
				i = adv
				continue
			}
			if gt := strings.IndexByte(s[i+1:], '>'); gt >= 0 {
				i += 1 + gt + 1
				continue
			}
		}

		if atLineStart && (c == '*' || c == '#' || c == ':' || c == ';') {
			depth := 0
			j := i
			for j < n && (s[j] == '*' || s[j] == '#' || s[j] == ':' || s[j] == ';') {
				depth++
				j++
			}
			for j < n && s[j] == ' ' {
				j++
			}
			buf.WriteString(listPrefix(s[i:i+depth], false))
			i = j
			continue
		}

		if c == '[' && i+1 < n && s[i+1] == '[' {
			if end := strings.Index(s[i+2:], "]]"); end >= 0 {
				content := s[i+2 : i+2+end]
				if pipe := strings.IndexByte(content, '|'); pipe >= 0 {
					buf.WriteString(strings.TrimSpace(content[pipe+1:]))
				} else {
					buf.WriteString(strings.TrimSpace(content))
				}
				i += 2 + end + 2
				continue
			}
		}
		if c == '[' && i+1 < n && s[i+1] == 'h' {
			rest := s[i+1:]
			if strings.HasPrefix(rest, "http://") || strings.HasPrefix(rest, "https://") {
				if end := strings.IndexByte(rest, ']'); end >= 0 {
					content := rest[:end]
					if sp := strings.IndexByte(content, ' '); sp >= 0 {
						buf.WriteString(content[sp+1:])
					}
					i += 1 + end + 1
					continue
				}
			}
		}

		if c == '=' && atLineStart {
			if adv, ok := writeHeadingText(buf, s, i); ok {
				i = adv
				continue
			}
		}

		if c == '\'' && i+1 < n && s[i+1] == '\'' {
			if adv, ok := writeEmphasisText(buf, s, i); ok {
				i = adv
				continue
			}
		}

		if c == '\n' && i+2 < n && s[i+1] == '\n' && s[i+2] == '\n' {
			buf.WriteString("\n\n")
			for i += 2; i < n && s[i] == '\n'; i++ {
			}
			continue
		}

		buf.WriteByte(c)
		i++
	}
	out := decodeEntities(strings.TrimSpace(buf.String()))
	bufPool.Put(buf)
	return out
}

// listPrefix maps a wikitext marker run ("*", "##", ":", ";") to a Markdown (or
// plain) list prefix with two-space indent per level.
func listPrefix(markers string, md bool) string {
	depth := len(markers)
	indent := strings.Repeat("  ", depth-1)
	switch markers[depth-1] {
	case '#':
		return indent + "1. "
	case ':':
		if md {
			return indent + "> "
		}
		return indent
	case ';':
		return indent
	default:
		return indent + "- "
	}
}

func writeWikiLinkMD(buf *bytes.Buffer, content, lang string) {
	page, display := content, content
	if pipe := strings.IndexByte(content, '|'); pipe >= 0 {
		page = content[:pipe]
		display = content[pipe+1:]
	}
	page = strings.TrimSpace(page)
	display = strings.TrimSpace(display)
	if strings.ContainsRune(page, ':') {
		lower := strings.ToLower(page)
		for _, pfx := range mdLinkPrefixes {
			if strings.HasPrefix(lower, pfx) {
				return
			}
		}
	}
	if idx := strings.IndexByte(page, '#'); idx >= 0 {
		page = page[:idx]
	}
	if page == "" {
		buf.WriteString(display)
		return
	}
	buf.WriteByte('[')
	buf.WriteString(display)
	buf.WriteString("](https://")
	buf.WriteString(lang)
	buf.WriteString(".wikipedia.org/wiki/")
	buf.WriteString(url.PathEscape(strings.ReplaceAll(page, " ", "_")))
	buf.WriteByte(')')
}

func writeExtLinkMD(buf *bytes.Buffer, s string, i int) (int, bool) {
	rest := s[i+1:]
	if !strings.HasPrefix(rest, "http://") && !strings.HasPrefix(rest, "https://") {
		return i, false
	}
	end := strings.IndexByte(rest, ']')
	if end < 0 {
		return i, false
	}
	content := rest[:end]
	if sp := strings.IndexByte(content, ' '); sp >= 0 {
		buf.WriteByte('[')
		buf.WriteString(strings.TrimSpace(content[sp+1:]))
		buf.WriteString("](")
		buf.WriteString(content[:sp])
		buf.WriteByte(')')
	} else {
		buf.WriteString(content)
	}
	return i + 1 + end + 1, true
}

func writeHeadingMD(buf *bytes.Buffer, s string, i int) (int, bool) {
	level, j := 0, i
	for j < len(s) && s[j] == '=' && level < 7 {
		level++
		j++
	}
	if level < 2 || level > 6 {
		return i, false
	}
	lineEnd := strings.IndexByte(s[j:], '\n')
	if lineEnd < 0 {
		lineEnd = len(s) - j
	}
	heading := strings.TrimSpace(strings.TrimRight(s[j:j+lineEnd], "= \t\r"))
	buf.WriteString(strings.Repeat("#", level))
	buf.WriteByte(' ')
	buf.WriteString(heading)
	return j + lineEnd, true
}

func writeHeadingText(buf *bytes.Buffer, s string, i int) (int, bool) {
	level, j := 0, i
	for j < len(s) && s[j] == '=' && level < 7 {
		level++
		j++
	}
	if level < 2 || level > 6 {
		return i, false
	}
	lineEnd := strings.IndexByte(s[j:], '\n')
	if lineEnd < 0 {
		lineEnd = len(s) - j
	}
	buf.WriteString(strings.TrimSpace(strings.TrimRight(s[j:j+lineEnd], "= \t\r")))
	return j + lineEnd, true
}

// writeEmphasisMD handles ”/”'/””' runs in Markdown mode, emitting */**/***
// and recursively converting the inner span so links and nested emphasis inside
// bold or italic are rendered, not copied verbatim.
func writeEmphasisMD(buf *bytes.Buffer, s string, i int, lang string) (int, bool) {
	return emphasis(buf, s, i, func(b *bytes.Buffer, inner string) {
		writeInlineMD(b, inner, lang)
	}, true)
}

// writeEmphasisText handles emphasis runs in plain-text mode, dropping the
// markers but still resolving links inside.
func writeEmphasisText(buf *bytes.Buffer, s string, i int) (int, bool) {
	return emphasis(buf, s, i, writeInlineText, false)
}

func emphasis(buf *bytes.Buffer, s string, i int, inner func(*bytes.Buffer, string), md bool) (int, bool) {
	n := len(s)
	q := 0
	for j := i; j < n && s[j] == '\'' && q < 5; j++ {
		q++
	}
	try := func(count int, mark string) (int, bool) {
		tok := strings.Repeat("'", count)
		if end := strings.Index(s[i+count:], tok); end >= 0 {
			if md {
				buf.WriteString(mark)
			}
			inner(buf, s[i+count:i+count+end])
			if md {
				buf.WriteString(mark)
			}
			return i + count + end + count, true
		}
		return i, false
	}
	if q >= 5 {
		if adv, ok := try(5, "***"); ok {
			return adv, true
		}
	}
	if q >= 3 {
		if adv, ok := try(3, "**"); ok {
			return adv, true
		}
	}
	if q >= 2 {
		if adv, ok := try(2, "*"); ok {
			return adv, true
		}
	}
	return i, false
}

// writeInlineMD converts the inline constructs (wiki links, external links,
// nested emphasis) inside a span such as the text wrapped by bold or italic.
func writeInlineMD(buf *bytes.Buffer, s, lang string) {
	n := len(s)
	for i := 0; i < n; {
		c := s[i]
		if c == '[' && i+1 < n && s[i+1] == '[' {
			if end := strings.Index(s[i+2:], "]]"); end >= 0 {
				writeWikiLinkMD(buf, s[i+2:i+2+end], lang)
				i += 2 + end + 2
				continue
			}
		}
		if c == '[' && i+1 < n && s[i+1] == 'h' {
			if adv, ok := writeExtLinkMD(buf, s, i); ok {
				i = adv
				continue
			}
		}
		if c == '\'' && i+1 < n && s[i+1] == '\'' {
			if adv, ok := writeEmphasisMD(buf, s, i, lang); ok {
				i = adv
				continue
			}
		}
		buf.WriteByte(c)
		i++
	}
}

// writeInlineText resolves links inside an emphasis span for plain-text output.
func writeInlineText(buf *bytes.Buffer, s string) {
	n := len(s)
	for i := 0; i < n; {
		c := s[i]
		if c == '[' && i+1 < n && s[i+1] == '[' {
			if end := strings.Index(s[i+2:], "]]"); end >= 0 {
				content := s[i+2 : i+2+end]
				if pipe := strings.IndexByte(content, '|'); pipe >= 0 {
					buf.WriteString(strings.TrimSpace(content[pipe+1:]))
				} else {
					buf.WriteString(strings.TrimSpace(content))
				}
				i += 2 + end + 2
				continue
			}
		}
		if c == '\'' && i+1 < n && s[i+1] == '\'' {
			if adv, ok := writeEmphasisText(buf, s, i); ok {
				i = adv
				continue
			}
		}
		buf.WriteByte(c)
		i++
	}
}

// writeCodeBlockMD emits a fenced code block (or inline code) for the code tag
// starting at i, returning the advance and whether a tag was handled.
func writeCodeBlockMD(buf *bytes.Buffer, s string, i int) (int, bool) {
	switch {
	case strings.HasPrefix(s[i:], "<syntaxhighlight"):
		return fenceMD(buf, s, i, "<syntaxhighlight", "</syntaxhighlight>")
	case strings.HasPrefix(s[i:], "<source"):
		return fenceMD(buf, s, i, "<source", "</source>")
	case strings.HasPrefix(s[i:], "<pre>"):
		if end := strings.Index(s[i+5:], "</pre>"); end >= 0 {
			buf.WriteString("\n```\n")
			buf.WriteString(strings.TrimSpace(s[i+5 : i+5+end]))
			buf.WriteString("\n```\n")
			return i + 5 + end + 6, true
		}
	case strings.HasPrefix(s[i:], "<code>"):
		if end := strings.Index(s[i+6:], "</code>"); end >= 0 {
			buf.WriteByte('`')
			buf.WriteString(s[i+6 : i+6+end])
			buf.WriteByte('`')
			return i + 6 + end + 7, true
		}
	}
	return i, false
}

// fenceMD renders a <syntaxhighlight>/<source> block, reading its lang attr.
func fenceMD(buf *bytes.Buffer, s string, i int, openTag, closeTag string) (int, bool) {
	end := strings.Index(s[i:], closeTag)
	if end < 0 {
		return i, false
	}
	gt := strings.IndexByte(s[i:i+end], '>')
	if gt < 0 {
		return i, false
	}
	attrs := s[i+len(openTag) : i+gt]
	code := strings.Trim(s[i+gt+1:i+end], "\n")
	lang := ""
	if m := langAttrRe.FindStringSubmatch(attrs); len(m) > 1 {
		lang = strings.ToLower(m[1])
	}
	buf.WriteString("\n```")
	buf.WriteString(lang)
	buf.WriteByte('\n')
	buf.WriteString(code)
	buf.WriteString("\n```\n")
	return i + end + len(closeTag), true
}

func writeCodeBlockText(buf *bytes.Buffer, s string, i int) (int, bool) {
	keep := func(closeTag string) (int, bool) {
		end := strings.Index(s[i:], closeTag)
		if end < 0 {
			return i, false
		}
		if gt := strings.IndexByte(s[i:i+end], '>'); gt >= 0 {
			buf.WriteString(strings.TrimSpace(s[i+gt+1 : i+end]))
		}
		return i + end + len(closeTag), true
	}
	switch {
	case strings.HasPrefix(s[i:], "<syntaxhighlight"):
		return keep("</syntaxhighlight>")
	case strings.HasPrefix(s[i:], "<source"):
		return keep("</source>")
	case strings.HasPrefix(s[i:], "<pre>"):
		if end := strings.Index(s[i+5:], "</pre>"); end >= 0 {
			buf.WriteString(strings.TrimSpace(s[i+5 : i+5+end]))
			return i + 5 + end + 6, true
		}
	case strings.HasPrefix(s[i:], "<code>"):
		if end := strings.Index(s[i+6:], "</code>"); end >= 0 {
			buf.WriteString(s[i+6 : i+6+end])
			return i + 6 + end + 7, true
		}
	}
	return i, false
}

func decodeEntities(s string) string {
	if !strings.ContainsRune(s, '&') {
		return s
	}
	return entityReplacer.Replace(s)
}

var entityReplacer = strings.NewReplacer(
	"&nbsp;", " ", "&amp;", "&", "&lt;", "<", "&gt;", ">",
	"&quot;", `"`, "&apos;", "'", "&ndash;", "-", "&mdash;", "-",
	"&minus;", "-", "&times;", "x", "&deg;", "°",
	"&rarr;", "->", "&larr;", "<-", "&hellip;", "...", "&middot;", "·",
)
