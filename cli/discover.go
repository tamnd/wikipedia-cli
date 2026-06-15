package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tamnd/wikipedia-cli/wiki"
)

// defaultDiscoverBudget caps a streaming walk when -n is not given, so
// `wiki discover <page>` always terminates instead of spidering the wiki forever.
const defaultDiscoverBudget = 500

// newDiscoverCmd is the breadth-first graph walk. Where each structure command
// answers one question about one page (its links, its categories, a category's
// members), discover chains them: from a seed page or category it follows the
// object's links and from each neighbor it follows theirs, hop by hop, streaming
// one row per node as it is reached.
func newDiscoverCmd(app *App) *cobra.Command {
	var (
		depth  int
		fanout int
		follow string
	)
	cmd := &cobra.Command{
		Use:     "discover <seed>...",
		Aliases: []string{"walk", "graph"},
		Short:   "Breadth-first walk of the graph linked from a page or category",
		Long: `Walk the graph of linked Wikipedia objects, breadth first, starting from one
or more seeds. A seed is anything wiki can resolve: an article title or URL, or
a category name or URL.

--follow chooses which links to traverse. It takes a preset or a comma-separated
edge list:

  content   a page's links and categories, a category's members and
            subcategories (the default; the obvious neighbors)
  network   a page's outgoing links and its backlinks
  cats      a page's categories, and a category's members and subcategories
  all       every edge

Edges: links, backlinks, categories, members, subcats. Name an edge directly to
follow just that link, e.g. --follow backlinks to walk what-links-here.

--depth is how many hops to follow (default 1; 0 emits only the seeds). --fanout
caps neighbors per edge (default 25). The walk streams nodes and stops after -n
nodes (default 500).

wiki keeps no local database, so discover streams to stdout. To keep a walk,
pipe it: wiki discover "Alan Turing" --depth 2 -o jsonl > turing-graph.jsonl.`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			edges, err := wiki.ParseEdges(follow)
			if err != nil {
				return usageErr(err.Error())
			}
			seeds, err := parseSeeds(args)
			if err != nil {
				return err
			}
			c, err := app.Client()
			if err != nil {
				return err
			}

			budget := app.Limit
			if budget <= 0 {
				budget = defaultDiscoverBudget
			}

			opts := wiki.WalkOptions{
				Depth:  depth,
				Max:    budget,
				Fanout: fanout,
				Edges:  edges,
				Note: func(s string) {
					if !app.quiet {
						fmt.Fprintf(os.Stderr, "wiki: note: %s\n", s)
					}
				},
			}

			sp := app.progress("walking")
			n := 0
			walkErr := c.Walk(cmd.Context(), seeds, opts, func(nd *wiki.Node) error {
				sp.stop() // clear the spinner before the first row reaches stdout
				n++
				return app.Out.Emit(nodeRow(nd))
			})
			sp.stop()
			flushErr := app.Out.Flush()
			if walkErr != nil {
				// The only fatal walk error is a seed that could not be fetched;
				// it surfaces like a failed single read. Deeper failures are notes.
				return wrapErr(walkErr)
			}
			if flushErr != nil {
				return flushErr
			}
			if n == 0 {
				return noResults("nothing discovered")
			}
			return nil
		},
	}
	cmd.Flags().IntVar(&depth, "depth", 1, "hops to follow from each seed (0 = seeds only)")
	cmd.Flags().IntVar(&fanout, "fanout", 25, "max neighbors to follow per edge (0 = unlimited)")
	cmd.Flags().StringVar(&follow, "follow", "content", "edges to follow ("+wiki.EdgeHelp()+")")
	return cmd
}

// parseSeeds turns the positional arguments into walk seeds, reporting an
// unrecognized reference as a usage error rather than a plain failure.
func parseSeeds(args []string) ([]wiki.Seed, error) {
	seeds := make([]wiki.Seed, 0, len(args))
	for _, a := range args {
		s, err := wiki.ParseSeed(a)
		if err != nil {
			return nil, usageErr(err.Error())
		}
		seeds = append(seeds, s)
	}
	return seeds, nil
}

// nodeRow renders a graph node discovered by `wiki discover`. The curated columns
// read the walk at a glance (how deep, by which edge, the title and a one-line
// gloss) while the full typed node rides in Value for json, jsonl, and templates.
func nodeRow(n *wiki.Node) Row {
	var title, summary, url string
	switch n.Kind {
	case wiki.KindPage:
		p := n.Page
		title = p.Title
		summary = oneline(firstNonEmpty(p.Description, p.Extract))
		url = p.URL
	case wiki.KindCategory:
		c := n.Category
		title = c.Title
		url = c.URL
	}
	return Row{
		Cols:  []string{"depth", "via", "kind", "title", "summary", "url"},
		Vals:  []string{itoa(n.Depth), string(n.Via), string(n.Kind), title, summary, url},
		Value: n,
	}
}

// oneline flattens a value to a single short line for a curated column, so a
// multi-line extract never breaks the list and table layouts.
func oneline(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	if len(s) > 80 {
		return s[:79] + "..."
	}
	return s
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
