package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/x/term"
	"github.com/mattn/go-isatty"
	"github.com/tamnd/any-cli/kit/render"
)

// Format is an output encoding. It aliases kit's render.Format so the command
// code keeps reading naturally (app.Out.Format() == FormatJSON) while the actual
// rendering is the shared kit engine — the same one every tamnd/*-cli uses.
type Format = render.Format

const (
	FormatAuto     = render.Auto
	FormatTable    = render.Table
	FormatMarkdown = render.Markdown
	FormatList     = render.List
	FormatJSON     = render.JSON
	FormatJSONL    = render.JSONL
	FormatCSV      = render.CSV
	FormatTSV      = render.TSV
	FormatURL      = render.URL
	FormatRaw      = render.Raw
)

// Row is one output record: an ordered, curated column set (Cols/Vals) for the
// list, table, csv, tsv, and url views plus the full typed value for json,
// jsonl, raw, and template. It is kit's render.Record, so every row builder
// feeds straight into the shared renderer with no per-format code of our own.
type Row = render.Record

// newRenderer builds the shared kit renderer over stdout from the global flags.
// It differs from kit's default in one place: with no -o, wiki prints the
// readable list/section view on a terminal (a table is a step away with
// -o table) and jsonl when piped, so scripts stay machine-readable. An explicit
// -o or --template always wins.
func newRenderer(g *globalFlags) (*render.Renderer, error) {
	isTTY := isatty.IsTerminal(os.Stdout.Fd())
	format := render.Format(g.output)
	if g.template == "" && (format == "" || format == FormatAuto) {
		if isTTY {
			format = FormatList
		} else {
			format = FormatJSONL
		}
	}
	var fields []string
	if g.fields != "" {
		fields = splitComma(g.fields)
	}
	return render.New(render.Options{
		Format:   format,
		IsTTY:    isTTY,
		Color:    colorEnabled(g.color, isTTY),
		Fields:   fields,
		NoHeader: g.noHeader,
		Template: g.template,
		Width:    termWidth(),
		Writer:   os.Stdout,
	})
}

// colorEnabled resolves --color against the terminal and the NO_COLOR
// convention: auto colors only an interactive terminal, always forces it on,
// never disables it. A pipe is not a terminal, so auto stays plain and
// `wiki search x | jq` never sees an escape code.
func colorEnabled(mode string, isTTY bool) bool {
	switch mode {
	case "always":
		return true
	case "never":
		return false
	default:
		return isTTY && os.Getenv("NO_COLOR") == ""
	}
}

// termWidth reports the terminal width in columns, or 0 when stdout is a pipe or
// file. The renderer uses it to shrink a too-wide table; 0 leaves output at its
// natural width, which is what a pipe wants. COLUMNS wins when set.
func termWidth() int {
	if v := os.Getenv("COLUMNS"); v != "" {
		var n int
		if _, err := fmt.Sscanf(v, "%d", &n); err == nil && n > 0 {
			return n
		}
	}
	if w, _, err := term.GetSize(os.Stdout.Fd()); err == nil && w > 0 {
		return w
	}
	return 0
}

func splitComma(s string) []string {
	var out []string
	for p := range strings.SplitSeq(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
