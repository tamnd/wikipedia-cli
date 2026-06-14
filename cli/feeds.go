package cli

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/tamnd/wikipedia-cli/wiki"
)

// parseDate parses a YYYY-MM-DD argument, defaulting to today (UTC).
func parseDate(s string) (time.Time, error) {
	if s == "" || s == "today" {
		return time.Now().UTC(), nil
	}
	return time.Parse("2006-01-02", s)
}

func newFeaturedCmd(app *App) *cobra.Command {
	var date string
	cmd := &cobra.Command{
		Use:     "featured [date]",
		Aliases: []string{"daily", "tfa"},
		Short:   "Show the featured content for a day",
		Long: `Show Wikipedia's featured content for a date (default today): today's
featured article, most-read pages, picture of the day, in the news and the
on-this-day highlights. Date is YYYY-MM-DD.

Examples:
  wiki featured
  wiki featured 2020-07-20
  wiki featured -o jsonl`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := app.Client()
			if err != nil {
				return err
			}
			if len(args) == 1 {
				date = args[0]
			}
			d, err := parseDate(date)
			if err != nil {
				return usageErr("date must be YYYY-MM-DD")
			}
			sp := app.progress("fetching featured feed")
			feed, err := c.Featured(cmd.Context(), d)
			sp.stop()
			if err != nil {
				return wrapErr(err)
			}
			return emitFeatured(app, feed)
		},
	}
	cmd.Flags().StringVar(&date, "date", "", "date as YYYY-MM-DD (default today)")
	return cmd
}

func emitFeatured(app *App, feed *wiki.FeaturedFeed) error {
	if app.Out.Format() == FormatJSON || app.Out.Format() == FormatJSONL {
		if err := app.Out.Emit(Row{Cols: []string{"value"}, Vals: []string{""}, Value: feed}); err != nil {
			return err
		}
		return app.Out.Flush()
	}
	if feed.TFA != nil {
		_, _ = fmt.Fprintf(cmdOut, "Featured article: %s\n  %s\n  %s\n\n", feed.TFA.Title, feed.TFA.Description, feed.TFA.URL)
	}
	if len(feed.MostRead) > 0 {
		_, _ = fmt.Fprintln(cmdOut, "Most read:")
		for _, a := range feed.MostRead {
			_, _ = fmt.Fprintf(cmdOut, "  %2d. %-40s %d views\n", a.Rank, a.Title, a.Views)
		}
		_, _ = fmt.Fprintln(cmdOut)
	}
	if feed.Image != nil {
		_, _ = fmt.Fprintf(cmdOut, "Picture of the day: %s\n  %s\n\n", feed.Image.Title, feed.Image.URL)
	}
	if len(feed.News) > 0 {
		_, _ = fmt.Fprintln(cmdOut, "In the news:")
		for _, n := range feed.News {
			_, _ = fmt.Fprintf(cmdOut, "  - %s\n", n.Story)
		}
		_, _ = fmt.Fprintln(cmdOut)
	}
	if len(feed.OnThisDay) > 0 {
		_, _ = fmt.Fprintln(cmdOut, "On this day:")
		for _, o := range feed.OnThisDay {
			_, _ = fmt.Fprintf(cmdOut, "  %d: %s\n", o.Year, o.Text)
		}
	}
	return nil
}

func newOnThisDayCmd(app *App) *cobra.Command {
	var eventType, date string
	cmd := &cobra.Command{
		Use:     "onthisday [date]",
		Aliases: []string{"otd"},
		Short:   "Show historical events for a day",
		Long: `Show events that happened on a given month and day across history. Date is
MM-DD or YYYY-MM-DD (the year is ignored). Pick a slice with --type:
all, selected, births, deaths, holidays, events.

Examples:
  wiki onthisday
  wiki onthisday 07-20 --type births
  wiki onthisday --type deaths -o jsonl`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := app.Client()
			if err != nil {
				return err
			}
			if len(args) == 1 {
				date = args[0]
			}
			month, day, err := parseMonthDay(date)
			if err != nil {
				return usageErr("date must be MM-DD or YYYY-MM-DD")
			}
			sp := app.progress("fetching events")
			events, err := c.OnThisDayEvents(cmd.Context(), eventType, month, day)
			sp.stop()
			if err != nil {
				return wrapErr(err)
			}
			if len(events) == 0 {
				return noResults("no events")
			}
			for _, e := range events {
				if err := app.Out.Emit(Row{
					Cols:  []string{"year", "text", "pages"},
					Vals:  []string{itoa(e.Year), e.Text, joinArgs(e.Pages)},
					Value: e,
				}); err != nil {
					return err
				}
			}
			return app.Out.Flush()
		},
	}
	cmd.Flags().StringVar(&eventType, "type", "all", "all|selected|births|deaths|holidays|events")
	cmd.Flags().StringVar(&date, "date", "", "date as MM-DD (default today)")
	return cmd
}

// parseMonthDay accepts MM-DD or YYYY-MM-DD, defaulting to today.
func parseMonthDay(s string) (int, int, error) {
	if s == "" || s == "today" {
		now := time.Now().UTC()
		return int(now.Month()), now.Day(), nil
	}
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return int(t.Month()), t.Day(), nil
	}
	t, err := time.Parse("01-02", s)
	if err != nil {
		return 0, 0, err
	}
	return int(t.Month()), t.Day(), nil
}

func newTopCmd(app *App) *cobra.Command {
	var date string
	cmd := &cobra.Command{
		Use:   "top [date]",
		Short: "List the most-viewed articles for a day or month",
		Long: `List the most-viewed articles on the wiki for a day (YYYY-MM-DD) or a whole
month (YYYY-MM). Defaults to yesterday, the most recent complete day.

Examples:
  wiki top
  wiki top 2024-01-01 -n 25
  wiki top 2024-01 -o jsonl`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := app.Client()
			if err != nil {
				return err
			}
			if len(args) == 1 {
				date = args[0]
			}
			year, month, day, err := parseTopDate(date)
			if err != nil {
				return usageErr("date must be YYYY-MM-DD or YYYY-MM")
			}
			sp := app.progress("fetching top articles")
			arts, err := c.Top(cmd.Context(), year, month, day, app.Limit)
			sp.stop()
			if err != nil {
				return wrapErr(err)
			}
			if len(arts) == 0 {
				return noResults("no data for that date")
			}
			for _, a := range arts {
				if err := app.Out.Emit(Row{
					Cols:  []string{"rank", "title", "views", "url"},
					Vals:  []string{itoa(a.Rank), a.Title, itoa(a.Views), a.URL},
					Value: a,
				}); err != nil {
					return err
				}
			}
			return app.Out.Flush()
		},
	}
	cmd.Flags().StringVar(&date, "date", "", "YYYY-MM-DD or YYYY-MM (default yesterday)")
	return cmd
}

// parseTopDate accepts YYYY-MM-DD (day) or YYYY-MM (whole month), defaulting to
// yesterday since today's data is incomplete.
func parseTopDate(s string) (year, month, day int, err error) {
	if s == "" {
		y := time.Now().UTC().AddDate(0, 0, -1)
		return y.Year(), int(y.Month()), y.Day(), nil
	}
	if t, e := time.Parse("2006-01-02", s); e == nil {
		return t.Year(), int(t.Month()), t.Day(), nil
	}
	t, e := time.Parse("2006-01", s)
	if e != nil {
		return 0, 0, 0, e
	}
	return t.Year(), int(t.Month()), 0, nil
}
