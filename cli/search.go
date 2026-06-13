package cli

import (
	"github.com/spf13/cobra"
	"github.com/tamnd/wikipedia-cli/wiki"
)

func searchRow(s wiki.SearchResult) Row {
	return Row{
		Cols:  []string{"title", "description", "snippet", "size", "wordcount", "timestamp", "url"},
		Vals:  []string{s.Title, s.Description, s.Snippet, itoa(s.Size), itoa(s.WordCount), s.Timestamp, s.URL},
		Value: s,
	}
}

func newSearchCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "search <query>",
		Aliases: []string{"s", "find"},
		Short:   "Full-text search the wiki",
		Long: `Search the wiki's full text and print the matching titles with a snippet.

CirrusSearch operators pass straight through (incategory:, intitle:, insource:).
Pipe the results into get with -o jsonl, or get just URLs with -o url.

Examples:
  wiki search "turing machine"
  wiki search "climate change" -n 50 -o jsonl | wiki get - --summary
  wiki search "incategory:Physics quantum" -o url`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := app.Client()
			if err != nil {
				return err
			}
			query := joinArgs(args)
			results, err := c.Search(cmd.Context(), query, app.Limit)
			if err != nil {
				return err
			}
			if len(results) == 0 {
				return noResults("no matches for " + query)
			}
			for _, r := range results {
				if err := app.Out.Emit(searchRow(r)); err != nil {
					return err
				}
			}
			return app.Out.Flush()
		},
	}
	return cmd
}

func newSuggestCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "suggest <prefix>",
		Aliases: []string{"complete", "opensearch"},
		Short:   "Autocomplete titles for a prefix",
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := app.Client()
			if err != nil {
				return err
			}
			limit := app.Limit
			if limit == 0 {
				limit = 10
			}
			results, err := c.Suggest(cmd.Context(), joinArgs(args), limit)
			if err != nil {
				return err
			}
			if len(results) == 0 {
				return noResults("no suggestions")
			}
			for _, s := range results {
				if err := app.Out.Emit(Row{
					Cols:  []string{"title", "description", "url"},
					Vals:  []string{s.Title, s.Description, s.URL},
					Value: s,
				}); err != nil {
					return err
				}
			}
			return app.Out.Flush()
		},
	}
	return cmd
}

func newRandomCmd(app *App) *cobra.Command {
	var namespace int
	cmd := &cobra.Command{
		Use:     "random",
		Aliases: []string{"rand"},
		Short:   "Show one or more random articles",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := app.Client()
			if err != nil {
				return err
			}
			n := app.Limit
			if n == 0 {
				n = 1
			}
			results, err := c.Random(cmd.Context(), n, namespace)
			if err != nil {
				return err
			}
			for _, r := range results {
				if err := app.Out.Emit(searchRow(r)); err != nil {
					return err
				}
			}
			return app.Out.Flush()
		},
	}
	cmd.Flags().IntVarP(&namespace, "namespace", "N", 0, "namespace id (0 = articles)")
	return cmd
}

func newRelatedCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "related <title>",
		Short: "Show pages related to a title",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, title, err := app.resolveTarget(args[0])
			if err != nil {
				return err
			}
			results, err := c.Related(cmd.Context(), title)
			if err != nil {
				return wrapErr(err)
			}
			if len(results) == 0 {
				return noResults("no related pages")
			}
			for _, r := range results {
				if err := app.Out.Emit(searchRow(r)); err != nil {
					return err
				}
			}
			return app.Out.Flush()
		},
	}
	return cmd
}
