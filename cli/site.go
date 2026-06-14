package cli

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tamnd/wikipedia-cli/wiki"
)

func newSitesCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "sites",
		Aliases: []string{"wikis", "projects"},
		Short:   "List the known projects and example hosts",
		Long: `List the Wikimedia projects wiki understands, with an example host for each,
so you can discover what --project, --site and -l accept.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			for _, p := range wiki.Projects() {
				cfg := app.Cfg
				cfg.Project = p
				cfg.SiteHost = ""
				site, err := cfg.Site()
				if err != nil {
					continue
				}
				kind := "language"
				if site.Lang == "" {
					kind = "global"
				}
				if err := app.Out.Emit(Row{
					Cols:  []string{"project", "kind", "host"},
					Vals:  []string{p, kind, site.Host},
					Value: map[string]string{"project": p, "kind": kind, "host": site.Host},
				}); err != nil {
					return err
				}
			}
			return app.Out.Flush()
		},
	}
	return cmd
}

func newStatsCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "stats",
		Aliases: []string{"siteinfo"},
		Short:   "Show statistics for the selected wiki",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := app.Client()
			if err != nil {
				return err
			}
			sp := app.progress("fetching statistics")
			s, err := c.Stats(cmd.Context())
			sp.stop()
			if err != nil {
				return wrapErr(err)
			}
			rows := [][2]string{
				{"sitename", s.SiteName},
				{"generator", s.Generator},
				{"language", s.Lang},
				{"mainpage", s.MainPage},
				{"pages", itoa(s.Pages)},
				{"articles", itoa(s.Articles)},
				{"edits", itoa(s.Edits)},
				{"images", itoa(s.Images)},
				{"users", itoa(s.Users)},
				{"activeusers", itoa(s.ActiveUsers)},
				{"admins", itoa(s.Admins)},
			}
			return emitKV(app, rows, s)
		},
	}
	return cmd
}

func newConvertCmd(app *App) *cobra.Command {
	var from, to string
	cmd := &cobra.Command{
		Use:   "convert <file | ->",
		Short: "Convert HTML/wikitext content to text, Markdown or JSON",
		Long: `Convert content you already have between forms, offline. Read from a file or
stdin with '-'. --from html (default) or wikitext; --to text (default),
markdown or json.

Examples:
  cat page.html | wiki convert - --to markdown
  wiki convert page.html --to text`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			src, err := readContentArg(args[0])
			if err != nil {
				return wrapErr(err)
			}
			out, err := convertContent(src, from, to, app.Cfg.Lang)
			if err != nil {
				return usageErr(err.Error())
			}
			_, err = fmt.Fprintln(cmdOut, out)
			return err
		},
	}
	cmd.Flags().StringVar(&from, "from", "html", "source form: html|wikitext")
	cmd.Flags().StringVar(&to, "to", "text", "target form: text|markdown|json")
	return cmd
}

func convertContent(src, from, to, lang string) (string, error) {
	if lang == "" {
		lang = "en"
	}
	wikitext := strings.EqualFold(from, "wikitext")
	switch strings.ToLower(to) {
	case "markdown", "md":
		if wikitext {
			return wiki.WikitextToMarkdown(src, lang), nil
		}
		return wiki.HTMLToMarkdown(src), nil
	case "json":
		var text string
		if wikitext {
			text = wiki.WikitextToText(src)
		} else {
			text = wiki.HTMLToText(src)
		}
		return toJSONString(map[string]string{"text": text}), nil
	case "text", "txt", "":
		if wikitext {
			return wiki.WikitextToText(src), nil
		}
		return wiki.HTMLToText(src), nil
	default:
		return "", fmt.Errorf("unknown --to %q (want text|markdown|json)", to)
	}
}

// readContentArg reads a file path or, for "-", all of stdin.
func readContentArg(arg string) (string, error) {
	if arg == "-" {
		b, err := io.ReadAll(os.Stdin)
		return string(b), err
	}
	b, err := os.ReadFile(arg)
	return string(b), err
}
