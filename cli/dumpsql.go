package cli

import (
	"io"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tamnd/wikipedia-cli/wiki/dump"
)

func newDumpSQLCmd(app *App) *cobra.Command {
	var columns string
	var schema bool
	cmd := &cobra.Command{
		Use:   "sql <file.sql[.gz]>",
		Short: "Stream the rows of a SQL table dump",
		Long: `Stream the rows of a SQL table dump (page, redirect, page_props,
categorylinks, pagelinks, linktarget and the others) as records. The reader
takes the column names from the CREATE TABLE statement of the file. It does
not need a database server. --columns selects columns by name, and a column
that the table does not have is an error. --schema prints the columns.

Examples:
  wiki dump sql enwiki-20261001-redirect.sql.gz -n 10
  wiki dump sql enwiki-20261001-page_props.sql.gz --columns pp_page,pp_propname,pp_value -o tsv
  wiki dump sql enwiki-20261001-pagelinks.sql.gz --schema`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sr, err := dump.OpenSQL(args[0])
			if err != nil {
				return wrapErr(err)
			}
			defer func() { _ = sr.Close() }()
			if schema {
				for _, c := range sr.Columns() {
					if err := app.Out.Emit(Row{
						Cols: []string{"table", "column", "type"}, Vals: []string{sr.Table(), c.Name, c.Type}, Value: c,
					}); err != nil {
						return err
					}
				}
				return app.Out.Flush()
			}
			var names []string
			if columns != "" {
				names = strings.Split(columns, ",")
			} else {
				for _, c := range sr.Columns() {
					names = append(names, c.Name)
				}
			}
			idx, err := sr.Index(names...)
			if err != nil {
				return usageErr(err.Error())
			}
			n := 0
			for {
				row, err := sr.Next()
				if err == io.EOF {
					break
				}
				if err != nil {
					return wrapErr(err)
				}
				vals := make([]string, len(idx))
				obj := make(map[string]any, len(idx))
				for i, j := range idx {
					vals[i] = row[j].String()
					if row[j].Null {
						obj[names[i]] = nil
					} else {
						obj[names[i]] = row[j].String()
					}
				}
				if err := app.Out.Emit(Row{Cols: names, Vals: vals, Value: obj}); err != nil {
					return err
				}
				n++
				if app.Limit > 0 && n >= app.Limit {
					break
				}
			}
			if n == 0 {
				return noResults("no rows in " + sr.Table())
			}
			return app.Out.Flush()
		},
	}
	cmd.Flags().StringVar(&columns, "columns", "", "comma-separated column names (default all)")
	cmd.Flags().BoolVar(&schema, "schema", false, "print the columns of the table")
	return cmd
}
