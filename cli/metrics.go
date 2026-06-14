package cli

import (
	"time"

	"github.com/spf13/cobra"
)

func newPageviewsCmd(app *App) *cobra.Command {
	var from, to, granularity, access, agent string
	var days int
	cmd := &cobra.Command{
		Use:     "pageviews <title>",
		Aliases: []string{"views", "pv"},
		Short:   "Show the pageview time series for an article",
		Long: `Show daily or monthly pageviews for an article between two dates.

Defaults to the last 30 days. Use --from/--to (YYYY-MM-DD) for an explicit
window, --granularity daily|monthly, and --access/--agent to filter.

Examples:
  wiki pageviews "Alan Turing"
  wiki pageviews "Pi" --from 2024-01-01 --to 2024-03-31 --granularity monthly
  wiki pageviews "Cat" --days 90 -o csv`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, title, err := app.resolveTarget(args[0])
			if err != nil {
				return err
			}
			start, end, err := pageviewWindow(from, to, days)
			if err != nil {
				return usageErr(err.Error())
			}
			sp := app.progress("fetching pageviews")
			pts, err := c.Pageviews(cmd.Context(), title, granularity, access, agent, start, end)
			sp.stop()
			if err != nil {
				return wrapErr(err)
			}
			if len(pts) == 0 {
				return noResults("no pageview data")
			}
			for _, p := range pts {
				if err := app.Out.Emit(Row{
					Cols:  []string{"date", "views"},
					Vals:  []string{p.Date, itoa(p.Views)},
					Value: p,
				}); err != nil {
					return err
				}
			}
			return app.Out.Flush()
		},
	}
	cmd.Flags().StringVar(&from, "from", "", "start date YYYY-MM-DD")
	cmd.Flags().StringVar(&to, "to", "", "end date YYYY-MM-DD")
	cmd.Flags().IntVar(&days, "days", 30, "window length when --from is omitted")
	cmd.Flags().StringVar(&granularity, "granularity", "daily", "daily|monthly")
	cmd.Flags().StringVar(&access, "access", "all-access", "all-access|desktop|mobile-web|mobile-app")
	cmd.Flags().StringVar(&agent, "agent", "all-agents", "all-agents|user|spider|automated")
	return cmd
}

// pageviewWindow resolves the from/to dates, defaulting to the last `days` days
// ending yesterday (today's data is incomplete).
func pageviewWindow(from, to string, days int) (start, end time.Time, err error) {
	if to == "" {
		end = time.Now().UTC().AddDate(0, 0, -1)
	} else if end, err = time.Parse("2006-01-02", to); err != nil {
		return start, end, errBadDate
	}
	if from == "" {
		if days <= 0 {
			days = 30
		}
		start = end.AddDate(0, 0, -days)
	} else if start, err = time.Parse("2006-01-02", from); err != nil {
		return start, end, errBadDate
	}
	return start, end, nil
}
