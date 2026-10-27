package cli

import (
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tamnd/wikipedia-cli/wiki/texvc"
)

func newTexvcCmd(app *App) *cobra.Command {
	var form string
	var display bool
	cmd := &cobra.Command{
		Use:   "texvc <tex | file | ->",
		Short: "Convert Wikipedia mathematics to standard TeX",
		Long: `Convert the mathematics that Wikipedia articles write into standard TeX
that KaTeX renders, and check that its braces, environments and \left/\right
pairs balance.

The input is the argument itself, a file whose name is the argument, or stdin
for "-". The form is one of:

  math       the content of a <math> tag, or the whole element (texvc)
  template   a {{math}} template, the whole call or its parameter text
  mvar       a {{mvar}} template
  tmath      a {{tmath}} template
  auto       guess from the input (the default)

Examples:
  wiki texvc '\R^n \to \Z/n\Z'
  wiki texvc '<math display="block">\begin{align} a &= b \end{align}</math>'
  wiki texvc "{{math|''a''<sup>''p''−1</sup> ≡ 1 (mod ''p'')}}"
  wiki texvc --form template "''G''/''H''"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			src, err := readTexvcInput(args[0])
			if err != nil {
				return err
			}
			var f texvc.Form
			if form == "" || form == "auto" {
				f = texvc.Detect(src)
			} else if f, err = texvc.ParseForm(form); err != nil {
				return usageErr(err.Error())
			}
			res := texvc.Convert(f, src)
			if display {
				res.Display = true
			}
			if err := app.Out.Emit(Row{
				Cols:  []string{"form", "display", "tex", "errors"},
				Vals:  []string{f.String(), boolStr(res.Display), res.TeX, strings.Join(res.Errors, "; ")},
				Value: texvcRow{f.String(), res},
			}); err != nil {
				return err
			}
			return app.Out.Flush()
		},
	}
	cmd.Flags().StringVar(&form, "form", "auto", "input form: auto|math|template|mvar|tmath")
	cmd.Flags().BoolVar(&display, "display", false, "mark the formula as display math")
	return cmd
}

// texvcRow is the record of one conversion.
type texvcRow struct {
	Form string `json:"form"`
	texvc.Result
}

// readTexvcInput reads the formula: stdin for "-", a file when the argument
// names one, or the argument itself.
func readTexvcInput(arg string) (string, error) {
	if arg == "-" {
		b, err := readAllStdin()
		return string(b), err
	}
	if !strings.ContainsAny(arg, "\\{}<|\n") {
		if b, err := os.ReadFile(arg); err == nil {
			return string(b), nil
		}
	}
	return arg, nil
}

func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
