package cli

import (
	"os"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/tamnd/wikipedia-cli/wiki/parse"
)

func newParseCmd(app *App) *cobra.Command {
	var tree, templates bool
	cmd := &cobra.Command{
		Use:   "parse <title | file | ->",
		Short: "Parse the wikitext of an article into a tree with byte spans",
		Long: `Parse the wikitext of an article and show what the parser found.

The input is a file whose name is the argument, stdin for "-", or else the
title of an article, whose wikitext is fetched. Each node of the tree has the
byte span of its source, so a tool can go back to the exact text.

With no flag, the command prints one row with the counts of sections, headings,
templates, links, tags and math elements. --tree prints the whole tree.
--templates prints one row for each template call, with its name, its key in
the template table, its span and the number of its parameters.

Examples:
  wiki parse "Sylow theorems"
  wiki parse "Sylow theorems" --templates -o tsv
  wiki parse page.wiki --tree
  wiki get "Group (mathematics)" --wikitext | wiki parse - --tree`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			src, name, err := app.readWikitext(cmd, args[0])
			if err != nil {
				return err
			}
			doc, err := parse.Parse(src)
			if err != nil {
				return wrapErr(err)
			}
			switch {
			case tree:
				_, err := os.Stdout.WriteString(doc.Dump())
				return err
			case templates:
				return emitTemplates(app, doc)
			}
			c := countNodes(doc)
			c.Page, c.Bytes = name, len(src)
			if err := app.Out.Emit(Row{
				Cols: []string{"page", "bytes", "sections", "templates", "links", "tags", "math"},
				Vals: []string{c.Page, strconv.Itoa(c.Bytes), strconv.Itoa(c.Sections), strconv.Itoa(c.Templates),
					strconv.Itoa(c.Links), strconv.Itoa(c.Tags), strconv.Itoa(c.Math)},
				Value: c,
			}); err != nil {
				return err
			}
			return app.Out.Flush()
		},
	}
	cmd.Flags().BoolVar(&tree, "tree", false, "print the whole tree with the byte spans")
	cmd.Flags().BoolVar(&templates, "templates", false, "print one row for each template call")
	return cmd
}

// readWikitext reads the wikitext: stdin for "-", a file when the argument
// names one, or the wikitext of the article with that title.
func (a *App) readWikitext(cmd *cobra.Command, arg string) (string, string, error) {
	if arg == "-" {
		b, err := readAllStdin()
		return string(b), "-", err
	}
	if b, err := os.ReadFile(arg); err == nil {
		return string(b), arg, nil
	}
	c, title, err := a.resolveTarget(arg)
	if err != nil {
		return "", "", err
	}
	sp := a.progress("fetching wikitext")
	src, err := c.Wikitext(cmd.Context(), title)
	sp.stop()
	return src, title, wrapErr(err)
}

// parseCounts is the summary of one parse.
type parseCounts struct {
	Page      string `json:"page"`
	Bytes     int    `json:"bytes"`
	Sections  int    `json:"sections"`
	Templates int    `json:"templates"`
	Links     int    `json:"links"`
	Tags      int    `json:"tags"`
	Math      int    `json:"math"`
}

func countNodes(doc *parse.Document) parseCounts {
	var c parseCounts
	parse.Walk(doc.Root, func(n *parse.Node) bool {
		switch n.Kind {
		case parse.KindSection:
			c.Sections++
		case parse.KindTemplate:
			c.Templates++
		case parse.KindLink:
			c.Links++
		case parse.KindTag:
			c.Tags++
			if n.Name == "math" {
				c.Math++
			}
		}
		return true
	})
	return c
}

// templateRow is the record of one template call.
type templateRow struct {
	Name   string `json:"name"`
	Key    string `json:"key"`
	Start  int    `json:"start"`
	End    int    `json:"end"`
	Params int    `json:"params"`
}

func emitTemplates(app *App, doc *parse.Document) error {
	list := parse.Find(doc.Root, parse.KindTemplate)
	if len(list) == 0 {
		return noResults("no templates")
	}
	if app.Limit > 0 && len(list) > app.Limit {
		list = list[:app.Limit]
	}
	for _, n := range list {
		r := templateRow{n.Name, parse.TemplateKey(n.Name), n.Start, n.End, len(n.Params)}
		if err := app.Out.Emit(Row{
			Cols:  []string{"name", "key", "start", "end", "params"},
			Vals:  []string{r.Name, r.Key, strconv.Itoa(r.Start), strconv.Itoa(r.End), strconv.Itoa(r.Params)},
			Value: r,
		}); err != nil {
			return err
		}
	}
	return app.Out.Flush()
}
