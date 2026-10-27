package store

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tamnd/wikipedia-cli/wiki/dump"
)

const pageDump = "CREATE TABLE `page` (\n" +
	"  `page_id` int(8) unsigned NOT NULL AUTO_INCREMENT,\n" +
	"  `page_namespace` int(11) NOT NULL DEFAULT 0,\n" +
	"  `page_title` varbinary(255) NOT NULL DEFAULT '',\n" +
	"  `page_is_redirect` tinyint(1) unsigned NOT NULL DEFAULT 0,\n" +
	"  `page_random` double unsigned NOT NULL DEFAULT 0,\n" +
	"  `page_lang` varbinary(35) DEFAULT NULL,\n" +
	"  PRIMARY KEY (`page_id`)\n" +
	") ENGINE=InnoDB;\n" +
	"INSERT INTO `page` VALUES (1,0,'Group_(mathematics)',0,0.5,NULL),(2,14,'Group_theory',0,0.25,NULL),(3,0,'Groups',1,0.75,'en');\n" +
	"INSERT INTO `page` VALUES (4,0,'Coset',0,0.1,NULL),(5,10,'Math_theorem',0,0.2,NULL);\n"

func reader(t *testing.T) *dump.SQLReader {
	t.Helper()
	sr, err := dump.NewSQLReader(strings.NewReader(pageDump))
	if err != nil {
		t.Fatal(err)
	}
	return sr
}

func open(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "store.sqlite")
	db, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadAll(t *testing.T) {
	db, err := Open(open(t))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	res, err := Load(context.Background(), db, reader(t), Options{Source: "page.sql"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Name != "page" || res.Read != 5 || res.Kept != 5 {
		t.Fatalf("result %+v", res)
	}
	var title string
	var random float64
	var lang any
	if err := db.QueryRow(`SELECT page_title, page_random, page_lang FROM page WHERE page_id = 1`).Scan(&title, &random, &lang); err != nil {
		t.Fatal(err)
	}
	if title != "Group_(mathematics)" || random != 0.5 || lang != nil {
		t.Errorf("row 1 is %q, %v, %v", title, random, lang)
	}
	var typ string
	if err := db.QueryRow(`SELECT typeof(page_id) FROM page LIMIT 1`).Scan(&typ); err != nil || typ != "integer" {
		t.Errorf("page_id has the type %q, %v", typ, err)
	}
	var source string
	var kept int
	if err := db.QueryRow(`SELECT source, rows_kept FROM _load WHERE name = 'page'`).Scan(&source, &kept); err != nil || source != "page.sql" || kept != 5 {
		t.Errorf("the _load row is %q, %d, %v", source, kept, err)
	}
}

func TestLoadFilters(t *testing.T) {
	db, err := Open(open(t))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	o := Options{
		Name:    "articles",
		Columns: []string{"page_id", "page_title"},
		Where:   map[string][]string{"page_namespace": {"0", "14"}, "page_is_redirect": {"0"}},
		In:      map[string]map[string]bool{"page_id": {"1": true, "2": true, "3": true, "5": true}},
		Index:   []string{"page_title"},
	}
	res, err := Load(context.Background(), db, reader(t), o)
	if err != nil {
		t.Fatal(err)
	}
	if res.Read != 5 || res.Kept != 2 {
		t.Fatalf("result %+v", res)
	}
	rows, err := db.Query(`SELECT page_id, page_title FROM articles ORDER BY page_id`)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rows.Close() }()
	var got []string
	for rows.Next() {
		var id int
		var title string
		if err := rows.Scan(&id, &title); err != nil {
			t.Fatal(err)
		}
		got = append(got, title)
	}
	if strings.Join(got, "|") != "Group_(mathematics)|Group_theory" {
		t.Errorf("the rows are %v", got)
	}
	var n int
	if err := db.QueryRow(`SELECT count(*) FROM sqlite_master WHERE type = 'index' AND name = 'articles_page_title'`).Scan(&n); err != nil || n != 1 {
		t.Errorf("the index: %d, %v", n, err)
	}
	var filters string
	if err := db.QueryRow(`SELECT filters FROM _load WHERE name = 'articles'`).Scan(&filters); err != nil || filters != "page_is_redirect=0; page_namespace=0,14; page_id in 4 values" {
		t.Errorf("the filters are %q, %v", filters, err)
	}

	// A second load replaces the table.
	o.Where = map[string][]string{"page_namespace": {"10"}}
	o.In = nil
	if res, err := Load(context.Background(), db, reader(t), o); err != nil || res.Kept != 1 {
		t.Fatalf("second load: %+v, %v", res, err)
	}
	if err := db.QueryRow(`SELECT count(*) FROM articles`).Scan(&n); err != nil || n != 1 {
		t.Errorf("after the second load: %d rows, %v", n, err)
	}
}

func TestLoadErrors(t *testing.T) {
	db, err := Open(open(t))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	for _, c := range []struct {
		o    Options
		want string
	}{
		{Options{Columns: []string{"page_id", "cl_to"}}, "has no column cl_to"},
		{Options{Where: map[string][]string{"cl_type": {"page"}}}, "has no column cl_type"},
		{Options{Columns: []string{"page_id"}, Index: []string{"page_title"}}, "not a kept column"},
		{Options{Name: "_load"}, "not a valid table name"},
		{Options{Name: "page; DROP"}, "not a valid table name"},
	} {
		if _, err := Load(context.Background(), db, reader(t), c.o); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%+v: %v", c.o, err)
		}
	}
}
