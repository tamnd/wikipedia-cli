package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tamnd/wikipedia-cli/wiki"
)

// contentFlags are the render options shared by read and get.
type contentFlags struct {
	text     bool
	markdown bool
	html     bool
	wikitext bool
	summary  bool
	lead     bool
	section  int
	rev      int
}

// explicitForm reports whether the user asked for a specific render form, so
// get can honour it instead of falling back to the JSON summary when piped.
func (f *contentFlags) explicitForm() bool {
	return f.text || f.markdown || f.html || f.wikitext || f.lead ||
		f.section >= 0 || f.rev > 0
}

func (f *contentFlags) bind(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&f.text, "text", false, "render as plain text (default)")
	cmd.Flags().BoolVarP(&f.markdown, "markdown", "m", false, "render as Markdown")
	cmd.Flags().BoolVar(&f.html, "html", false, "the server-rendered HTML")
	cmd.Flags().BoolVar(&f.wikitext, "wikitext", false, "the raw wikitext source")
	cmd.Flags().BoolVar(&f.summary, "summary", false, "a one-paragraph summary")
	cmd.Flags().BoolVar(&f.lead, "lead", false, "the lead section only (text)")
	cmd.Flags().IntVar(&f.section, "section", -1, "render only this section index")
	cmd.Flags().IntVar(&f.rev, "rev", 0, "a specific revision id")
}

// render fetches and renders the requested form of a page into a string.
func (f *contentFlags) render(cmd *cobra.Command, c *wiki.Client, title string) (string, error) {
	ctx := cmd.Context()
	switch {
	case f.rev > 0:
		src, err := c.RevisionContent(ctx, f.rev)
		if err != nil {
			return "", wrapErr(err)
		}
		if f.markdown {
			return wiki.HTMLToMarkdown("<pre>" + src + "</pre>"), nil
		}
		return src, nil
	case f.summary:
		s, err := c.GetSummary(ctx, title)
		if err != nil {
			return "", wrapErr(err)
		}
		return s.Extract, nil
	case f.wikitext:
		return wrapText(c.Wikitext(ctx, title))
	case f.html:
		b, err := c.HTML(ctx, title)
		return string(b), wrapErr(err)
	case f.section >= 0:
		h, err := c.SectionHTML(ctx, title, f.section)
		if err != nil {
			return "", wrapErr(err)
		}
		if f.markdown {
			return wiki.HTMLToMarkdown(h), nil
		}
		return wiki.HTMLToText(h), nil
	case f.markdown:
		b, err := c.HTML(ctx, title)
		if err != nil {
			return "", wrapErr(err)
		}
		return wiki.HTMLToMarkdown(string(b)), nil
	default: // text (and --lead)
		text, err := c.Extract(ctx, title, f.lead)
		return text, wrapErr(err)
	}
}

func wrapText(s string, err error) (string, error) { return s, wrapErr(err) }

func newReadCmd(app *App) *cobra.Command {
	f := &contentFlags{}
	cmd := &cobra.Command{
		Use:     "read <title>",
		Aliases: []string{"show", "cat"},
		Short:   "Read an article (paged plain text by default)",
		Long: `Fetch and render an article for reading.

By default the article is shown as clean plain text in your pager. Switch the
form with --markdown, --html, --wikitext, or --summary. A pasted Wikipedia URL
works as the argument and selects the right wiki automatically.

Examples:
  wiki read "Alan Turing"
  wiki read Go_(programming_language) --markdown
  wiki read "Berlin" -l de
  wiki read https://en.wikipedia.org/wiki/Cat`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, title, err := app.resolveTarget(args[0])
			if err != nil {
				return err
			}
			sp := app.progress("fetching article")
			body, err := f.render(cmd, c, title)
			sp.stop()
			if err != nil {
				return err
			}
			return app.writeContent(body)
		},
	}
	f.bind(cmd)
	cmd.Flags().BoolVar(&app.noPager, "no-pager", false, "never page output")
	return cmd
}

func newGetCmd(app *App) *cobra.Command {
	f := &contentFlags{}
	cmd := &cobra.Command{
		Use:   "get <title>",
		Short: "Fetch an article for pipelines (curl for Wikipedia)",
		Long: `Fetch an article and write it to stdout without paging.

This is the scriptable sibling of read: default render is plain text, output is
never paged, and -o json returns the structured summary. Reads a title on stdin
with '-' so search results pipe straight in.

Examples:
  wiki get "Alan Turing" --text | wc -w
  wiki get "Pi" --html > pi.html
  wiki search "turing" -o jsonl | wiki get - --summary`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app.noPager = true
			titles, err := argsOrStdin(args)
			if err != nil {
				return err
			}
			if len(titles) == 0 {
				return usageErr("specify a title (or pipe titles on stdin with -)")
			}
			for _, raw := range titles {
				c, title, err := app.resolveTarget(raw)
				if err != nil {
					return err
				}
				jsonOut := app.Out.Format() == FormatJSON || app.Out.Format() == FormatJSONL
				if jsonOut && !f.explicitForm() {
					sp := app.progress("fetching summary")
					s, err := c.GetSummary(cmd.Context(), title)
					sp.stop()
					if err != nil {
						return wrapErr(err)
					}
					if err := app.Out.Emit(summaryRow(s)); err != nil {
						return err
					}
					continue
				}
				sp := app.progress("fetching article")
				body, err := f.render(cmd, c, title)
				sp.stop()
				if err != nil {
					return err
				}
				if _, err := fmt.Fprintln(cmdOut, body); err != nil {
					return err
				}
			}
			return app.Out.Flush()
		},
	}
	f.bind(cmd)
	return cmd
}

func newSummaryCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "summary <title>",
		Aliases: []string{"tldr"},
		Short:   "Print a one-paragraph summary of an article",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, title, err := app.resolveTarget(args[0])
			if err != nil {
				return err
			}
			sp := app.progress("fetching summary")
			s, err := c.GetSummary(cmd.Context(), title)
			sp.stop()
			if err != nil {
				return wrapErr(err)
			}
			f := app.Out.Format()
			if f != FormatList && f != FormatTable && f != FormatURL {
				if err := app.Out.Emit(summaryRow(s)); err != nil {
					return err
				}
				return app.Out.Flush()
			}
			_, err = fmt.Fprintln(cmdOut, s.Extract)
			return err
		},
	}
	return cmd
}

func newOpenCmd(app *App) *cobra.Command {
	var printOnly bool
	cmd := &cobra.Command{
		Use:   "open <title>",
		Short: "Open an article in the browser (or print its URL)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, title, err := app.resolveTarget(args[0])
			if err != nil {
				return err
			}
			u := c.Site.PageURL(title)
			if printOnly {
				_, err := fmt.Fprintln(cmdOut, u)
				return err
			}
			return openBrowser(u)
		},
	}
	cmd.Flags().BoolVar(&printOnly, "print", false, "print the URL instead of opening it")
	return cmd
}

func summaryRow(s *wiki.Summary) Row {
	return Row{
		Cols:  []string{"title", "description", "extract", "type", "lang", "url"},
		Vals:  []string{s.Title, s.Description, s.Extract, s.Type, s.Lang, s.URL},
		Value: s,
	}
}
