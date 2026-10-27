package cli

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/tamnd/wikipedia-cli/wiki/dump"
	"github.com/tamnd/wikipedia-cli/wiki/dump/store"
)

func newDumpLoadCmd(app *App) *cobra.Command {
	var dbPath, dumpWiki, date, name, columns string
	var where, in, index []string
	cmd := &cobra.Command{
		Use:   "load <file.sql[.gz] | table>",
		Short: "Load a SQL table dump into a SQLite store",
		Long: `Load the rows of a SQL table dump into a table of a SQLite store. The
source is a local file, or the name of a table. For a table name, the command
streams the file of the table from the dump host and does not write it to
disk. When the connection drops, the stream continues from the byte that it
reached.

--columns keeps some of the columns. --where col=v1,v2 keeps the rows in which
the column has one of the values. --in col=FILE keeps the rows in which the
column has a value of the file, one value on each line. All the filters must
match. --index adds an index on a kept column after the load. A load replaces
the table, and the table _load of the store records the source, the filters
and the counts of each load.

Examples:
  wiki dump load --db store.sqlite enwiki-20261001-redirect.sql.gz
  wiki dump load --db store.sqlite --date 20261001 page \
    --columns page_id,page_namespace,page_title,page_is_redirect \
    --where page_namespace=0,10,14 --index page_title
  wiki dump load --db store.sqlite --date 20261001 pagelinks --in pl_from=s1.txt`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if dbPath == "" {
				return usageErr("--db is required")
			}
			o := store.Options{Name: name, Where: map[string][]string{}, In: map[string]map[string]bool{}, Index: index}
			if columns != "" {
				o.Columns = strings.Split(columns, ",")
			}
			for _, w := range where {
				col, vals, ok := strings.Cut(w, "=")
				if !ok || col == "" {
					return usageErr(fmt.Sprintf("--where %q: use col=v1,v2", w))
				}
				o.Where[col] = append(o.Where[col], strings.Split(vals, ",")...)
			}
			for _, w := range in {
				col, file, ok := strings.Cut(w, "=")
				if !ok || col == "" || file == "" {
					return usageErr(fmt.Sprintf("--in %q: use col=FILE", w))
				}
				set, err := readValues(file)
				if err != nil {
					return err
				}
				o.In[col] = set
			}

			rc, source, err := openTable(cmd, app, args[0], dumpWiki, date)
			if err != nil {
				return err
			}
			defer func() { _ = rc.Close() }()
			o.Source = source
			sr, err := dump.NewSQLReader(rc)
			if err != nil {
				return fmt.Errorf("%s: %w", source, err)
			}
			db, err := store.Open(dbPath)
			if err != nil {
				return err
			}
			defer func() { _ = db.Close() }()
			fmt.Fprintf(os.Stderr, "loading %s from %s\n", sr.Table(), source)
			res, err := store.Load(cmd.Context(), db, sr, o)
			fmt.Fprintln(os.Stderr)
			if err != nil {
				return fmt.Errorf("%s: %w", source, err)
			}
			if err := app.Out.Emit(Row{
				Cols:  []string{"table", "read", "kept", "source"},
				Vals:  []string{res.Name, itoa64(res.Read), itoa64(res.Kept), source},
				Value: map[string]any{"table": res.Name, "read": res.Read, "kept": res.Kept, "source": source, "db": dbPath},
			}); err != nil {
				return err
			}
			return app.Out.Flush()
		},
	}
	cmd.Flags().StringVar(&dbPath, "db", "", "the SQLite store (required)")
	cmd.Flags().StringVar(&dumpWiki, "wiki", "", "dump database name, for a table name (default from --project/-l)")
	cmd.Flags().StringVar(&date, "date", "latest", "dump date YYYYMMDD or latest, for a table name")
	cmd.Flags().StringVar(&name, "as", "", "the name of the table in the store (default the dump table)")
	cmd.Flags().StringVar(&columns, "columns", "", "the columns to keep, separated by commas (default all)")
	cmd.Flags().StringArrayVar(&where, "where", nil, "keep the rows in which col has one of the values: col=v1,v2 (repeatable)")
	cmd.Flags().StringArrayVar(&in, "in", nil, "keep the rows in which col has a value of FILE: col=FILE (repeatable)")
	cmd.Flags().StringArrayVar(&index, "index", nil, "add an index on a kept column (repeatable)")
	return cmd
}

// openTable opens the source of a load: a local file, or the file of a
// table on the dump host as a stream. It gives the decompressed reader and the
// source.
func openTable(cmd *cobra.Command, app *App, arg, dumpWiki, date string) (io.ReadCloser, string, error) {
	var raw io.ReadCloser
	var source string
	var size int64
	if info, err := os.Stat(arg); err == nil {
		f, err := os.Open(arg) // #nosec G304 -- the user names the dump file
		if err != nil {
			return nil, "", err
		}
		raw, source, size = f, arg, info.Size()
	} else {
		if strings.ContainsAny(arg, "/.") {
			return nil, "", notFound(arg + ": no such file")
		}
		c, err := app.Client()
		if err != nil {
			return nil, "", err
		}
		files, _, err := c.DumpTableFiles(cmd.Context(), dumpWiki, date, []string{arg})
		if err != nil {
			return nil, "", wrapErr(err)
		}
		raw, source, size = c.StreamFile(cmd.Context(), files[0].URL), files[0].URL, files[0].Size
	}
	r := io.Reader(&progressReader{r: raw, total: size})
	if !strings.HasSuffix(source, ".gz") {
		return readCloser{r, raw}, source, nil
	}
	gz, err := gzip.NewReader(bufio.NewReaderSize(r, 1<<20))
	if err != nil {
		_ = raw.Close()
		return nil, "", fmt.Errorf("%s: %w", source, err)
	}
	return readCloser{gz, raw}, source, nil
}

type readCloser struct {
	io.Reader
	c io.Closer
}

func (r readCloser) Close() error { return r.c.Close() }

// progressReader prints the bytes read to stderr, at most once a second.
type progressReader struct {
	r     io.Reader
	total int64
	have  int64
	last  time.Time
}

func (p *progressReader) Read(b []byte) (int, error) {
	n, err := p.r.Read(b)
	p.have += int64(n)
	if now := time.Now(); now.Sub(p.last) >= time.Second || err == io.EOF {
		p.last = now
		downloadProgress(p.total)(p.have)
	}
	return n, err
}

// readValues reads a file with one value on each line. It skips empty lines
// and lines that start with #.
func readValues(path string) (map[string]bool, error) {
	f, err := os.Open(path) // #nosec G304 -- the user names the file
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	set := map[string]bool{}
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64<<10), 1<<20)
	for sc.Scan() {
		if v := strings.TrimSpace(sc.Text()); v != "" && !strings.HasPrefix(v, "#") {
			set[v] = true
		}
	}
	return set, sc.Err()
}
