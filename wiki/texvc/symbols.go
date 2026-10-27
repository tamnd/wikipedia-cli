package texvc

// symbols maps the Unicode characters that {{math}} text uses to TeX.
var symbols = map[rune]string{
	// Greek, lower case.
	'α': `\alpha`, 'β': `\beta`, 'γ': `\gamma`, 'δ': `\delta`, 'ε': `\varepsilon`,
	'ϵ': `\epsilon`, 'ζ': `\zeta`, 'η': `\eta`, 'θ': `\theta`, 'ϑ': `\vartheta`,
	'ι': `\iota`, 'κ': `\kappa`, 'ϰ': `\varkappa`, 'λ': `\lambda`, 'μ': `\mu`,
	'µ': `\mu`, 'ν': `\nu`, 'ξ': `\xi`, 'ο': `o`, 'π': `\pi`, 'ϖ': `\varpi`,
	'ρ': `\rho`, 'ϱ': `\varrho`, 'σ': `\sigma`, 'ς': `\varsigma`, 'τ': `\tau`,
	'υ': `\upsilon`, 'φ': `\varphi`, 'ϕ': `\phi`, 'χ': `\chi`, 'ψ': `\psi`,
	'ω': `\omega`, 'ϝ': `\digamma`,
	// Greek, upper case. The ones that look like Latin letters become them.
	'Α': `\mathrm{A}`, 'Β': `\mathrm{B}`, 'Γ': `\Gamma`, 'Δ': `\Delta`, 'Ε': `\mathrm{E}`,
	'Ζ': `\mathrm{Z}`, 'Η': `\mathrm{H}`, 'Θ': `\Theta`, 'Ι': `\mathrm{I}`, 'Κ': `\mathrm{K}`,
	'Λ': `\Lambda`, 'Μ': `\mathrm{M}`, 'Ν': `\mathrm{N}`, 'Ξ': `\Xi`, 'Ο': `\mathrm{O}`,
	'Π': `\Pi`, 'Ρ': `\mathrm{P}`, 'Σ': `\Sigma`, 'Τ': `\mathrm{T}`, 'Υ': `\Upsilon`,
	'Φ': `\Phi`, 'Χ': `\mathrm{X}`, 'Ψ': `\Psi`, 'Ω': `\Omega`,

	// Letter-like symbols.
	'ℝ': `\mathbb{R}`, 'ℕ': `\mathbb{N}`, 'ℤ': `\mathbb{Z}`, 'ℚ': `\mathbb{Q}`,
	'ℂ': `\mathbb{C}`, 'ℍ': `\mathbb{H}`, 'ℙ': `\mathbb{P}`, '𝔽': `\mathbb{F}`,
	'𝔸': `\mathbb{A}`, '𝔼': `\mathbb{E}`, '𝕂': `\mathbb{K}`,
	'ℓ': `\ell`, 'ℏ': `\hbar`, 'ℵ': `\aleph`, 'ℶ': `\beth`, '℘': `\wp`,
	'ℑ': `\Im`, 'ℜ': `\Re`, 'ı': `\imath`, 'ȷ': `\jmath`, 'ð': `\eth`,
	'∂': `\partial`, '∇': `\nabla`, '∞': `\infty`, '∅': `\emptyset`, '⌀': `\varnothing`,
	'°': `^{\circ}`, '†': `\dagger`, '‡': `\ddagger`, '★': `\star`, '♯': `\sharp`,
	'♭': `\flat`, '♮': `\natural`, '§': `\S`, '¶': `\P`,

	// Relations.
	'≤': `\le`, '≥': `\ge`, '≦': `\leqq`, '≧': `\geqq`, '⩽': `\leqslant`, '⩾': `\geqslant`,
	'≠': `\ne`, '≈': `\approx`, '≡': `\equiv`, '≢': `\not\equiv`, '≅': `\cong`,
	'∼': `\sim`, '≃': `\simeq`, '∝': `\propto`, '≪': `\ll`, '≫': `\gg`,
	'≺': `\prec`, '≻': `\succ`, '⪯': `\preceq`, '⪰': `\succeq`, '≼': `\preccurlyeq`,
	'≽': `\succcurlyeq`, '≐': `\doteq`, '≔': `\coloneqq`, '≜': `\triangleq`,
	'∈': `\in`, '∉': `\notin`, '∋': `\ni`, '∌': `\not\ni`,
	'⊂': `\subset`, '⊃': `\supset`, '⊆': `\subseteq`, '⊇': `\supseteq`,
	'⊊': `\subsetneq`, '⊋': `\supsetneq`, '⊄': `\not\subset`, '⊈': `\nsubseteq`,
	'⊏': `\sqsubset`, '⊐': `\sqsupset`, '⊑': `\sqsubseteq`, '⊒': `\sqsupseteq`,
	'⊲': `\lhd`, '⊳': `\rhd`, '⊴': `\unlhd`, '⊵': `\unrhd`, '⋉': `\ltimes`, '⋊': `\rtimes`,
	'∣': `\mid`, '∤': `\nmid`, '∥': `\parallel`, '∦': `\nparallel`, '⊥': `\perp`,
	'⊢': `\vdash`, '⊣': `\dashv`, '⊨': `\models`, '⊩': `\Vdash`, '⊬': `\nvdash`,
	'≮': `\nless`, '≯': `\ngtr`, '≰': `\nleq`, '≱': `\ngeq`, '≍': `\asymp`,
	'≲': `\lesssim`, '≳': `\gtrsim`, '∽': `\backsim`, '≀': `\wr`,

	// Arrows.
	'→': `\to`, '←': `\leftarrow`, '↔': `\leftrightarrow`, '↦': `\mapsto`,
	'⇒': `\Rightarrow`, '⇐': `\Leftarrow`, '⇔': `\Leftrightarrow`, '⟹': `\implies`,
	'⟸': `\impliedby`, '⟺': `\iff`, '⟶': `\longrightarrow`, '⟵': `\longleftarrow`,
	'⟷': `\longleftrightarrow`, '⟼': `\longmapsto`, '↑': `\uparrow`, '↓': `\downarrow`,
	'⇑': `\Uparrow`, '⇓': `\Downarrow`, '↕': `\updownarrow`, '↪': `\hookrightarrow`,
	'↩': `\hookleftarrow`, '↠': `\twoheadrightarrow`, '↣': `\rightarrowtail`,
	'⇀': `\rightharpoonup`, '⇌': `\rightleftharpoons`, '↗': `\nearrow`, '↘': `\searrow`,
	'↙': `\swarrow`, '↖': `\nwarrow`, '↺': `\circlearrowleft`, '↻': `\circlearrowright`,
	'⇝': `\leadsto`, '↝': `\leadsto`,

	// Binary operators.
	'×': `\times`, '÷': `\div`, '·': `\cdot`, '⋅': `\cdot`, '∙': `\bullet`, '•': `\bullet`,
	'±': `\pm`, '∓': `\mp`, '−': `-`, '–': `-`, '∗': `\ast`, '∘': `\circ`, '○': `\circ`,
	'⊕': `\oplus`, '⊖': `\ominus`, '⊗': `\otimes`, '⊘': `\oslash`, '⊙': `\odot`,
	'∪': `\cup`, '∩': `\cap`, '⊔': `\sqcup`, '⊓': `\sqcap`, '⊎': `\uplus`,
	'∧': `\land`, '∨': `\lor`, '⋀': `\bigwedge`, '⋁': `\bigvee`, '⊻': `\veebar`,
	'∖': `\setminus`, '⧵': `\setminus`, '⋆': `\star`, '⋄': `\diamond`, '△': `\triangle`,
	'▽': `\triangledown`, '⊠': `\boxtimes`, '⊞': `\boxplus`, '∔': `\dotplus`,

	// Large operators.
	'∑': `\sum`, '∏': `\prod`, '∐': `\coprod`, '∫': `\int`, '∬': `\iint`, '∭': `\iiint`,
	'∮': `\oint`, '⋃': `\bigcup`, '⋂': `\bigcap`, '⨁': `\bigoplus`, '⨂': `\bigotimes`,
	'⨆': `\bigsqcup`, '√': `\surd`,

	// Logic.
	'∀': `\forall`, '∃': `\exists`, '∄': `\nexists`, '¬': `\neg`, '⊤': `\top`,
	'∴': `\therefore`, '∵': `\because`, '□': `\square`, '■': `\blacksquare`, '◊': `\lozenge`,

	// Delimiters and punctuation.
	'⟨': `\langle`, '⟩': `\rangle`, '〈': `\langle`, '〉': `\rangle`,
	'⌈': `\lceil`, '⌉': `\rceil`, '⌊': `\lfloor`, '⌋': `\rfloor`,
	'‖': `\|`, '⟦': `[\![`, '⟧': `]\!]`,
	'′': `'`, '″': `''`, '‴': `'''`, '…': `\ldots`, '⋯': `\cdots`, '⋮': `\vdots`,
	'⋱': `\ddots`, '∠': `\angle`, '∡': `\measuredangle`, '‰': `\text{‰}`,
	'¼': `\tfrac{1}{4}`, '½': `\tfrac{1}{2}`, '¾': `\tfrac{3}{4}`, '⅓': `\tfrac{1}{3}`,
	'⅔': `\tfrac{2}{3}`, '\u2002': `\enspace`, '\u2003': `\quad`,
	'\u2009': `\,`, '\u200a': `\,`, '\u202f': `\,`, '\u00a0': ` `,
	'\u2061': ``, '\u2062': ``, '\u200b': ``, '\u2060': ``,
}

// supDigits and subDigits map the Unicode superscript and subscript forms.
var supDigits = map[rune]byte{
	'⁰': '0', '¹': '1', '²': '2', '³': '3', '⁴': '4', '⁵': '5', '⁶': '6', '⁷': '7',
	'⁸': '8', '⁹': '9', '⁺': '+', '⁻': '-', '⁼': '=', '⁽': '(', '⁾': ')', 'ⁿ': 'n', 'ⁱ': 'i',
}

var subDigits = map[rune]byte{
	'₀': '0', '₁': '1', '₂': '2', '₃': '3', '₄': '4', '₅': '5', '₆': '6', '₇': '7',
	'₈': '8', '₉': '9', '₊': '+', '₋': '-', '₌': '=', '₍': '(', '₎': ')',
}

// functionNames are the upright operator names KaTeX has as macros.
var functionNames = map[string]bool{
	"sin": true, "cos": true, "tan": true, "cot": true, "sec": true, "csc": true,
	"sinh": true, "cosh": true, "tanh": true, "coth": true, "arcsin": true,
	"arccos": true, "arctan": true, "arg": true, "deg": true, "det": true, "dim": true,
	"exp": true, "gcd": true, "hom": true, "inf": true, "ker": true, "lg": true,
	"lim": true, "liminf": true, "limsup": true, "ln": true, "log": true, "max": true,
	"min": true, "Pr": true, "sup": true,
}

// textWords are the English words that {{math}} text uses between formulas.
// They stay words, in \text.
var textWords = map[string]bool{
	"and": true, "or": true, "if": true, "iff": true, "for": true, "all": true,
	"where": true, "then": true, "otherwise": true, "with": true, "some": true,
	"is": true, "of": true, "not": true, "such": true, "that": true, "when": true,
	"in": true, "on": true, "as": true, "to": true, "the": true, "an": true,
	"else": true, "by": true, "every": true, "each": true,
	"any": true, "are": true, "be": true, "has": true, "have": true, "there": true,
	"exists": true, "but": true, "so": true, "at": true, "from": true, "even": true,
	"odd": true, "prime": true, "times": true, "whenever": true, "unless": true,
	"over": true,
}
