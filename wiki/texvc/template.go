package texvc

import (
	"html"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

// conv turns the wikitext of {{math}} and {{mvar}} into TeX. It keeps the
// problems it meets in errs.
type conv struct {
	errs []string
}

// out is the TeX being written. It collapses spaces, puts a space between a
// control word and a letter, and guards against a double superscript.
type out struct {
	b       []byte
	space   bool // a space is pending
	scripts byte // 1 for ^ and 2 for _ already on the last atom
}

func (o *out) write(s string) {
	if s == "" {
		return
	}
	if o.space {
		if len(o.b) > 0 {
			o.b = append(o.b, ' ')
		}
		o.space = false
	} else if endsWithControlWord(o.b) && isASCIILetter(s[0]) {
		o.b = append(o.b, ' ')
	}
	o.b = append(o.b, s...)
	o.scripts = 0
}

// script writes ^{inner} or _{inner} on the last atom.
func (o *out) script(op byte, inner string) {
	o.space = false
	bit := byte(1)
	if op == '_' {
		bit = 2
	}
	if o.scripts&bit != 0 || len(o.b) == 0 {
		o.b = append(o.b, '{', '}')
		o.scripts = 0
	}
	o.b = append(o.b, op, '{')
	o.b = append(o.b, inner...)
	o.b = append(o.b, '}')
	o.scripts |= bit
}

// prime writes a prime, which TeX reads as a superscript too.
func (o *out) prime(n int) {
	if o.scripts&1 != 0 {
		o.b = append(o.b, '{', '}')
	}
	o.space = false
	o.b = append(o.b, strings.Repeat("'", n)...)
	o.scripts = 0
}

func (o *out) endsWith(s string) bool {
	return strings.HasSuffix(strings.TrimRight(string(o.b), " "), s)
}

func (o *out) String() string { return strings.TrimSpace(string(o.b)) }

func endsWithControlWord(b []byte) bool {
	i := len(b)
	for i > 0 && isASCIILetter(b[i-1]) {
		i--
	}
	if i == len(b) || i == 0 || b[i-1] != '\\' {
		return false
	}
	// "\\x" is a line break and a letter, not a control word.
	n := 0
	for k := i - 1; k >= 0 && b[k] == '\\'; k-- {
		n++
	}
	return n%2 == 1
}

var (
	tagRe    = regexp.MustCompile(`^<(/?)([A-Za-z][A-Za-z0-9]*)\b([^<>]*?)(/?)>`)
	entityRe = regexp.MustCompile(`^&(#[0-9]+|#[xX][0-9A-Fa-f]+|[A-Za-z][A-Za-z0-9]*);`)
)

// convert converts the wikitext s. italic and bold are the quote state that
// s starts in.
func (c *conv) convert(s string, italic, bold bool) string {
	var o out
	opened := 0 // bold groups opened in s
	setBold := func(on bool) {
		switch {
		case on && !bold:
			if italic {
				o.write(`\boldsymbol{`)
			} else {
				o.write(`\mathbf{`)
			}
			opened++
		case !on && bold && opened > 0:
			o.write("}")
			opened--
		}
		bold = on
	}
	for i := 0; i < len(s); {
		rest := s[i:]
		switch {
		case strings.HasPrefix(rest, "<!--"):
			end := strings.Index(rest, "-->")
			if end < 0 {
				i = len(s)
			} else {
				i += end + 3
			}
		case strings.HasPrefix(rest, "{{"):
			end := matchDouble(s, i, "{{", "}}")
			if end < 0 {
				o.write(`\{`)
				i++
				continue
			}
			tex := c.template(s[i+2:end], italic, bold)
			for len(tex) > 1 && (tex[0] == '^' || tex[0] == '_') && tex[1] == '{' {
				close := matchBrace(tex, 1)
				if close >= len(tex) {
					break
				}
				o.script(tex[0], tex[2:close])
				tex = tex[close+1:]
			}
			o.write(tex)
			i = end + 2
		case strings.HasPrefix(rest, "[["):
			end := matchDouble(s, i, "[[", "]]")
			if end < 0 {
				o.write("[")
				i++
				continue
			}
			o.write(c.link(s[i+2:end], italic, bold))
			i = end + 2
		case s[i] == '\'':
			n := 1
			for i+n < len(s) && s[i+n] == '\'' {
				n++
			}
			i += n
			switch {
			case n == 1:
				o.prime(1)
			case n == 2:
				italic = !italic
			case n == 3:
				setBold(!bold)
			case n == 4:
				o.prime(1)
				setBold(!bold)
			default:
				if n > 5 {
					o.prime(n - 5)
				}
				// Close the inner one first so the groups nest.
				if bold {
					setBold(false)
					italic = !italic
				} else {
					italic = !italic
					setBold(true)
				}
			}
		case s[i] == '<':
			n := c.tag(s, i, &o, italic, bold)
			if n == 0 {
				o.write("<")
				n = 1
			}
			i += n
		case s[i] == '&':
			m := entityRe.FindString(rest)
			dec := html.UnescapeString(m)
			if m == "" || dec == m {
				o.write(`\&`)
				i++
				continue
			}
			for _, r := range dec {
				c.char(&o, r)
			}
			i += len(m)
		case isASCIILetter(s[i]):
			j := i
			for j < len(s) && isASCIILetter(s[j]) {
				j++
			}
			word := s[i:j]
			i = j
			if word == "mod" && !italic && !bold {
				i = c.mod(s, i, &o)
				continue
			}
			c.word(&o, word, italic || bold, i < len(s) && (s[i] == ' ' || s[i] == '\t'))
		default:
			r, size := utf8.DecodeRuneInString(rest)
			if d, ok := supDigits[r]; ok {
				run := []byte{d}
				for i += size; i < len(s); i += size {
					r, size = utf8.DecodeRuneInString(s[i:])
					d, ok := supDigits[r]
					if !ok {
						break
					}
					run = append(run, d)
				}
				o.script('^', string(run))
				continue
			}
			if d, ok := subDigits[r]; ok {
				run := []byte{d}
				for i += size; i < len(s); i += size {
					r, size = utf8.DecodeRuneInString(s[i:])
					d, ok := subDigits[r]
					if !ok {
						break
					}
					run = append(run, d)
				}
				o.script('_', string(run))
				continue
			}
			c.char(&o, r)
			i += size
		}
	}
	for opened > 0 {
		o.write("}")
		opened--
	}
	return o.String()
}

// word writes a run of ASCII letters. Upright words of two letters or more are
// operator names or English words; italic and bold runs are variables.
func (c *conv) word(o *out, w string, variable, spaceAfter bool) {
	switch {
	case variable || len(w) == 1:
		o.write(w)
	case functionNames[w]:
		o.write(`\` + w)
	case textWords[w]:
		t := w
		if o.space {
			t = " " + t
			o.space = false
		}
		if spaceAfter {
			t += " "
		}
		o.write(`\text{` + t + `}`)
	default:
		o.write(`\operatorname{` + w + `}`)
	}
}

// mod writes an upright "mod" that starts at s[:i]. "(mod n)" becomes
// \pmod{n}; a bare "mod" becomes \bmod. It returns where to go on.
func (c *conv) mod(s string, i int, o *out) int {
	if o.endsWith("(") {
		if end := strings.IndexByte(s[i:], ')'); end >= 0 && !strings.Contains(s[i:i+end], "(") {
			inner := c.convert(s[i:i+end], false, false)
			t := strings.TrimRight(string(o.b), " ")
			o.b = []byte(strings.TrimRight(t[:len(t)-1], " "))
			o.space = false
			o.write(`\pmod{` + inner + `}`)
			return i + end + 1
		}
	}
	o.write(`\bmod`)
	return i
}

// char writes one character that is not part of a letter run or markup.
func (c *conv) char(o *out, r rune) {
	if t, ok := symbols[r]; ok {
		if t == " " {
			o.space = true
		} else if t == "'" || t == "''" || t == "'''" {
			o.prime(len(t))
		} else {
			o.write(t)
		}
		return
	}
	if unicode.IsSpace(r) {
		o.space = true
		return
	}
	switch r {
	case '%', '#', '$', '_', '&', '{', '}':
		o.write(`\` + string(r))
	case '\\':
		o.write(`\backslash`)
	case '~':
		o.write(`\sim`)
	case '^':
		o.write(`\hat{}`)
	case '"':
		o.write(`''`)
	default:
		if r < utf8.RuneSelf {
			o.write(string(r))
		} else {
			o.write(`\text{` + string(r) + `}`)
		}
	}
}

// tag converts the HTML tag at s[i:] and returns the bytes it used, or 0 when
// s[i:] is not a tag.
func (c *conv) tag(s string, i int, o *out, italic, bold bool) int {
	m := tagRe.FindStringSubmatch(s[i:])
	if m == nil {
		return 0
	}
	closing, name, attrs, selfClose := m[1] != "", strings.ToLower(m[2]), m[3], m[4] != ""
	n := len(m[0])
	if closing {
		return n // a stray end tag
	}
	switch name {
	case "br", "wbr", "hr":
		o.space = true
		return n
	}
	if selfClose {
		return n
	}
	start := i + n
	end, after := matchTag(s, start, name)
	inner := s[start:end]
	used := after - i
	switch name {
	case "sup":
		o.script('^', c.convert(inner, italic, bold))
	case "sub":
		o.script('_', c.convert(inner, italic, bold))
	case "i", "var", "em":
		o.write(c.convert(inner, true, bold))
	case "b", "strong":
		if bold {
			o.write(c.convert(inner, italic, true))
		} else if italic {
			o.write(`\boldsymbol{` + c.convert(inner, italic, true) + `}`)
		} else {
			o.write(`\mathbf{` + c.convert(inner, italic, true) + `}`)
		}
	case "u":
		o.write(`\underline{` + c.convert(inner, italic, bold) + `}`)
	case "span":
		t := c.convert(inner, italic, bold)
		if strings.Contains(attrs, "overline") {
			t = `\overline{` + t + `}`
		}
		o.write(t)
	case "ref":
	case "math":
		o.write(ExpandMacros(strings.TrimSpace(inner)))
	default:
		o.write(c.convert(inner, italic, bold))
	}
	return used
}

// matchTag finds the end tag of name that closes the tag whose content starts
// at start. It returns where the content ends and where the end tag ends.
func matchTag(s string, start int, name string) (int, int) {
	depth := 1
	lower := strings.ToLower(s)
	for i := start; i < len(s); i++ {
		if s[i] != '<' {
			continue
		}
		m := tagRe.FindStringSubmatch(lower[i:])
		if m == nil || m[2] != name || m[4] != "" {
			continue
		}
		if m[1] == "" {
			depth++
			continue
		}
		depth--
		if depth == 0 {
			return i, i + len(m[0])
		}
	}
	return len(s), len(s)
}

// link converts the body of a wikilink to its label.
func (c *conv) link(body string, italic, bold bool) string {
	parts := splitTop(body)
	target := strings.ToLower(strings.TrimSpace(parts[0]))
	for _, ns := range []string{"file:", "image:", "category:"} {
		if strings.HasPrefix(target, ns) {
			return ""
		}
	}
	return c.convert(parts[len(parts)-1], italic, bold)
}

// matchDouble returns the index of the close that matches the open at s[i:],
// or -1.
func matchDouble(s string, i int, open, close string) int {
	depth := 0
	for k := i; k+1 < len(s); {
		switch {
		case strings.HasPrefix(s[k:], open):
			depth++
			k += 2
		case strings.HasPrefix(s[k:], close):
			depth--
			if depth == 0 {
				return k
			}
			k += 2
		default:
			k++
		}
	}
	return -1
}

// splitTop splits a template or link body at the pipes that are not inside a
// nested template, link, <math> or <nowiki>.
func splitTop(body string) []string {
	var parts []string
	depth, last := 0, 0
	for i := 0; i < len(body); i++ {
		switch {
		case strings.HasPrefix(body[i:], "{{") || strings.HasPrefix(body[i:], "[["):
			depth++
			i++
		case (strings.HasPrefix(body[i:], "}}") || strings.HasPrefix(body[i:], "]]")) && depth > 0:
			depth--
			i++
		case body[i] == '<':
			for _, t := range []string{"math", "nowiki"} {
				if strings.HasPrefix(strings.ToLower(body[i:]), "<"+t) {
					if end := strings.Index(strings.ToLower(body[i:]), "</"+t+">"); end > 0 {
						i += end + len(t) + 2
					}
				}
			}
		case body[i] == '|' && depth == 0:
			parts = append(parts, body[last:i])
			last = i + 1
		}
	}
	return append(parts, body[last:])
}

// namedParams are the parameter names that the math templates take. Any other
// "x=" in a parameter is read as part of the mathematics, which is what an
// editor who forgot {{=}} meant.
var namedParams = map[string]bool{
	"size": true, "style": true, "class": true, "p": true, "b": true, "a": true,
	"w": true, "id": true, "ref": true,
}

// params splits template parameters into positional and named ones.
func params(parts []string) (pos []string, named map[string]string) {
	named = map[string]string{}
	for _, p := range parts {
		if eq := strings.IndexByte(p, '='); eq > 0 {
			key := strings.TrimSpace(p[:eq])
			if isDigits(key) {
				for len(pos) < atoi(key) {
					pos = append(pos, "")
				}
				pos[atoi(key)-1] = p[eq+1:]
				continue
			}
			if namedParams[key] {
				named[key] = p[eq+1:]
				continue
			}
		}
		pos = append(pos, p)
	}
	return pos, named
}

func isDigits(s string) bool {
	if s == "" || len(s) > 3 {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return s != "0"
}

func atoi(s string) int {
	n := 0
	for i := 0; i < len(s); i++ {
		n = n*10 + int(s[i]-'0')
	}
	return n
}

// templateName normalizes a template name for lookup.
func templateName(s string) string {
	s = strings.TrimSpace(strings.ReplaceAll(s, "_", " "))
	s = strings.Join(strings.Fields(s), " ")
	s = strings.ToLower(s)
	return strings.TrimPrefix(s, "template:")
}

// template converts a template inside {{math}} text. body is the text between
// the braces.
func (c *conv) template(body string, italic, bold bool) string {
	parts := splitTop(body)
	name := templateName(parts[0])
	pos, named := params(parts[1:])
	arg := func(n int) string {
		if n < len(pos) {
			return c.convert(pos[n], italic, bold)
		}
		return ""
	}
	switch name {
	case "=":
		return "="
	case "!":
		return "|"
	case "!!":
		return "||"
	case "math", "nowrap", "nobr", "nobreak", "bigmath", "tooltip", "abbr", "small", "big", "nobold", "noitalic":
		return arg(0)
	case "mvar", "var":
		if len(pos) > 0 {
			return c.convert(pos[0], true, bold)
		}
		return ""
	case "tmath":
		if len(pos) > 0 {
			return ExpandMacros(strings.TrimSpace(unescapeTemplates(pos[0])))
		}
		return ""
	case "radic", "sqrt":
		if len(pos) > 1 && strings.TrimSpace(pos[1]) != "" {
			return `\sqrt[` + arg(1) + `]{` + arg(0) + `}`
		}
		return `\sqrt{` + arg(0) + `}`
	case "sfrac", "frac", "fract":
		switch len(pos) {
		case 0:
			return `/`
		case 1:
			return `\tfrac{1}{` + arg(0) + `}`
		case 2:
			return `\tfrac{` + arg(0) + `}{` + arg(1) + `}`
		default:
			return arg(0) + `\tfrac{` + arg(1) + `}{` + arg(2) + `}`
		}
	case "overline":
		return `\overline{` + arg(0) + `}`
	case "underline":
		return `\underline{` + arg(0) + `}`
	case "vec":
		return `\vec{` + arg(0) + `}`
	case "hat":
		return `\hat{` + arg(0) + `}`
	case "abs":
		return `\lvert ` + arg(0) + `\rvert`
	case "norm":
		return `\lVert ` + arg(0) + `\rVert`
	case "ceil":
		return `\lceil ` + arg(0) + `\rceil`
	case "floor":
		return `\lfloor ` + arg(0) + `\rfloor`
	case "angbr", "angle bracket", "angbr ip":
		return `\langle ` + arg(0) + `\rangle`
	case "langle":
		return `\langle`
	case "rangle":
		return `\rangle`
	case "bra":
		return `\langle ` + arg(0) + `\rvert`
	case "ket":
		return `\lvert ` + arg(0) + `\rangle`
	case "bra-ket", "braket":
		return `\langle ` + arg(0) + `\mid ` + arg(1) + `\rangle`
	case "pi":
		return `\pi`
	case "sup":
		return `^{` + arg(0) + `}`
	case "sub":
		return `_{` + arg(0) + `}`
	case "su":
		var t string
		if p, ok := named["p"]; ok {
			t += `^{` + c.convert(p, italic, bold) + `}`
		}
		if b, ok := named["b"]; ok {
			t += `_{` + c.convert(b, italic, bold) + `}`
		}
		return t
	case "thinsp", "thin space", "hair space", "hsp", "narrow no-break space":
		return `\,`
	case "nbsp", "spaces", "space":
		return `\ `
	case "ic", "italic correction", "zwsp", "nowrap begin", "nowrap end", "wbr", "sic":
		return ""
	case "times":
		return `\times`
	case "minus":
		return `-`
	case "plusminus", "pm":
		return `\pm`
	case "degree":
		return `^{\circ}`
	}
	c.errs = append(c.errs, "template {{"+strings.TrimSpace(parts[0])+"}} not converted")
	return arg(0)
}

// unescapeTemplates replaces {{=}}, {{!}} and {{!!}}, which TeX in a template
// parameter uses for =, | and ||.
func unescapeTemplates(s string) string {
	return strings.NewReplacer("{{=}}", "=", "{{!}}", "|", "{{!!}}", "||").Replace(s)
}
