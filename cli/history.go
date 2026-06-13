package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tamnd/wikipedia-cli/wiki"
)

func newRevisionsCmd(app *App) *cobra.Command {
	var user string
	cmd := &cobra.Command{
		Use:     "revisions <title>",
		Aliases: []string{"history", "log"},
		Short:   "List a page's revision history (newest first)",
		Long: `List the revision history of a page: id, timestamp, author, byte size and
edit summary, newest first. Filter to one author with --user.

Examples:
  wiki revisions "Go (programming language)" -n 20
  wiki revisions "Climate change" --user Jimbo -o jsonl`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, title, err := app.resolveTarget(args[0])
			if err != nil {
				return err
			}
			revs, err := c.Revisions(cmd.Context(), title, app.Limit, user)
			if err != nil {
				return wrapErr(err)
			}
			if len(revs) == 0 {
				return noResults("no revisions")
			}
			for _, r := range revs {
				if err := app.Out.Emit(Row{
					Cols:  []string{"revid", "parentid", "timestamp", "user", "userid", "size", "minor", "comment", "tags"},
					Vals:  []string{itoa(r.RevID), itoa(r.ParentID), r.Timestamp, r.User, itoa(r.UserID), itoa(r.Size), boolWord(r.Minor, "m", ""), r.Comment, r.TagsLine()},
					Value: r,
				}); err != nil {
					return err
				}
			}
			return app.Out.Flush()
		},
	}
	cmd.Flags().StringVar(&user, "user", "", "only revisions by this user")
	return cmd
}

func newDiffCmd(app *App) *cobra.Command {
	var to string
	cmd := &cobra.Command{
		Use:   "diff <from-revid> [to-revid]",
		Short: "Show a unified diff between two revisions",
		Long: `Show the difference between two revisions of a page.

Give one revision id and the diff is against its parent; give two, or use
--to with prev/next/cur, to compare explicitly.

Examples:
  wiki diff 123456789
  wiki diff 123456789 123460000
  wiki diff 123456789 --to next`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := app.Client()
			if err != nil {
				return err
			}
			from, err := atoiArg(args[0])
			if err != nil {
				return usageErr("from-revid must be a number")
			}
			target := to
			if len(args) == 2 {
				target = args[1]
			}
			if target == "" {
				target = "prev"
			}
			lines, err := c.Diff(cmd.Context(), from, target)
			if err != nil {
				return wrapErr(err)
			}
			if len(lines) == 0 {
				return noResults("no differences")
			}
			return emitDiff(app, lines)
		},
	}
	cmd.Flags().StringVar(&to, "to", "", "compare against prev|next|cur or a revid")
	return cmd
}

// emitDiff prints diff lines, as colored +/- text for humans or rows for data.
func emitDiff(app *App, lines []wiki.DiffLine) error {
	switch app.Out.Format() {
	case FormatJSON, FormatJSONL, FormatCSV, FormatTSV:
		for _, l := range lines {
			if err := app.Out.Emit(Row{
				Cols:  []string{"op", "text"},
				Vals:  []string{l.Op, l.Text},
				Value: l,
			}); err != nil {
				return err
			}
		}
		return app.Out.Flush()
	}
	for _, l := range lines {
		var prefix string
		switch l.Op {
		case "add":
			prefix = "+"
		case "del":
			prefix = "-"
		default:
			prefix = " "
		}
		if _, err := fmt.Fprintf(cmdOut, "%s %s\n", prefix, l.Text); err != nil {
			return err
		}
	}
	return nil
}
