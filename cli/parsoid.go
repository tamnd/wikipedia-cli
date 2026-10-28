package cli

import (
	"bytes"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/tamnd/wikipedia-cli/wiki"
	"github.com/tamnd/wikipedia-cli/wiki/parsoid"
)

func newParsoidCmd(app *App) *cobra.Command {
	var rev int
	var raw bool
	cmd := &cobra.Command{
		Use:   "parsoid <title>",
		Short: "Fetch the Parsoid HTML of a revision and read it",
		Long: `Fetch the Parsoid HTML of a revision from the REST API, and read the
formulas, the templates, the sections, the anchors and the references from
it. The fetch stays at or below 200 requests a minute.

With --html, the command writes the HTML itself.

Examples:
  wiki parsoid "Lagrange's theorem (group theory)"
  wiki parsoid --rev 1321322691 "Lagrange's theorem (group theory)"
  wiki parsoid --html Coset > coset.html`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			site, err := app.Cfg.Site()
			if err != nil {
				return usageErr(err.Error())
			}
			ua := app.Cfg.UserAgent
			if ua == "" {
				ua = wiki.UserAgent
			}
			f := &parsoid.Fetcher{Base: "https://" + site.Host + "/api/rest_v1", UserAgent: ua}
			body, err := f.Fetch(cmd.Context(), args[0], rev)
			if err != nil {
				return err
			}
			if raw {
				_, err := cmd.OutOrStdout().Write(body)
				return err
			}
			r, err := parsoid.Read(bytes.NewReader(body))
			if err != nil {
				return err
			}
			if err := app.Out.Emit(Row{
				Cols:  []string{"page", "revision", "math", "templates", "sections", "anchors", "refs"},
				Vals:  []string{strconv.Itoa(r.Page), strconv.Itoa(r.Revision), strconv.Itoa(len(r.Math)), strconv.Itoa(len(r.Transclusions)), strconv.Itoa(len(r.Sections)), strconv.Itoa(len(r.Anchors)), strconv.Itoa(r.Refs)},
				Value: r,
			}); err != nil {
				return err
			}
			return app.Out.Flush()
		},
	}
	cmd.Flags().IntVar(&rev, "rev", 0, "the revision id. The default is the current revision.")
	cmd.Flags().BoolVar(&raw, "html", false, "write the HTML, not the reading")
	return cmd
}
