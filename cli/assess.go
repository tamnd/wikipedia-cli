package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/tamnd/wikipedia-cli/wiki"
)

func newAssessCmd(app *App) *cobra.Command {
	var ns int
	cmd := &cobra.Command{
		Use:   "assess <project...>",
		Short: "List the pages that a WikiProject tags, with their assessment",
		Long: `List each page that one of the WikiProjects tags, with the class and the
importance that the project gives it. The command follows the list to the end,
or to --limit pages. Name a project as the API does, for example Mathematics or
"Philosophy/Logic". A page that two projects tag gives one row for each
project.

The API gives the subject page of each tagged talk page. --ns keeps the pages
of one namespace. The default is 0, the articles. Use --ns -1 to keep all.

Examples:
  wiki assess Mathematics -o tsv > mathematics.tsv
  wiki assess Statistics "Philosophy/Logic" Cryptography -o jsonl
  wiki assess Mathematics --ns 14 -n 20`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := app.Client()
			if err != nil {
				return err
			}
			n := 0
			err = c.ProjectPages(cmd.Context(), args, ns, app.Limit, func(p wiki.ProjectPage) error {
				n++
				if n%5000 == 0 && !app.quiet {
					fmt.Fprintf(os.Stderr, "%d pages\n", n)
				}
				return app.Out.Emit(Row{
					Cols:  []string{"pageid", "ns", "title", "project", "class", "importance"},
					Vals:  []string{itoa64(p.PageID), itoa(p.NS), p.Title, p.Project, p.Class, p.Importance},
					Value: p,
				})
			})
			if err != nil {
				return wrapErr(err)
			}
			if n == 0 {
				return noResults("no pages")
			}
			return app.Out.Flush()
		},
	}
	cmd.Flags().IntVar(&ns, "ns", 0, "keep the pages of this namespace (-1 for all)")
	return cmd
}
