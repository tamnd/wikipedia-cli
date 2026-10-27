package texvc

import "strings"

// aliases maps a texvc macro that standard TeX or KaTeX does not know to the
// standard TeX it stands for. The table follows texutil.json of the MediaWiki
// Math extension (src/WikiTexVC/texutil.json): the nullary_macro_aliase,
// deprecated_nullary_macro_aliase, other_delimiters2, other_fun_ar1 and
// mediawiki_function_names sets. \le, \ge and \ne are left out on purpose,
// because KaTeX knows them.
var aliases = map[string]string{
	// nullary_macro_aliase
	`\Alpha`:       `\mathrm{A}`,
	`\Beta`:        `\mathrm{B}`,
	`\Chi`:         `\mathrm{X}`,
	`\Epsilon`:     `\mathrm{E}`,
	`\Eta`:         `\mathrm{H}`,
	`\Iota`:        `\mathrm{I}`,
	`\Kappa`:       `\mathrm{K}`,
	`\Mu`:          `\mathrm{M}`,
	`\Nu`:          `\mathrm{N}`,
	`\Omicron`:     `\mathrm{O}`,
	`\Rho`:         `\mathrm{P}`,
	`\Tau`:         `\mathrm{T}`,
	`\Zeta`:        `\mathrm{Z}`,
	`\omicron`:     `o`,
	`\Complex`:     `\mathbb{C}`,
	`\cnums`:       `\mathbb{C}`,
	`\N`:           `\mathbb{N}`,
	`\natnums`:     `\mathbb{N}`,
	`\Q`:           `\mathbb{Q}`,
	`\R`:           `\mathbb{R}`,
	`\Reals`:       `\mathbb{R}`,
	`\reals`:       `\mathbb{R}`,
	`\Z`:           `\mathbb{Z}`,
	`\Dagger`:      `\ddagger`,
	`\Doteq`:       `\doteqdot`,
	`\Harr`:        `\Leftrightarrow`,
	`\hArr`:        `\Leftrightarrow`,
	`\Lrarr`:       `\Leftrightarrow`,
	`\lrArr`:       `\Leftrightarrow`,
	`\Larr`:        `\Leftarrow`,
	`\lArr`:        `\Leftarrow`,
	`\Rarr`:        `\Rightarrow`,
	`\rArr`:        `\Rightarrow`,
	`\harr`:        `\leftrightarrow`,
	`\lrarr`:       `\leftrightarrow`,
	`\larr`:        `\leftarrow`,
	`\rarr`:        `\rightarrow`,
	`\O`:           `\emptyset`,
	`\empty`:       `\emptyset`,
	`\alef`:        `\aleph`,
	`\alefsym`:     `\aleph`,
	`\bull`:        `\bullet`,
	`\clubs`:       `\clubsuit`,
	`\diamonds`:    `\diamondsuit`,
	`\hearts`:      `\heartsuit`,
	`\spades`:      `\spadesuit`,
	`\doublecap`:   `\Cap`,
	`\doublecup`:   `\Cup`,
	`\exist`:       `\exists`,
	`\gggtr`:       `\ggg`,
	`\image`:       `\Im`,
	`\real`:        `\Re`,
	`\infin`:       `\infty`,
	`\isin`:        `\in`,
	`\plusmn`:      `\pm`,
	`\restriction`: `\upharpoonright`,
	`\sdot`:        `\cdot`,
	`\sect`:        `\S`,
	`\sub`:         `\subset`,
	`\sube`:        `\subseteq`,
	`\supe`:        `\supseteq`,
	`\thetasym`:    `\vartheta`,
	`\weierp`:      `\wp`,
	`\varcoppa`:    `\text{ϙ}`,

	// deprecated_nullary_macro_aliase
	`\C`:    `\mathbb{C}`,
	`\H`:    `\mathbb{H}`,
	`\and`:  `\land`,
	`\or`:   `\lor`,
	`\ang`:  `\angle`,
	`\part`: `\partial`,

	// other_delimiters2
	`\Darr`: `\Downarrow`,
	`\dArr`: `\Downarrow`,
	`\darr`: `\downarrow`,
	`\Uarr`: `\Uparrow`,
	`\uArr`: `\Uparrow`,
	`\uarr`: `\uparrow`,
	`\lang`: `\langle`,
	`\rang`: `\rangle`,

	// other_fun_ar1
	`\bold`: `\mathbf`,
	`\Bbb`:  `\mathbb`,

	// mediawiki_function_names
	`\arccot`: `\operatorname{arccot}`,
	`\arccsc`: `\operatorname{arccsc}`,
	`\arcsec`: `\operatorname{arcsec}`,
	`\sen`:    `\operatorname{sen}`,
	`\sgn`:    `\operatorname{sgn}`,
}

// textChars maps the texvc macros that texvc draws in an \mbox (the
// nullary_macro_in_mbox set) to their character. Outside text they become
// \text{c}; inside a text group they become the bare character.
var textChars = map[string]string{
	`\AA`:               "Å",
	`\Coppa`:            "Ϙ",
	`\coppa`:            "ϙ",
	`\Digamma`:          "Ϝ",
	`\Koppa`:            "Ϟ",
	`\koppa`:            "ϟ",
	`\Sampi`:            "Ϡ",
	`\sampi`:            "ϡ",
	`\Stigma`:           "Ϛ",
	`\stigma`:           "ϛ",
	`\varstigma`:        "ϛ",
	`\euro`:             "€",
	`\geneuro`:          "€",
	`\geneuronarrow`:    "€",
	`\geneurowide`:      "€",
	`\officialeuro`:     "€",
	`\textvisiblespace`: "␣",
}

// environments maps the texvc environments that KaTeX allows only in display
// mode to the forms that work anywhere. texvc does the same rewrite.
var environments = map[string]string{
	"align":    "aligned",
	"align*":   "aligned",
	"alignat":  "alignedat",
	"alignat*": "alignedat",
	"gather":   "gathered",
	"gather*":  "gathered",
}

// textCommands take a text-mode argument; macros inside it are not math.
var textCommands = map[string]bool{
	`\text`: true, `\mbox`: true, `\hbox`: true, `\textrm`: true, `\textit`: true,
	`\textbf`: true, `\textsf`: true, `\texttt`: true, `\emph`: true,
}

// ExpandMacros rewrites the texvc macros in tex to standard TeX that KaTeX
// renders: \R becomes \mathbb{R}, \and becomes \land, \sgn becomes
// \operatorname{sgn}, \begin{align} becomes \begin{aligned}, and so on. The
// rest of the input is kept byte for byte.
func ExpandMacros(tex string) string {
	var b strings.Builder
	b.Grow(len(tex) + 16)
	expandInto(&b, tex, false)
	return b.String()
}

func expandInto(b *strings.Builder, s string, inText bool) {
	for i := 0; i < len(s); {
		c := s[i]
		if c != '\\' {
			b.WriteByte(c)
			i++
			continue
		}
		j := i + 1
		for j < len(s) && isASCIILetter(s[j]) {
			j++
		}
		if j == i+1 {
			// A control symbol such as \{ or \\: copy it whole.
			if j < len(s) {
				j++
			}
			b.WriteString(s[i:j])
			i = j
			continue
		}
		name := s[i:j]
		switch {
		case textChars[name] != "":
			if inText {
				b.WriteString(textChars[name])
			} else {
				b.WriteString(`\text{` + textChars[name] + `}`)
			}
			i = skipSpaceAfterWord(s, j, inText)
		case inText:
			b.WriteString(name)
			i = j
		case aliases[name] != "":
			b.WriteString(aliases[name])
			i = j
		case textCommands[name]:
			b.WriteString(name)
			k := j
			for k < len(s) && (s[k] == ' ' || s[k] == '\t' || s[k] == '\n') {
				k++
			}
			if k < len(s) && s[k] == '{' {
				end := matchBrace(s, k)
				b.WriteString(s[j : k+1])
				expandInto(b, s[k+1:end], true)
				if end < len(s) {
					b.WriteByte('}')
					end++
				}
				i = end
			} else {
				i = j
			}
		case name == `\begin` || name == `\end`:
			b.WriteString(name)
			i = j
			if j < len(s) && s[j] == '{' {
				if k := strings.IndexByte(s[j:], '}'); k > 0 {
					env := s[j+1 : j+k]
					if to, ok := environments[env]; ok {
						env = to
					}
					b.WriteString("{" + env + "}")
					i = j + k + 1
				}
			}
		default:
			b.WriteString(name)
			i = j
		}
	}
}

// skipSpaceAfterWord drops the spaces that end a control word in text mode, as
// TeX does, so "\euro 5" in a text group reads "€5".
func skipSpaceAfterWord(s string, j int, inText bool) int {
	if !inText {
		return j
	}
	for j < len(s) && s[j] == ' ' {
		j++
	}
	return j
}

// matchBrace returns the index of the brace that closes the one at open, or
// len(s) when it is not closed.
func matchBrace(s string, open int) int {
	depth := 0
	for i := open; i < len(s); i++ {
		switch s[i] {
		case '\\':
			i++
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return len(s)
}

func isASCIILetter(c byte) bool { return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' }
