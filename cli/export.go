package cli

import (
	"fmt"
	"hash/fnv"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/tamnd/wikipedia-cli/wiki"
)

func newDumpExportCmd(app *App) *cobra.Command {
	var (
		outDir, format, dumpWiki, date string
		namespace, minBytes            int
		download, withRedirects        bool
	)
	cmd := &cobra.Command{
		Use:   "export [file.xml[.bz2|.gz]]",
		Short: "Convert a dump into Markdown (or text), one file per article",
		Long: `Turn a Wikimedia pages-articles dump into clean Markdown. Each article's
wikitext is parsed and converted (headings, lists, bold/italic, code blocks,
internal and external links); templates, tables, references and File/Category
chrome are dropped. Redirects and non-article namespaces are skipped.

Point it at a local dump file, or pass --download to fetch the selected wiki's
latest pages-articles dump first. With --out-dir, one file per article is
written into sharded subdirectories; otherwise the articles stream to stdout.

Examples:
  wiki dump export simplewiki-latest-pages-articles.xml.bz2 --out-dir ./md
  wiki dump export --download --wiki simplewiki --out-dir ./md
  wiki dump export dump.xml.bz2 -n 50 > sample.md
  wiki dump export dump.xml.bz2 --format text --out-dir ./txt`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ext, conv, err := exporter(format, app.Cfg.Lang)
			if err != nil {
				return usageErr(err.Error())
			}

			path := ""
			if len(args) == 1 {
				path = args[0]
			}
			if download {
				p, err := downloadPagesArticles(cmd, app, dumpWiki, date)
				if err != nil {
					return err
				}
				path = p
			}
			if path == "" {
				return usageErr("give a dump file, or --download to fetch one")
			}

			var (
				count, written int
				bytes          int64
				start          = time.Now()
			)
			err = wiki.StreamPages(path, namespace, true, func(p wiki.DumpPage) error {
				if p.Redirect || strings.TrimSpace(p.Text) == "" {
					return nil
				}
				if minBytes > 0 && len(p.Text) < minBytes {
					return nil
				}
				if !withRedirects && strings.HasPrefix(strings.ToLower(strings.TrimSpace(p.Text)), "#redirect") {
					return nil
				}
				body := conv(p.Text)
				if strings.TrimSpace(body) == "" {
					return nil
				}
				count++

				if outDir == "" {
					_, _ = fmt.Fprintf(cmdOut, "# %s\n\n%s\n\n", p.Title, body)
				} else {
					dest := shardPath(outDir, p.Title, ext)
					if err := writeArticle(dest, p.Title, body, ext); err != nil {
						return err
					}
					written++
					bytes += int64(len(body))
					if written%1000 == 0 {
						fmt.Fprintf(os.Stderr, "\r%d articles, %s", written, humanSize(bytes))
					}
				}

				if app.Limit > 0 && count >= app.Limit {
					return errStopStream
				}
				return nil
			})
			if err != nil && err != errStopStream {
				return wrapErr(err)
			}
			if count == 0 {
				return noResults("no articles to export")
			}
			if outDir != "" {
				fmt.Fprintf(os.Stderr, "\rexported %d articles (%s) to %s in %s\n",
					written, humanSize(bytes), outDir, time.Since(start).Round(time.Millisecond))
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&outDir, "out-dir", "", "write one file per article here (default: stream to stdout)")
	cmd.Flags().StringVar(&format, "format", "markdown", "output form: markdown|text")
	cmd.Flags().IntVarP(&namespace, "namespace", "N", 0, "namespace to export (0 = articles)")
	cmd.Flags().IntVar(&minBytes, "min-bytes", 0, "skip articles whose wikitext is smaller than this")
	cmd.Flags().BoolVar(&download, "download", false, "download the latest pages-articles dump first")
	cmd.Flags().StringVar(&dumpWiki, "wiki", "", "dump database name for --download (default from --project/-l)")
	cmd.Flags().StringVar(&date, "date", "latest", "dump date for --download")
	cmd.Flags().BoolVar(&withRedirects, "keep-redirects", false, "do not skip #REDIRECT pages")
	return cmd
}

// exporter returns the file extension and a wikitext converter for the format.
func exporter(format, lang string) (ext string, conv func(string) string, err error) {
	switch strings.ToLower(format) {
	case "markdown", "md", "":
		return ".md", func(s string) string { return wiki.WikitextToMarkdown(s, lang) }, nil
	case "text", "txt":
		return ".txt", wiki.WikitextToText, nil
	default:
		return "", nil, fmt.Errorf("unknown --format %q (want markdown|text)", format)
	}
}

// downloadPagesArticles resolves and downloads the wiki's combined
// pages-articles dump, returning the local path.
func downloadPagesArticles(cmd *cobra.Command, app *App, dumpWiki, date string) (string, error) {
	c, err := app.Client()
	if err != nil {
		return "", err
	}
	files, resolved, err := c.DumpList(cmd.Context(), dumpWiki, date)
	if err != nil {
		return "", wrapErr(err)
	}
	match := pickPagesArticles(files)
	if match == nil {
		return "", notFound("no pages-articles dump in " + resolved)
	}
	fmt.Fprintf(os.Stderr, "downloading %s (%s)\n", match.Name, humanSize(match.Size))
	path, err := c.DownloadFile(cmd.Context(), match.URL, "", downloadProgress(match.Size))
	if err != nil {
		return "", wrapErr(err)
	}
	fmt.Fprintln(os.Stderr)
	return path, nil
}

// pickPagesArticles selects the combined pages-articles.xml.bz2 (not the
// multistream variant, its index, or a numbered split), preferring the smallest
// such file when several match.
func pickPagesArticles(files []wiki.DumpFile) *wiki.DumpFile {
	var best *wiki.DumpFile
	for i := range files {
		name := files[i].Name
		if !strings.HasSuffix(name, "pages-articles.xml.bz2") {
			continue
		}
		if strings.Contains(name, "multistream") || strings.Contains(name, "index") {
			continue
		}
		if best == nil || files[i].Size < best.Size {
			best = &files[i]
		}
	}
	if best != nil {
		return best
	}
	for i := range files {
		if strings.Contains(files[i].Name, "pages-articles") && strings.HasSuffix(files[i].Name, ".bz2") &&
			!strings.Contains(files[i].Name, "index") {
			return &files[i]
		}
	}
	return nil
}

// shardPath returns DIR/<aa>/<safe-title><ext>, sharding by a hash of the title
// so no single directory holds millions of files.
func shardPath(dir, title, ext string) string {
	h := fnv.New32a()
	_, _ = h.Write([]byte(title))
	shard := fmt.Sprintf("%02x", h.Sum32()&0xff)
	return filepath.Join(dir, shard, safeFilename(title)+ext)
}

// safeFilename makes a title safe to use as a filename across platforms.
func safeFilename(title string) string {
	var b strings.Builder
	for _, r := range title {
		switch r {
		case '/', '\\', ':', '*', '?', '"', '<', '>', '|', 0:
			b.WriteByte('_')
		default:
			if r < 0x20 {
				b.WriteByte('_')
			} else {
				b.WriteRune(r)
			}
		}
	}
	name := strings.TrimSpace(b.String())
	if len(name) > 200 {
		name = name[:200]
	}
	if name == "" {
		name = "_"
	}
	return name
}

func writeArticle(dest, title, body, ext string) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	var content string
	if ext == ".md" {
		content = "# " + title + "\n\n" + body
	} else {
		content = title + "\n\n" + body
	}
	return os.WriteFile(dest, []byte(content), 0o644)
}
