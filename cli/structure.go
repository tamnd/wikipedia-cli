package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/tamnd/wikipedia-cli/wiki"
)

func newLinksCmd(app *App) *cobra.Command {
	var namespace int
	var external bool
	cmd := &cobra.Command{
		Use:   "links <title>",
		Short: "List the links on a page",
		Long: `List the internal wiki links on a page, or, with --external, the external
URLs it references.

Examples:
  wiki links "Alan Turing" -o url | head
  wiki links "Alan Turing" --external -o url`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, title, err := app.resolveTarget(args[0])
			if err != nil {
				return err
			}
			var links []wiki.Link
			if external {
				links, err = c.ExternalLinks(cmd.Context(), title, app.Limit)
			} else {
				links, err = c.Links(cmd.Context(), title, namespace, app.Limit)
			}
			if err != nil {
				return wrapErr(err)
			}
			return emitLinks(app, links, "no links")
		},
	}
	cmd.Flags().IntVarP(&namespace, "namespace", "N", -1, "filter to a namespace id")
	cmd.Flags().BoolVar(&external, "external", false, "list external URLs instead")
	return cmd
}

func newBacklinksCmd(app *App) *cobra.Command {
	var namespace int
	cmd := &cobra.Command{
		Use:     "backlinks <title>",
		Aliases: []string{"whatlinkshere"},
		Short:   "List pages that link to a title",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, title, err := app.resolveTarget(args[0])
			if err != nil {
				return err
			}
			links, err := c.Backlinks(cmd.Context(), title, namespace, app.Limit)
			if err != nil {
				return wrapErr(err)
			}
			return emitLinks(app, links, "nothing links here")
		},
	}
	cmd.Flags().IntVarP(&namespace, "namespace", "N", -1, "filter to a namespace id")
	return cmd
}

func emitLinks(app *App, links []wiki.Link, empty string) error {
	if len(links) == 0 {
		return noResults(empty)
	}
	for _, l := range links {
		url := l.URL
		row := Row{Cols: []string{"title", "ns", "url"}, Vals: []string{l.Title, itoa(l.NS), url}, Value: l}
		if l.Title == "" { // external link
			row.Cols = []string{"url"}
			row.Vals = []string{url}
		}
		if err := app.Out.Emit(row); err != nil {
			return err
		}
	}
	return app.Out.Flush()
}

func newCategoriesCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "categories <title>",
		Aliases: []string{"cats"},
		Short:   "List the categories a page belongs to",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, title, err := app.resolveTarget(args[0])
			if err != nil {
				return err
			}
			cats, err := c.Categories(cmd.Context(), title, app.Limit)
			if err != nil {
				return wrapErr(err)
			}
			if len(cats) == 0 {
				return noResults("no categories")
			}
			for _, cat := range cats {
				if err := app.Out.Emit(Row{
					Cols:  []string{"title", "ns", "url"},
					Vals:  []string{cat.Title, itoa(cat.NS), cat.URL},
					Value: cat,
				}); err != nil {
					return err
				}
			}
			return app.Out.Flush()
		},
	}
	return cmd
}

func newCategoryCmd(app *App) *cobra.Command {
	var memberType string
	cmd := &cobra.Command{
		Use:   "category <name>",
		Short: "List the members of a category",
		Long: `List the pages, subcategories, or files in a category. The "Category:"
prefix is added if you omit it.

Examples:
  wiki category "British computer scientists" -n 100 -o url
  wiki category Physics --type subcat`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := app.Client()
			if err != nil {
				return err
			}
			members, err := c.CategoryMembers(cmd.Context(), args[0], memberType, app.Limit)
			if err != nil {
				return wrapErr(err)
			}
			if len(members) == 0 {
				return noResults("no members")
			}
			for _, m := range members {
				if err := app.Out.Emit(Row{
					Cols:  []string{"title", "ns", "type", "timestamp", "url"},
					Vals:  []string{m.Title, itoa(m.NS), m.Type, m.Timestamp, m.URL},
					Value: m,
				}); err != nil {
					return err
				}
			}
			return app.Out.Flush()
		},
	}
	cmd.Flags().StringVar(&memberType, "type", "", "member type: page|subcat|file")
	return cmd
}

func newMediaCmd(app *App) *cobra.Command {
	var download bool
	var outDir string
	cmd := &cobra.Command{
		Use:     "media <title>",
		Aliases: []string{"images", "files"},
		Short:   "List the media used on a page",
		Long: `List the files used on a page with their URL, MIME, size and license. With
--download, save them to a directory.

Examples:
  wiki media "Alan Turing"
  wiki media "Alan Turing" --download --out-dir imgs/`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, title, err := app.resolveTarget(args[0])
			if err != nil {
				return err
			}
			media, err := c.Media(cmd.Context(), title, app.Limit)
			if err != nil {
				return wrapErr(err)
			}
			if len(media) == 0 {
				return noResults("no media")
			}
			if download {
				return downloadMedia(cmd, app, c, media, outDir)
			}
			for _, m := range media {
				if err := app.Out.Emit(Row{
					Cols:  []string{"title", "url", "mime", "width", "height", "size", "license", "author"},
					Vals:  []string{m.Title, m.URL, m.Mime, itoa(m.Width), itoa(m.Height), itoa(m.Size), m.License, m.Author},
					Value: m,
				}); err != nil {
					return err
				}
			}
			return app.Out.Flush()
		},
	}
	cmd.Flags().BoolVar(&download, "download", false, "download the files")
	cmd.Flags().StringVar(&outDir, "out-dir", "media", "directory for --download")
	return cmd
}

func downloadMedia(cmd *cobra.Command, app *App, c *wiki.Client, media []wiki.Media, outDir string) error {
	for _, m := range media {
		path, err := c.DownloadFile(cmd.Context(), m.URL, outDir, nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "wiki: %s: %v\n", filepath.Base(m.URL), err)
			continue
		}
		_, _ = fmt.Fprintln(cmdOut, path)
	}
	return nil
}

func newReferencesCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "references <title>",
		Aliases: []string{"refs"},
		Short:   "List the external references of a page",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, title, err := app.resolveTarget(args[0])
			if err != nil {
				return err
			}
			links, err := c.ExternalLinks(cmd.Context(), title, app.Limit)
			if err != nil {
				return wrapErr(err)
			}
			return emitLinks(app, links, "no references")
		},
	}
	return cmd
}

func newCiteCmd(app *App) *cobra.Command {
	var format string
	cmd := &cobra.Command{
		Use:   "cite <title>",
		Short: "Generate a citation for an article",
		Long: `Emit a citation for the article itself (not its sources), in the chosen
format: bibtex, ris, mla, or apa.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, title, err := app.resolveTarget(args[0])
			if err != nil {
				return err
			}
			info, err := c.Info(cmd.Context(), title)
			if err != nil {
				return wrapErr(err)
			}
			cite := formatCitation(format, info, c.Site.Host)
			_, err = fmt.Fprintln(cmdOut, cite)
			return err
		},
	}
	cmd.Flags().StringVar(&format, "format", "bibtex", "citation format: bibtex|ris|mla|apa")
	return cmd
}

func formatCitation(format string, info *wiki.PageInfo, host string) string {
	accessed := time.Now().UTC().Format("2006-01-02")
	key := strings.ReplaceAll(info.Title, " ", "")
	switch strings.ToLower(format) {
	case "ris":
		return strings.Join([]string{
			"TY  - ELEC",
			"TI  - " + info.Title,
			"PB  - Wikipedia, The Free Encyclopedia",
			"UR  - " + info.URL,
			"Y2  - " + accessed,
			"ER  - ",
		}, "\n")
	case "mla":
		return fmt.Sprintf("%q. Wikipedia, The Free Encyclopedia. Web. %s. <%s>.", info.Title, accessed, info.URL)
	case "apa":
		return fmt.Sprintf("%s. (n.d.). In Wikipedia. Retrieved %s, from %s", info.Title, accessed, info.URL)
	default: // bibtex
		return fmt.Sprintf("@misc{wiki:%s,\n  title        = {%s},\n  howpublished = {Wikipedia, The Free Encyclopedia},\n  url          = {%s},\n  note         = {Accessed: %s}\n}",
			key, info.Title, info.URL, accessed)
	}
}

func newLangsCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "langs <title>",
		Aliases: []string{"langlinks"},
		Short:   "List the same article in other languages",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, title, err := app.resolveTarget(args[0])
			if err != nil {
				return err
			}
			links, err := c.LangLinks(cmd.Context(), title, app.Limit)
			if err != nil {
				return wrapErr(err)
			}
			if len(links) == 0 {
				return noResults("no interlanguage links")
			}
			for _, l := range links {
				if err := app.Out.Emit(Row{
					Cols:  []string{"lang", "title", "autonym", "url"},
					Vals:  []string{l.Lang, l.Title, l.Autonym, l.URL},
					Value: l,
				}); err != nil {
					return err
				}
			}
			return app.Out.Flush()
		},
	}
	return cmd
}

func newInfoCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "info <title>",
		Short: "Show page metadata",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, title, err := app.resolveTarget(args[0])
			if err != nil {
				return err
			}
			info, err := c.Info(cmd.Context(), title)
			if err != nil {
				return wrapErr(err)
			}
			rows := [][2]string{
				{"pageid", itoa(info.Pageid)},
				{"ns", itoa(info.NS)},
				{"title", info.Title},
				{"length", itoa(info.Length)},
				{"touched", info.Touched},
				{"lastrevid", itoa(info.LastRevID)},
				{"contentmodel", info.ContentModel},
				{"language", info.Lang},
				{"redirect", boolWord(info.Redirect, "yes", "no")},
				{"protection", protectionSummary(info.Protection)},
				{"url", info.URL},
			}
			return emitKV(app, rows, info)
		},
	}
	return cmd
}

// emitKV renders a list of key/value pairs as rows; the whole value is attached
// to the first row for json output.
func emitKV(app *App, rows [][2]string, value any) error {
	switch app.Out.Format() {
	case FormatJSON, FormatJSONL:
		if err := app.Out.Emit(Row{Cols: []string{"value"}, Vals: []string{""}, Value: value}); err != nil {
			return err
		}
		return app.Out.Flush()
	}
	for _, r := range rows {
		if err := app.Out.Emit(Row{
			Cols:  []string{"key", "value"},
			Vals:  []string{r[0], r[1]},
			Value: map[string]string{"key": r[0], "value": r[1]},
		}); err != nil {
			return err
		}
	}
	return app.Out.Flush()
}
