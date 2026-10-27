package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/tamnd/wikipedia-cli/wiki"
)

func newDumpTablesCmd(app *App) *cobra.Command {
	var dumpWiki, date, outDir string
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "tables [table...]",
		Short: "Download the SQL table dumps of one date",
		Long: `Download the SQL table dumps of one dump date and check each file against
its sha1. With no table, the command downloads the tables that a base build
reads: page, redirect, page_props, categorylinks, linktarget, pagelinks,
templatelinks, langlinks and wbc_entity_usage.

With --date latest, the command takes the newest dump in which the job of
each table is done. The newest dump can still run, so it can be older than
the newest date of "wiki dump list". A file that is already in the output
directory with the correct sha1 is not downloaded again. A partial file is
resumed.

Examples:
  wiki dump tables --dry-run
  wiki dump tables --out-dir work/dumps/20261001 --date 20261001
  wiki dump tables page redirect page_props`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := app.Client()
			if err != nil {
				return err
			}
			tables := args
			if len(tables) == 0 {
				tables = wiki.DumpTables
			}
			files, resolved, err := c.DumpTableFiles(cmd.Context(), dumpWiki, date, tables)
			if err != nil {
				return wrapErr(err)
			}
			dir := outDir
			if dir == "" {
				w := dumpWiki
				if w == "" {
					w = c.Site.DumpWiki()
				}
				dir = filepath.Join(c.Cfg.DataDir, "dumps", w, resolved)
			}
			var total int64
			for _, f := range files {
				total += f.Size
			}
			fmt.Fprintf(os.Stderr, "dump %s: %d tables, %s, to %s\n", resolved, len(files), humanSize(total), dir)
			for i, f := range files {
				path := filepath.Join(dir, f.Name)
				state := "planned"
				if !dryRun {
					if state, err = fetchTable(cmd, c, f, dir); err != nil {
						return err
					}
				}
				if err := app.Out.Emit(Row{
					Cols:  []string{"table", "name", "size", "state", "path"},
					Vals:  []string{tables[i], f.Name, itoa64(f.Size), state, path},
					Value: map[string]any{"table": tables[i], "date": resolved, "name": f.Name, "size": f.Size, "sha1": f.Sha1, "state": state, "path": path},
				}); err != nil {
					return err
				}
			}
			return app.Out.Flush()
		},
	}
	cmd.Flags().StringVar(&dumpWiki, "wiki", "", "dump database name (default from --project/-l)")
	cmd.Flags().StringVar(&date, "date", "latest", "dump date YYYYMMDD or latest")
	cmd.Flags().StringVar(&outDir, "out-dir", "", "output directory (default data-dir/dumps/<wiki>/<date>)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "list the files and do not download them")
	return cmd
}

// fetchTable makes sure that the file f is in dir with the correct sha1. It
// gives "have" when the file was already there, and "downloaded" when it
// downloaded the file or the rest of it.
func fetchTable(cmd *cobra.Command, c *wiki.Client, f wiki.DumpFile, dir string) (string, error) {
	path := filepath.Join(dir, f.Name)
	if info, err := os.Stat(path); err == nil {
		switch {
		case info.Size() == f.Size:
			ok, err := wiki.VerifySha1(path, f.Sha1)
			if err != nil {
				return "", err
			}
			if ok {
				return "have", nil
			}
			fmt.Fprintf(os.Stderr, "%s: the sha1 is not correct, so the file is downloaded again\n", f.Name)
			if err := os.Remove(path); err != nil {
				return "", err
			}
		case info.Size() > f.Size:
			if err := os.Remove(path); err != nil {
				return "", err
			}
		}
	}
	_, err := os.Stat(path)
	resumed := err == nil
	for {
		fmt.Fprintf(os.Stderr, "downloading %s (%s)\n", f.Name, humanSize(f.Size))
		got, err := c.DownloadFile(cmd.Context(), f.URL, dir, downloadProgress(f.Size))
		if err != nil {
			return "", wrapErr(err)
		}
		fmt.Fprintln(os.Stderr)
		ok, err := wiki.VerifySha1(got, f.Sha1)
		switch {
		case err != nil:
			return "", err
		case ok:
			return "downloaded", nil
		case !resumed:
			return "", fmt.Errorf("sha1 mismatch for %s", got)
		}
		// The start of the partial file was not correct. Download all of it again.
		fmt.Fprintf(os.Stderr, "%s: the sha1 of the resumed file is not correct, so the file is downloaded again\n", f.Name)
		if err := os.Remove(got); err != nil {
			return "", err
		}
		resumed = false
	}
}
