// Package store loads the rows of a SQL table dump into a SQLite database.
//
// A load keeps the columns and the rows that the options select, so the
// store of a large wiki can hold only the part of a table that a job reads.
// Each load records its source, its filters and its counts in the table
// _load.
package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/tamnd/wikipedia-cli/wiki/dump"

	_ "modernc.org/sqlite" // the database/sql driver "sqlite"
)

// Open opens or creates a store. The store does not keep a rollback
// journal, because a load that stops is run again.
func Open(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	for _, p := range []string{"PRAGMA journal_mode=OFF", "PRAGMA synchronous=OFF", "PRAGMA temp_store=MEMORY"} {
		if _, err := db.Exec(p); err != nil {
			_ = db.Close()
			return nil, err
		}
	}
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS _load (
		name TEXT PRIMARY KEY, dump_table TEXT, source TEXT, columns TEXT, filters TEXT,
		rows_read INTEGER, rows_kept INTEGER, loaded TEXT)`); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

// Options select what a load keeps.
type Options struct {
	// Name is the name of the table in the store. Empty means the name of the
	// dump table.
	Name string
	// Columns are the columns to keep, in order. Empty means all columns.
	Columns []string
	// Where keeps a row when each named column has one of its values.
	Where map[string][]string
	// In keeps a row when each named column has a value of its set.
	In map[string]map[string]bool
	// Index names the columns that get an index after the load.
	Index []string
	// Source is the file or URL of the dump. It is recorded in _load.
	Source string
}

// Result gives the counts of a load.
type Result struct {
	Name string `json:"name"`
	Read int64  `json:"read"`
	Kept int64  `json:"kept"`
}

// batch is the number of rows in one transaction.
const batch = 50000

type filter struct {
	col  int
	vals map[string]bool
}

// Load reads each row of sr and writes the rows that o selects to the table
// o.Name. It replaces the table when it exists.
func Load(ctx context.Context, db *sql.DB, sr *dump.SQLReader, o Options) (Result, error) {
	name := o.Name
	if name == "" {
		name = sr.Table()
	}
	if !validName(name) {
		return Result{}, fmt.Errorf("store: %q is not a valid table name", name)
	}
	cols := o.Columns
	if len(cols) == 0 {
		for _, c := range sr.Columns() {
			cols = append(cols, c.Name)
		}
	}
	idx, err := sr.Index(cols...)
	if err != nil {
		return Result{}, err
	}
	filters, err := makeFilters(sr, o)
	if err != nil {
		return Result{}, err
	}
	for _, c := range o.Index {
		if !slices.Contains(cols, c) {
			return Result{}, fmt.Errorf("store: the index column %s is not a kept column", c)
		}
	}

	ints := make([]bool, len(idx))
	defs := make([]string, len(idx))
	for i, j := range idx {
		c := sr.Columns()[j]
		typ := sqlType(c.Type)
		ints[i] = typ == "INTEGER"
		defs[i] = quote(c.Name) + " " + typ
	}
	if _, err := db.ExecContext(ctx, "DROP TABLE IF EXISTS "+quote(name)); err != nil {
		return Result{}, err
	}
	if _, err := db.ExecContext(ctx, "CREATE TABLE "+quote(name)+" ("+strings.Join(defs, ", ")+")"); err != nil {
		return Result{}, err
	}
	insert := "INSERT INTO " + quote(name) + " VALUES (" + strings.TrimSuffix(strings.Repeat("?,", len(idx)), ",") + ")"

	res := Result{Name: name}
	args := make([]any, len(idx))
	var tx *sql.Tx
	var stmt *sql.Stmt
	commit := func() error {
		if tx == nil {
			return nil
		}
		_ = stmt.Close()
		err := tx.Commit()
		tx, stmt = nil, nil
		return err
	}
	defer func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}()
	for {
		row, err := sr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return res, err
		}
		res.Read++
		if !keep(row, filters) {
			continue
		}
		if tx == nil {
			if tx, err = db.BeginTx(ctx, nil); err != nil {
				return res, err
			}
			if stmt, err = tx.PrepareContext(ctx, insert); err != nil {
				return res, err
			}
		}
		for i, j := range idx {
			args[i] = value(row[j], ints[i])
		}
		if _, err := stmt.ExecContext(ctx, args...); err != nil {
			return res, err
		}
		res.Kept++
		if res.Kept%batch == 0 {
			if err := commit(); err != nil {
				return res, err
			}
		}
	}
	if err := commit(); err != nil {
		return res, err
	}
	for _, c := range o.Index {
		q := "CREATE INDEX " + quote(name+"_"+c) + " ON " + quote(name) + " (" + quote(c) + ")"
		if _, err := db.ExecContext(ctx, q); err != nil {
			return res, err
		}
	}
	_, err = db.ExecContext(ctx, `INSERT OR REPLACE INTO _load VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		name, sr.Table(), o.Source, strings.Join(cols, ","), describe(o), res.Read, res.Kept, time.Now().UTC().Format(time.RFC3339))
	return res, err
}

func makeFilters(sr *dump.SQLReader, o Options) ([]filter, error) {
	var out []filter
	add := func(col string, vals map[string]bool) error {
		j, err := sr.Index(col)
		if err != nil {
			return err
		}
		out = append(out, filter{col: j[0], vals: vals})
		return nil
	}
	for _, col := range sortedKeys(o.Where) {
		vals := map[string]bool{}
		for _, v := range o.Where[col] {
			vals[v] = true
		}
		if err := add(col, vals); err != nil {
			return nil, err
		}
	}
	for _, col := range sortedKeys(o.In) {
		if err := add(col, o.In[col]); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func keep(row dump.Row, filters []filter) bool {
	for _, f := range filters {
		if v := row[f.col]; v.Null || !f.vals[v.Raw] {
			return false
		}
	}
	return true
}

// value gives the SQL value of a field. An integer column that does not
// parse keeps its text.
func value(v dump.Value, isInt bool) any {
	if v.Null {
		return nil
	}
	if isInt {
		if n, err := strconv.ParseInt(v.Raw, 10, 64); err == nil {
			return n
		}
	}
	return v.Raw
}

// sqlType maps a MySQL column type to a SQLite type.
func sqlType(t string) string {
	t = strings.ToLower(t)
	switch {
	case strings.Contains(t, "int"):
		return "INTEGER"
	case strings.HasPrefix(t, "float"), strings.HasPrefix(t, "double"), strings.HasPrefix(t, "decimal"):
		return "REAL"
	}
	return "TEXT"
}

func describe(o Options) string {
	var parts []string
	for _, col := range sortedKeys(o.Where) {
		parts = append(parts, col+"="+strings.Join(o.Where[col], ","))
	}
	for _, col := range sortedKeys(o.In) {
		parts = append(parts, fmt.Sprintf("%s in %d values", col, len(o.In[col])))
	}
	return strings.Join(parts, "; ")
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}

func validName(s string) bool {
	if s == "" || strings.HasPrefix(s, "_") {
		return false
	}
	for _, r := range s {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != '_' {
			return false
		}
	}
	return true
}

func quote(s string) string { return `"` + strings.ReplaceAll(s, `"`, `""`) + `"` }
