package dump

import (
	"io"
	"os"
	"strings"
	"testing"
)

const oldForm = "-- MySQL dump\n" +
	"CREATE TABLE `page_props` (\n" +
	"  `pp_page` int(10) unsigned NOT NULL,\n" +
	"  `pp_propname` varbinary(60) NOT NULL,\n" +
	"  `pp_value` blob NOT NULL,\n" +
	"  `pp_sortkey` float DEFAULT NULL,\n" +
	"  PRIMARY KEY (`pp_page`,`pp_propname`)\n" +
	") ENGINE=InnoDB DEFAULT CHARSET=binary;\n" +
	"/*!40000 ALTER TABLE `page_props` DISABLE KEYS */;\n" +
	"INSERT INTO `page_props` VALUES (12,'wikibase_item','Q6199',NULL),(25,'wikibase-shortdesc','It\\'s a \\\"test\\\"\\\\n',1.5);\n" +
	"INSERT INTO `page_props` VALUES (39,'disambiguation','',NULL),(40,'x','a''b,c)',-2);\n" +
	"/*!40000 ALTER TABLE `page_props` ENABLE KEYS */;\n"

func readAll(t *testing.T, sr *SQLReader) []Row {
	t.Helper()
	var rows []Row
	for {
		row, err := sr.Next()
		if err == io.EOF {
			return rows
		}
		if err != nil {
			t.Fatal(err)
		}
		rows = append(rows, row)
	}
}

func TestOldForm(t *testing.T) {
	sr, err := NewSQLReader(strings.NewReader(oldForm))
	if err != nil {
		t.Fatal(err)
	}
	if sr.Table() != "page_props" {
		t.Fatalf("table %q", sr.Table())
	}
	if len(sr.Columns()) != 4 || sr.Columns()[3].Type != "float" {
		t.Fatalf("columns %+v", sr.Columns())
	}
	rows := readAll(t, sr)
	if len(rows) != 4 {
		t.Fatalf("got %d rows", len(rows))
	}
	if rows[0][2].String() != "Q6199" || !rows[0][3].Null {
		t.Errorf("row 0 %+v", rows[0])
	}
	if got := rows[1][2].String(); got != "It's a \"test\"\\n" {
		t.Errorf("escapes: %q", got)
	}
	if got := rows[3][2].String(); got != "a'b,c)" {
		t.Errorf("doubled quote: %q", got)
	}
	if n, err := rows[3][3].Int(); err != nil || n != -2 {
		t.Errorf("int %d %v", n, err)
	}
	if !rows[1][2].Quoted || rows[1][0].Quoted {
		t.Errorf("quoted flags")
	}
}

func TestNewForm(t *testing.T) {
	sr, err := OpenSQL("testdata/redirect-head.sql")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = sr.Close() }()
	idx, err := sr.Index("rd_from", "rd_title", "rd_fragment")
	if err != nil {
		t.Fatal(err)
	}
	rows := readAll(t, sr)
	last := rows[len(rows)-1]
	if last[idx[1]].String() != "Automorphism" {
		t.Errorf("last row %+v", last)
	}
	if rows[0][idx[0]].String() != "10" || rows[0][idx[2]].String() != "" {
		t.Errorf("first row %+v", rows[0])
	}
}

func TestMissingColumn(t *testing.T) {
	sr, err := NewSQLReader(strings.NewReader(oldForm))
	if err != nil {
		t.Fatal(err)
	}
	_, err = sr.Index("pp_page", "pl_title", "pl_namespace")
	if err == nil || !strings.Contains(err.Error(), "pl_title, pl_namespace") {
		t.Fatalf("err %v", err)
	}
}

func TestBadInput(t *testing.T) {
	if _, err := NewSQLReader(strings.NewReader("-- nothing\n")); err == nil {
		t.Error("no CREATE TABLE: want an error")
	}
	head := "CREATE TABLE `t` (\n  `a` int,\n  `b` int\n);\n"
	for _, body := range []string{
		"INSERT INTO `t` VALUES (1,2,3);\n",
		"INSERT INTO `t` VALUES (1,'x",
		"INSERT INTO `t` VALUES (1,2)x;\n",
	} {
		sr, err := NewSQLReader(strings.NewReader(head + body))
		if err != nil {
			t.Fatal(err)
		}
		for {
			_, err = sr.Next()
			if err != nil {
				break
			}
		}
		if err == io.EOF {
			t.Errorf("%q: want an error, got io.EOF", body)
		}
	}
}

func TestSums(t *testing.T) {
	dir := t.TempDir()
	p := dir + "/enwiki-x-redirect.sql.gz"
	if err := os.WriteFile(p, []byte("abc"), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := ParseSums(strings.NewReader("a9993e364706816aba3e25717850c26c9cd0d89d  enwiki-x-redirect.sql.gz\n\n"))
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Check(p); err != nil {
		t.Error(err)
	}
	s["enwiki-x-redirect.sql.gz"] = strings.Repeat("0", 40)
	if err := s.Check(p); err == nil {
		t.Error("bad digest: want an error")
	}
	if err := s.Check(dir + "/other"); err == nil {
		t.Error("no digest: want an error")
	}
	if _, err := ParseSums(strings.NewReader("zz  name\n")); err == nil {
		t.Error("bad line: want an error")
	}
}
