package cli

import (
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tamnd/wikipedia-cli/wiki"
)

func newEntityCmd(app *App) *cobra.Command {
	var lang, props string
	var byTitle bool
	cmd := &cobra.Command{
		Use:     "entity <Q-id | title>",
		Aliases: []string{"wd"},
		Short:   "Show a Wikidata entity",
		Long: `Show a Wikidata entity: its label, description, aliases and claims.

Pass a Q-id (or P-id) directly, or a Wikipedia article title with --title to
resolve it via the current wiki. Restrict claims with --props P31,P569.

Examples:
  wiki entity Q937
  wiki entity "Albert Einstein" --title --props P31,P569,P570
  wiki entity Q64 --lang de`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := app.Client()
			if err != nil {
				return err
			}
			var propList []string
			if props != "" {
				propList = splitComma(props)
			}
			arg := args[0]
			var ent *wiki.Entity
			sp := app.progress("fetching entity")
			if byTitle || !looksLikeEntityID(arg) {
				ent, err = c.EntityByTitle(cmd.Context(), arg, lang, propList)
			} else {
				ent, err = c.EntityByID(cmd.Context(), strings.ToUpper(arg), lang, propList)
			}
			sp.stop()
			if err != nil {
				return wrapErr(err)
			}
			displayLang := lang
			if displayLang == "" {
				displayLang = c.Cfg.Lang
			}
			if displayLang == "" {
				displayLang = "en"
			}
			return emitEntity(app, ent, displayLang)
		},
	}
	cmd.Flags().StringVar(&lang, "lang", "", "language for the flattened table view (JSON keeps all languages)")
	cmd.Flags().StringVar(&props, "props", "", "comma-separated property ids to include")
	cmd.Flags().BoolVar(&byTitle, "title", false, "treat the argument as an article title")
	return cmd
}

// looksLikeEntityID reports whether s is a bare Wikidata id like Q42 or P31.
func looksLikeEntityID(s string) bool {
	if len(s) < 2 {
		return false
	}
	switch s[0] {
	case 'Q', 'P', 'q', 'p':
	default:
		return false
	}
	for _, r := range s[1:] {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func emitEntity(app *App, e *wiki.Entity, lang string) error {
	rows := [][2]string{
		{"id", e.ID},
		{"label", e.LabelFor(lang)},
		{"description", e.DescriptionFor(lang)},
		{"aliases", strings.Join(e.AliasesFor(lang), ", ")},
	}
	pids := make([]string, 0, len(e.Claims))
	for p := range e.Claims {
		pids = append(pids, p)
	}
	sort.Strings(pids)
	for _, p := range pids {
		vals := make([]string, 0, len(e.Claims[p]))
		for _, st := range e.Claims[p] {
			vals = append(vals, st.Mainsnak.ValueString())
		}
		rows = append(rows, [2]string{p, strings.Join(vals, ", ")})
	}
	return emitKV(app, rows, e)
}

func newSPARQLCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sparql <query | @file.rq | ->",
		Short: "Run a SPARQL query against the Wikidata Query Service",
		Long: `Run a SPARQL query against the Wikidata Query Service and print the result
rows. The query may be given inline, read from a file with @path, or from
stdin with '-'. Entity URIs are shortened to bare Q/P ids in the table view;
-o json keeps each binding's full value, term type, language and datatype.

Examples:
  wiki sparql 'SELECT ?c ?p WHERE { ?c wdt:P31 wd:Q515; wdt:P1082 ?p } ORDER BY DESC(?p) LIMIT 10'
  wiki sparql @capitals.rq -o csv`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := app.Client()
			if err != nil {
				return err
			}
			query, err := readQuery(args[0])
			if err != nil {
				return usageErr(err.Error())
			}
			sp := app.progress("running query")
			res, err := c.SPARQL(cmd.Context(), query)
			sp.stop()
			if err != nil {
				return wrapErr(err)
			}
			if len(res.Rows) == 0 {
				return noResults("no results")
			}
			for _, row := range res.Rows {
				vals := make([]string, len(res.Vars))
				for i, v := range res.Vars {
					vals[i] = wiki.ShortenURI(row[v].Value)
				}
				if err := app.Out.Emit(Row{Cols: res.Vars, Vals: vals, Value: row}); err != nil {
					return err
				}
			}
			return app.Out.Flush()
		},
	}
	return cmd
}

// readQuery resolves a SPARQL query argument: inline text, @file, or - (stdin).
func readQuery(arg string) (string, error) {
	switch {
	case arg == "-":
		b, err := readAllStdin()
		return string(b), err
	case strings.HasPrefix(arg, "@"):
		b, err := os.ReadFile(strings.TrimPrefix(arg, "@"))
		return string(b), err
	default:
		return arg, nil
	}
}
