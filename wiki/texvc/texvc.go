// Package texvc converts the mathematics that Wikipedia articles write into
// standard TeX that KaTeX renders.
//
// Wikipedia writes mathematics in four forms: the <math> tag, whose content is
// texvc (TeX with extra macros such as \R and \and); the {{math}} template,
// whose content is wikitext with HTML and Unicode symbols; the {{mvar}}
// template, which holds one italic variable; and the {{tmath}} template, whose
// content is texvc like the <math> tag. Each From function turns one form into
// TeX and runs a small balance check on it. The real parse check with KaTeX
// happens later, outside this package.
package texvc

import (
	"fmt"
	"regexp"
	"strings"
)

// Form is one of the ways Wikipedia writes mathematics.
type Form int

// The forms.
const (
	FormMath         Form = iota // <math>...</math>, texvc
	FormMathTemplate             // {{math|...}}, wikitext
	FormMvar                     // {{mvar|...}}, one variable
	FormTmath                    // {{tmath|...}}, texvc
)

var formNames = []string{"math", "template", "mvar", "tmath"}

func (f Form) String() string {
	if f >= 0 && int(f) < len(formNames) {
		return formNames[f]
	}
	return fmt.Sprintf("Form(%d)", int(f))
}

// ParseForm reads a form name: math, template, mvar or tmath.
func ParseForm(s string) (Form, error) {
	for i, n := range formNames {
		if strings.EqualFold(s, n) {
			return Form(i), nil
		}
	}
	return 0, fmt.Errorf("unknown form %q (want math, template, mvar or tmath)", s)
}

// Result is the TeX for one formula.
type Result struct {
	TeX     string   `json:"tex"`
	Display bool     `json:"display"`
	Errors  []string `json:"errors,omitempty"`
}

// FromMathTag converts the content of a <math> tag. attrs are the tag's
// attributes; display="block" sets Display.
func FromMathTag(src string, attrs map[string]string) Result {
	tex := ExpandMacros(strings.TrimSpace(src))
	return finish(tex, strings.EqualFold(strings.TrimSpace(attrs["display"]), "block"), nil)
}

// FromMathTemplate converts a {{math}} template. src is either the whole call,
// "{{math|...}}", or the text between "{{math|" and the closing "}}". In the
// second form, as in wikitext, a bare | starts a new parameter; the formula is
// the first positional parameter (or 1=), and named parameters such as size=
// are ignored.
func FromMathTemplate(src string) Result {
	c := &conv{}
	tex := c.convert(templateArg(src, "math"), false, false)
	return finish(tex, false, c.errs)
}

// FromMvar converts a {{mvar}} template. src is read as in FromMathTemplate.
// The content is italic, so letter runs are variables.
func FromMvar(src string) Result {
	c := &conv{}
	tex := c.convert(templateArg(src, "mvar"), true, false)
	return finish(tex, false, c.errs)
}

// FromTmath converts a {{tmath}} template. src is read as in
// FromMathTemplate. {{=}} and {{!}} become = and |, and the rest is texvc as
// in a <math> tag.
func FromTmath(src string) Result {
	arg := unescapeTemplates(templateArg(src, "tmath"))
	return FromMathTag(arg, nil)
}

// Convert converts src in the given form. For FormMath, src may be the whole
// <math ...>...</math> element or only its content.
func Convert(form Form, src string) Result {
	switch form {
	case FormMathTemplate:
		return FromMathTemplate(src)
	case FormMvar:
		return FromMvar(src)
	case FormTmath:
		return FromTmath(src)
	}
	if body, attrs, ok := ParseMathTag(src); ok {
		return FromMathTag(body, attrs)
	}
	return FromMathTag(src, nil)
}

// Detect guesses the form of src from its start: a <math> tag, or a {{math}},
// {{mvar}} or {{tmath}} call. Anything else is read as texvc.
func Detect(src string) Form {
	s := strings.TrimSpace(src)
	if strings.HasPrefix(s, "{{") {
		name := s[2:]
		if i := strings.IndexAny(name, "|}"); i >= 0 {
			name = name[:i]
		}
		switch templateName(name) {
		case "math":
			return FormMathTemplate
		case "mvar":
			return FormMvar
		case "tmath":
			return FormTmath
		}
	}
	return FormMath
}

var (
	mathTagRe = regexp.MustCompile(`(?is)^\s*<math(\s[^>]*)?>(.*)</math\s*>\s*$`)
	attrRe    = regexp.MustCompile(`([A-Za-z_:-]+)\s*=\s*(?:"([^"]*)"|'([^']*)'|([^\s"'>]+))`)
)

// ParseMathTag splits a whole <math ...>...</math> element into its content
// and its attributes. ok is false when s is not one such element.
func ParseMathTag(s string) (body string, attrs map[string]string, ok bool) {
	m := mathTagRe.FindStringSubmatch(s)
	if m == nil {
		return "", nil, false
	}
	attrs = map[string]string{}
	for _, a := range attrRe.FindAllStringSubmatch(m[1], -1) {
		attrs[strings.ToLower(a[1])] = a[2] + a[3] + a[4]
	}
	return m[2], attrs, true
}

// templateArg returns the formula parameter of a template call. src is the
// whole call when it is wrapped in {{name|...}} with one of names, or the
// parameter text otherwise.
func templateArg(src string, names ...string) string {
	s := strings.TrimSpace(src)
	var parts []string
	if strings.HasPrefix(s, "{{") && matchDouble(s, 0, "{{", "}}") == len(s)-2 {
		all := splitTop(s[2 : len(s)-2])
		for _, n := range names {
			if templateName(all[0]) == n {
				parts = all[1:]
			}
		}
	}
	if parts == nil {
		parts = splitTop(s)
	}
	pos, _ := params(parts)
	if len(pos) == 0 {
		return ""
	}
	return pos[0]
}

func finish(tex string, display bool, errs []string) Result {
	tex = strings.TrimSpace(tex)
	errs = append(errs, Check(tex)...)
	if tex == "" {
		errs = append(errs, "empty formula")
	}
	return Result{TeX: tex, Display: display, Errors: errs}
}
