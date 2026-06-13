package cli

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tamnd/wikipedia-cli/wiki"
)

func newDumpCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dump",
		Short: "Work with the public Wikimedia XML dumps",
		Long: `List, download and stream-parse the public Wikimedia XML dumps.

  wiki dump list                 available files for the latest dump
  wiki dump download <file>      download with resume + sha1 verify
  wiki dump pages <file.xml>     stream a pages-articles dump to records
  wiki dump grep <pattern> <f>   stream pages whose title/text matches`,
	}
	cmd.AddCommand(newDumpListCmd(app), newDumpDownloadCmd(app), newDumpPagesCmd(app), newDumpGrepCmd(app))
	return cmd
}

func newDumpListCmd(app *App) *cobra.Command {
	var dumpWiki, date string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List the files of a dump",
		Long: `List the per-job files of a dump (name, size, sha1, url). Defaults to the
selected wiki's latest dated dump.

Examples:
  wiki dump list
  wiki dump list --wiki enwiki --date latest
  wiki dump list --wiki dewiki -o jsonl`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := app.Client()
			if err != nil {
				return err
			}
			files, resolved, err := c.DumpList(cmd.Context(), dumpWiki, date)
			if err != nil {
				return wrapErr(err)
			}
			if len(files) == 0 {
				return noResults("no dump files for " + resolved)
			}
			for _, f := range files {
				if err := app.Out.Emit(Row{
					Cols:  []string{"job", "name", "size", "sha1", "url"},
					Vals:  []string{f.Job, f.Name, itoa64(f.Size), f.Sha1, f.URL},
					Value: f,
				}); err != nil {
					return err
				}
			}
			return app.Out.Flush()
		},
	}
	cmd.Flags().StringVar(&dumpWiki, "wiki", "", "dump database name (default from --project/-l)")
	cmd.Flags().StringVar(&date, "date", "latest", "dump date YYYYMMDD or latest")
	return cmd
}

func newDumpDownloadCmd(app *App) *cobra.Command {
	var dumpWiki, date, outDir string
	cmd := &cobra.Command{
		Use:   "download <file | job>",
		Short: "Download a dump file (resumable, sha1-verified)",
		Long: `Download a dump file by its name or job. The download resumes a partial
file and verifies the sha1 from dumpstatus.json when present.

Examples:
  wiki dump download enwiki-latest-pages-articles1.xml-p1p41242.bz2
  wiki dump download metahistory7zdump --out-dir ~/dumps`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := app.Client()
			if err != nil {
				return err
			}
			files, resolved, err := c.DumpList(cmd.Context(), dumpWiki, date)
			if err != nil {
				return wrapErr(err)
			}
			match := pickDumpFile(files, args[0])
			if match == nil {
				return notFound(fmt.Sprintf("no dump file matching %q in %s", args[0], resolved))
			}
			fmt.Fprintf(os.Stderr, "downloading %s (%s)\n", match.Name, humanSize(match.Size))
			path, err := c.DownloadFile(cmd.Context(), match.URL, outDir, downloadProgress(match.Size))
			if err != nil {
				return wrapErr(err)
			}
			fmt.Fprintln(os.Stderr)
			if match.Sha1 != "" {
				ok, err := wiki.VerifySha1(path, match.Sha1)
				if err != nil {
					return err
				}
				if !ok {
					return fmt.Errorf("sha1 mismatch for %s", path)
				}
				fmt.Fprintf(os.Stderr, "sha1 ok\n")
			}
			fmt.Fprintln(cmdOut, path)
			return nil
		},
	}
	cmd.Flags().StringVar(&dumpWiki, "wiki", "", "dump database name (default from --project/-l)")
	cmd.Flags().StringVar(&date, "date", "latest", "dump date YYYYMMDD or latest")
	cmd.Flags().StringVar(&outDir, "out-dir", "", "output directory (default data-dir/dumps)")
	return cmd
}

// pickDumpFile finds a file by exact name, then by job name, then by substring.
func pickDumpFile(files []wiki.DumpFile, q string) *wiki.DumpFile {
	for i := range files {
		if files[i].Name == q {
			return &files[i]
		}
	}
	for i := range files {
		if files[i].Job == q {
			return &files[i]
		}
	}
	for i := range files {
		if strings.Contains(files[i].Name, q) {
			return &files[i]
		}
	}
	return nil
}

func newDumpPagesCmd(app *App) *cobra.Command {
	var namespace int
	var withText bool
	cmd := &cobra.Command{
		Use:   "pages <file.xml[.bz2|.gz]>",
		Short: "Stream-parse a pages-articles dump into records",
		Long: `Stream a local pages-articles XML dump (optionally bz2/gz) into records in
constant memory. Honors -n/--limit and --namespace; --text includes the body.

Examples:
  wiki dump pages enwiki-latest-pages-articles1.xml.bz2 --namespace 0 -n 100 -o jsonl
  wiki dump pages simplewiki-latest-pages-articles.xml.bz2 --text -n 1 -o json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			n := 0
			err := wiki.StreamPages(args[0], namespace, withText, func(p wiki.DumpPage) error {
				if err := app.Out.Emit(dumpPageRow(p, withText)); err != nil {
					return err
				}
				n++
				if app.Limit > 0 && n >= app.Limit {
					return errStopStream
				}
				return nil
			})
			if err != nil && err != errStopStream {
				return wrapErr(err)
			}
			if n == 0 {
				return noResults("no pages matched")
			}
			return app.Out.Flush()
		},
	}
	cmd.Flags().IntVarP(&namespace, "namespace", "N", -1, "filter to a namespace id")
	cmd.Flags().BoolVar(&withText, "text", false, "include the page body")
	return cmd
}

func newDumpGrepCmd(app *App) *cobra.Command {
	var namespace int
	var titleOnly, withText bool
	cmd := &cobra.Command{
		Use:   "grep <pattern> <file.xml[.bz2|.gz]>",
		Short: "Stream a dump and emit pages matching a regexp",
		Long: `Stream a local dump and emit pages whose title or text matches a regular
expression. --title-only restricts the match to titles.

Examples:
  wiki dump grep '(?i)quantum' simplewiki-latest-pages-articles.xml.bz2 -n 20
  wiki dump grep '^List of' enwiki-...-pages-articles1.xml.bz2 --title-only`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			re, err := regexp.Compile(args[0])
			if err != nil {
				return usageErr("bad pattern: " + err.Error())
			}
			n := 0
			err = wiki.StreamPages(args[1], namespace, true, func(p wiki.DumpPage) error {
				if !re.MatchString(p.Title) && (titleOnly || !re.MatchString(p.Text)) {
					return nil
				}
				if err := app.Out.Emit(dumpPageRow(p, withText)); err != nil {
					return err
				}
				n++
				if app.Limit > 0 && n >= app.Limit {
					return errStopStream
				}
				return nil
			})
			if err != nil && err != errStopStream {
				return wrapErr(err)
			}
			if n == 0 {
				return noResults("no pages matched")
			}
			return app.Out.Flush()
		},
	}
	cmd.Flags().IntVarP(&namespace, "namespace", "N", -1, "filter to a namespace id")
	cmd.Flags().BoolVar(&titleOnly, "title-only", false, "match titles only, not body")
	cmd.Flags().BoolVar(&withText, "text", false, "include the page body in output")
	return cmd
}

func dumpPageRow(p wiki.DumpPage, withText bool) Row {
	cols := []string{"id", "ns", "title", "revid", "timestamp"}
	vals := []string{itoa(p.ID), itoa(p.NS), p.Title, itoa(p.RevID), p.Timestamp}
	if withText {
		cols = append(cols, "text")
		vals = append(vals, p.Text)
	}
	return Row{Cols: cols, Vals: vals, Value: p}
}
