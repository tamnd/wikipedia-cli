package texvc

import (
	"reflect"
	"strings"
	"testing"
)

func TestConvert(t *testing.T) {
	cases := []struct {
		name string
		form Form
		in   string
		want string
	}{
		// <math> content (texvc).
		{"reals power", FormMath, `\R^n`, `\mathbb{R}^n`},
		{"integers mod n", FormMath, `\Z/n\Z`, `\mathbb{Z}/n\mathbb{Z}`},
		{"number sets", FormMath, `\N \subset \Z \subset \Q \subset \R \subset \C`, `\mathbb{N} \subset \mathbb{Z} \subset \mathbb{Q} \subset \mathbb{R} \subset \mathbb{C}`},
		{"long set names", FormMath, `\reals \Reals \natnums \Complex \cnums`, `\mathbb{R} \mathbb{R} \mathbb{N} \mathbb{C} \mathbb{C}`},
		{"quaternions", FormMath, `\H`, `\mathbb{H}`},
		{"logic", FormMath, `p \and q \or r`, `p \land q \lor r`},
		{"partial", FormMath, `\frac{\part f}{\part x}`, `\frac{\partial f}{\partial x}`},
		{"sgn", FormMath, `\sgn(x)`, `\operatorname{sgn}(x)`},
		{"arc functions", FormMath, `\arccot x + \arcsec y + \arccsc z`, `\operatorname{arccot} x + \operatorname{arcsec} y + \operatorname{arccsc} z`},
		{"bold", FormMath, `\bold{v} \cdot \bold w`, `\mathbf{v} \cdot \mathbf w`},
		{"Bbb", FormMath, `\Bbb{F}_p`, `\mathbb{F}_p`},
		{"upper Greek", FormMath, `\Alpha \Beta \Epsilon \Zeta \Eta \Iota \Kappa \Mu \Nu \Omicron \Rho \Tau \Chi`, `\mathrm{A} \mathrm{B} \mathrm{E} \mathrm{Z} \mathrm{H} \mathrm{I} \mathrm{K} \mathrm{M} \mathrm{N} \mathrm{O} \mathrm{P} \mathrm{T} \mathrm{X}`},
		{"omicron", FormMath, `\omicron`, `o`},
		{"aleph", FormMath, `\alef_0 < \alefsym_1`, `\aleph_0 < \aleph_1`},
		{"arrows", FormMath, `\larr \rarr \lrarr \harr \Larr \Rarr \Lrarr \hArr \lArr \rArr \lrArr \Harr`, `\leftarrow \rightarrow \leftrightarrow \leftrightarrow \Leftarrow \Rightarrow \Leftrightarrow \Leftrightarrow \Leftarrow \Rightarrow \Leftrightarrow \Leftrightarrow`},
		{"vertical arrows", FormMath, `\uarr \darr \Uarr \Darr \uArr \dArr`, `\uparrow \downarrow \Uparrow \Downarrow \Uparrow \Downarrow`},
		{"sets", FormMath, `A \sub B \sube C \supe D`, `A \subset B \subseteq C \supseteq D`},
		{"symbols", FormMath, `\infin \isin \empty \exist \plusmn \thetasym \weierp \image \real \sdot`, `\infty \in \emptyset \exists \pm \vartheta \wp \Im \Re \cdot`},
		{"angle brackets", FormMath, `\lang x, y \rang`, `\langle x, y \rangle`},
		{"suits", FormMath, `\clubs \diamonds \hearts \spades \bull \Dagger`, `\clubsuit \diamondsuit \heartsuit \spadesuit \bullet \ddagger`},
		{"misc aliases", FormMath, `\ang \sect \restriction \doublecap \doublecup \gggtr \Doteq \O`, `\angle \S \upharpoonright \Cap \Cup \ggg \doteqdot \emptyset`},
		{"standard kept", FormMath, `\le \ge \ne \Re \alpha \mathbb{R} \Rightarrow \partial`, `\le \ge \ne \Re \alpha \mathbb{R} \Rightarrow \partial`},
		{"no partial match", FormMath, `\Reals\Rho \Rightarrow \Ra`, `\mathbb{R}\mathrm{P} \Rightarrow \Ra`},
		{"align", FormMath, `\begin{align} a &= b \\ c &= d \end{align}`, `\begin{aligned} a &= b \\ c &= d \end{aligned}`},
		{"align star and gather", FormMath, `\begin{align*} x \end{align*} \begin{gather} y \end{gather}`, `\begin{aligned} x \end{aligned} \begin{gathered} y \end{gathered}`},
		{"alignat", FormMath, `\begin{alignat}{2} a &= b \end{alignat}`, `\begin{alignedat}{2} a &= b \end{alignedat}`},
		{"other environments kept", FormMath, `\begin{pmatrix} 1 & 0 \\ 0 & 1 \end{pmatrix}`, `\begin{pmatrix} 1 & 0 \\ 0 & 1 \end{pmatrix}`},
		{"text group untouched", FormMath, `x \in \R \text{ for \and all}`, `x \in \mathbb{R} \text{ for \and all}`},
		{"euro", FormMath, `5\euro + \mbox{\euro 3}`, `5\text{€} + \mbox{€3}`},
		{"archaic Greek", FormMath, `\Digamma \koppa \sampi \varcoppa`, `\text{Ϝ} \text{ϟ} \text{ϡ} \text{ϙ}`},
		{"AA", FormMath, `1\,\AA`, `1\,\text{Å}`},
		{"escaped brace", FormMath, `\{ x \in \R : x > 0 \}`, `\{ x \in \mathbb{R} : x > 0 \}`},
		{"line break then letters", FormMath, `a \\R`, `a \\R`},
		{"whole element", FormMath, `<math>\R</math>`, `\mathbb{R}`},

		// {{tmath}}.
		{"tmath", FormTmath, `\R^n`, `\mathbb{R}^n`},
		{"tmath call", FormTmath, `{{tmath|x {{=}} \sgn y}}`, `x = \operatorname{sgn} y`},
		{"tmath named", FormTmath, `{{tmath|1=a = b \and c}}`, `a = b \land c`},
		{"tmath pipe", FormTmath, `{{tmath|{{!}}x{{!}}}}`, `|x|`},

		// {{mvar}}.
		{"mvar", FormMvar, `x`, `x`},
		{"mvar call", FormMvar, `{{mvar|G}}`, `G`},
		{"mvar greek", FormMvar, `{{mvar|φ}}`, `\varphi`},
		{"mvar word", FormMvar, `{{mvar|ab}}`, `ab`},
		{"mvar subscript", FormMvar, `x<sub>i</sub>`, `x_{i}`},

		// {{math}}.
		{"quotient group", FormMathTemplate, `{{math|''G''/''H''}}`, `G/H`},
		{"Lagrange", FormMathTemplate, `{{math|{{abs|''G''}} {{=}} [''G'' : ''H''] · {{abs|''H''}}}}`, `\lvert G\rvert = [G : H] \cdot \lvert H\rvert`},
		{"Fermat", FormMathTemplate, `{{math|''a''<sup>''p''−1</sup> ≡ 1 (mod ''p'')}}`, `a^{p-1} \equiv 1\pmod{p}`},
		{"bare mod", FormMathTemplate, `''a'' mod ''n''`, `a \bmod n`},
		{"bold vector", FormMathTemplate, `'''v''' ∈ ℝ<sup>3</sup>`, `\mathbf{v} \in \mathbb{R}^{3}`},
		{"bold italic", FormMathTemplate, `'''''v'''''`, `\boldsymbol{v}`},
		{"function words", FormMathTemplate, `sin ''x'' + log ''y''`, `\sin x + \log y`},
		{"text words", FormMathTemplate, `''x'' and ''y''`, `x\text{ and } y`},
		{"operator name", FormMathTemplate, `[[Galois group|Gal]](''L''/''K'')`, `\operatorname{Gal}(L/K)`},
		{"map", FormMathTemplate, `''f'' : ''A'' → ''B''`, `f : A \to B`},
		{"mapsto", FormMathTemplate, `''x'' ↦ ''x''<sup>2</sup>`, `x \mapsto x^{2}`},
		{"radic", FormMathTemplate, `{{radic|2}}`, `\sqrt{2}`},
		{"radic nth", FormMathTemplate, `{{radic|''x''|3}}`, `\sqrt[3]{x}`},
		{"sfrac", FormMathTemplate, `{{sfrac|1|2}}`, `\tfrac{1}{2}`},
		{"sfrac mixed", FormMathTemplate, `{{sfrac|3|1|4}}`, `3\tfrac{1}{4}`},
		{"sfrac one", FormMathTemplate, `{{sfrac|''n''}}`, `\tfrac{1}{n}`},
		{"overline", FormMathTemplate, `{{overline|''z''}}`, `\overline{z}`},
		{"norm", FormMathTemplate, `{{norm|'''x'''}}`, `\lVert \mathbf{x}\rVert`},
		{"ceil floor", FormMathTemplate, `{{ceil|''x''}} − {{floor|''x''}}`, `\lceil x\rceil - \lfloor x\rfloor`},
		{"angle templates", FormMathTemplate, `{{langle}}''u'', ''v''{{rangle}}`, `\langle u, v\rangle`},
		{"pi", FormMathTemplate, `2{{pi}}''r''`, `2\pi r`},
		{"nested mvar", FormMathTemplate, `{{mvar|n}} ≥ 1`, `n \ge 1`},
		{"nbsp entity", FormMathTemplate, `''a''&nbsp;+&nbsp;''b''`, `a + b`},
		{"minus entity", FormMathTemplate, `&minus;''x''`, `-x`},
		{"thin space template", FormMathTemplate, `''a''{{thinsp}}''b''`, `a\,b`},
		{"span dropped", FormMathTemplate, `<span class="texhtml">''x''</span> + 1`, `x + 1`},
		{"span overline", FormMathTemplate, `<span style="text-decoration:overline">''AB''</span>`, `\overline{AB}`},
		{"var tag", FormMathTemplate, `<var>x</var> + <i>y</i>`, `x + y`},
		{"percent", FormMathTemplate, `50%`, `50\%`},
		{"specials", FormMathTemplate, `#''S'' &amp; ''a''_''b''`, `\#S \& a\_b`},
		{"unicode scripts", FormMathTemplate, `''x''² + ''a''₁`, `x^{2} + a_{1}`},
		{"prime", FormMathTemplate, `''f''′(''x'')`, `f'(x)`},
		{"prime after superscript", FormMathTemplate, `''x''<sup>2</sup>′`, `x^{2}{}'`},
		{"sub then sup", FormMathTemplate, `''x''<sub>''i''</sub><sup>2</sup>`, `x_{i}^{2}`},
		{"double sup", FormMathTemplate, `''x''<sup>2</sup><sup>3</sup>`, `x^{2}{}^{3}`},
		{"su", FormMathTemplate, `''T''{{su|p=''i''|b=''j''}}`, `T^{i}_{j}`},
		{"sup template", FormMathTemplate, `''e''{{sup|''iπ''}}`, `e^{i\pi}`},
		{"set symbols", FormMathTemplate, `''A'' ∪ ''B'' ⊆ ''C'' ∩ ''D'', ''x'' ∉ ∅`, `A \cup B \subseteq C \cap D, x \notin \emptyset`},
		{"logic symbols", FormMathTemplate, `∀''x'' ∃''y'' ¬(''p'' ∧ ''q'') ⇒ ''r'' ∨ ''s'' ⇔ ''t''`, `\forall x \exists y \neg(p \land q) \Rightarrow r \lor s \Leftrightarrow t`},
		{"big operators", FormMathTemplate, `∑ ∏ ∫ √ ∂ ∇`, `\sum \prod \int \surd \partial \nabla`},
		{"relations", FormMathTemplate, `≠ ≈ ≡ ≅ ∼ ⊕ ⊗ ∘ ≤ ≥ ⊂`, `\ne \approx \equiv \cong \sim \oplus \otimes \circ \le \ge \subset`},
		{"blackboard", FormMathTemplate, `ℕ ℤ ℚ ℝ ℂ`, `\mathbb{N} \mathbb{Z} \mathbb{Q} \mathbb{R} \mathbb{C}`},
		{"greek", FormMathTemplate, `αβΓΔ`, `\alpha\beta\Gamma\Delta`},
		{"greek then letter", FormMathTemplate, `λ''x''`, `\lambda x`},
		{"times infinity", FormMathTemplate, `2 × ∞`, `2 \times \infty`},
		{"chevrons", FormMathTemplate, `⟨''x'', ''y''⟩`, `\langle x, y\rangle`},
		{"named size ignored", FormMathTemplate, `{{math|''x''|size=120%}}`, `x`},
		{"numbered param", FormMathTemplate, `{{math|1=''a'' = ''b''}}`, `a = b`},
		{"stray equals kept", FormMathTemplate, `''a'' = ''b''`, `a = b`},
		{"pipe template", FormMathTemplate, `{{!}}''x''{{!}}`, `|x|`},
		{"nested math tag", FormMathTemplate, `''x'' = <math>\R</math>`, `x = \mathbb{R}`},
		{"ref dropped", FormMathTemplate, `''x''<ref>Source.</ref>`, `x`},
		{"comment dropped", FormMathTemplate, `''x''<!-- note -->`, `x`},
		{"tmath inside math", FormMathTemplate, `{{tmath|\R}} ∋ ''x''`, `\mathbb{R} \ni x`},
		{"det", FormMathTemplate, `det(''A'') ≠ 0`, `\det(A) \ne 0`},
		{"upright d", FormMathTemplate, `d''x''`, `dx`},
		{"brackets", FormMathTemplate, `{''x'' : ''x'' > 0}`, `\{x : x > 0\}`},
		{"backslash", FormMathTemplate, `''A'' \ ''B''`, `A \backslash B`},
		{"setminus", FormMathTemplate, `''A'' ∖ ''B''`, `A \setminus B`},
		{"ellipsis", FormMathTemplate, `1, 2, …, ''n''`, `1, 2, \ldots, n`},
		{"plus minus", FormMathTemplate, `±1`, `\pm1`},
		{"control word before digit", FormMathTemplate, `∞1`, `\infty1`},
		{"link plain", FormMathTemplate, `[[Pi|π]] ≈ 3.14`, `\pi \approx 3.14`},
		{"accented letter", FormMathTemplate, `''x''é`, `x\text{é}`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := Convert(c.form, c.in)
			if got.TeX != c.want {
				t.Errorf("Convert(%v, %q)\n got %q\nwant %q", c.form, c.in, got.TeX, c.want)
			}
			if len(got.Errors) > 0 {
				t.Errorf("Convert(%v, %q) errors: %v", c.form, c.in, got.Errors)
			}
		})
	}
}

func TestDisplay(t *testing.T) {
	if r := FromMathTag(`x`, map[string]string{"display": "block"}); !r.Display {
		t.Error(`display="block" should set Display`)
	}
	if r := FromMathTag(`x`, map[string]string{"display": "inline"}); r.Display {
		t.Error(`display="inline" should not set Display`)
	}
	if r := FromMathTag(`x`, nil); r.Display {
		t.Error("no attributes should not set Display")
	}
	if r := Convert(FormMath, `<math display="block">\R</math>`); !r.Display || r.TeX != `\mathbb{R}` {
		t.Errorf("whole block element: %+v", r)
	}
	if r := FromMathTemplate(`''x''`); r.Display {
		t.Error("{{math}} is never display math")
	}
}

func TestCheck(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{`\frac{a}{b}`, nil},
		{`\left( x \right)`, nil},
		{`\left. x \right|`, nil},
		{`\left\{ x \right\}`, nil},
		{`\{ x \}`, nil},
		{`a \\ b`, nil},
		{`50\%`, nil},
		{`\begin{cases} 1 \end{cases}`, nil},
		{`\frac{a}{b`, []string{"unbalanced braces: { without }"}},
		{`a}`, []string{"unbalanced braces: } without {"}},
		{`\begin{aligned} a`, []string{`\begin{aligned} without \end`}},
		{`a \end{aligned}`, []string{`\end{aligned} without \begin`}},
		{`\begin{matrix} a \end{pmatrix}`, []string{`\end{pmatrix} closes \begin{matrix}`}},
		{`\left( x`, []string{`\left without \right`}},
		{`x \right)`, []string{`\right without \left`}},
		{`{\left( x} \right)`, []string{`\left without \right`, `\right without \left`}},
		{`a % comment {`, nil},
	}
	for _, c := range cases {
		if got := Check(c.in); !reflect.DeepEqual(got, c.want) {
			t.Errorf("Check(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestErrors(t *testing.T) {
	r := FromMathTemplate(`{{no such template|''x''}}`)
	if r.TeX != "x" || len(r.Errors) != 1 || !strings.Contains(r.Errors[0], "no such template") {
		t.Errorf("unknown template: %+v", r)
	}
	r = FromMathTag(`\begin{align} a`, nil)
	if len(r.Errors) != 1 {
		t.Errorf("unclosed environment: %+v", r)
	}
	r = FromMathTemplate(``)
	if len(r.Errors) != 1 || r.Errors[0] != "empty formula" {
		t.Errorf("empty: %+v", r)
	}
}

func TestDetect(t *testing.T) {
	cases := map[string]Form{
		`{{math|x}}`:          FormMathTemplate,
		`{{Math|x}}`:          FormMathTemplate,
		`{{mvar|x}}`:          FormMvar,
		`{{tmath|x}}`:         FormTmath,
		`<math>x</math>`:      FormMath,
		`\R^n`:                FormMath,
		` {{ math | ''x'' }}`: FormMathTemplate,
	}
	for in, want := range cases {
		if got := Detect(in); got != want {
			t.Errorf("Detect(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestParseForm(t *testing.T) {
	for _, n := range []string{"math", "template", "mvar", "tmath"} {
		f, err := ParseForm(n)
		if err != nil || f.String() != n {
			t.Errorf("ParseForm(%q) = %v, %v", n, f, err)
		}
	}
	if _, err := ParseForm("latex"); err == nil {
		t.Error("ParseForm(latex) should fail")
	}
}
