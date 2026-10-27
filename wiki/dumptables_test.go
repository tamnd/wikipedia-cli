package wiki

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// dumpHost serves the date index of enwiki and a dumpstatus.json for each
// date. status gives the status of the page table job for each date.
func dumpHost(t *testing.T, status map[string]string) *Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/enwiki/" {
			for d := range status {
				_, _ = fmt.Fprintf(w, `<a href="%s/">%s/</a>`+"\n", d, d)
			}
			return
		}
		for d, st := range status {
			if r.URL.Path == "/enwiki/"+d+"/dumpstatus.json" {
				_, _ = fmt.Fprintf(w, `{"jobs":{
"pagetable":{"status":%q,"files":{"enwiki-%s-page.sql.gz":{"size":10,"sha1":"aa","url":"/enwiki/%s/enwiki-%s-page.sql.gz"}}},
"pagepropstable":{"status":"done","files":{"enwiki-%s-page_props.sql.gz":{"size":20,"sha1":"bb","url":"/enwiki/%s/enwiki-%s-page_props.sql.gz"}}},
"articlesdump":{"status":"done","files":{"enwiki-%s-pages-articles.xml.bz2":{"size":30,"url":"/x"}}}}}`,
					st, d, d, d, d, d, d, d)
				return
			}
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(srv.Close)
	cfg := DefaultConfig()
	cfg.DumpsURL = srv.URL
	cfg.Delay = 0
	cfg.Retries = 0
	c, err := New(cfg, NewCache(t.TempDir(), false))
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestTableJob(t *testing.T) {
	for in, want := range map[string]string{"page": "pagetable", "page_props": "pagepropstable", "wbc_entity_usage": "wbcentityusagetable"} {
		if got := TableJob(in); got != want {
			t.Errorf("TableJob(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestDumpTableFiles(t *testing.T) {
	ctx := context.Background()
	c := dumpHost(t, map[string]string{"20260901": "done", "20261001": "done", "20261020": "in-progress"})

	// latest skips the dump that still runs.
	files, date, err := c.DumpTableFiles(ctx, "enwiki", "latest", []string{"page_props", "page"})
	if err != nil {
		t.Fatal(err)
	}
	if date != "20261001" || len(files) != 2 {
		t.Fatalf("date %s, %d files", date, len(files))
	}
	if files[0].Name != "enwiki-20261001-page_props.sql.gz" || files[1].Name != "enwiki-20261001-page.sql.gz" {
		t.Errorf("the files are %s and %s", files[0].Name, files[1].Name)
	}
	if !strings.HasSuffix(files[1].URL, "/enwiki/20261001/enwiki-20261001-page.sql.gz") || files[1].Sha1 != "aa" || files[1].Status != "done" {
		t.Errorf("the page file is %+v", files[1])
	}

	// A concrete date does not step back.
	if _, _, err := c.DumpTableFiles(ctx, "enwiki", "20261020", []string{"page"}); err == nil || !strings.Contains(err.Error(), "the job pagetable is not done") {
		t.Errorf("20261020: %v", err)
	}
	if _, _, err := c.DumpTableFiles(ctx, "enwiki", "20261001", []string{"pagelinks"}); err == nil || !strings.Contains(err.Error(), "no file for the table pagelinks") {
		t.Errorf("pagelinks: %v", err)
	}
}

func TestDumpTableFilesNoDoneDump(t *testing.T) {
	c := dumpHost(t, map[string]string{"20260801": "done", "20260901": "failed", "20261001": "waiting", "20261020": "in-progress"})
	_, _, err := c.DumpTableFiles(context.Background(), "enwiki", "latest", []string{"page"})
	if err == nil || !strings.Contains(err.Error(), "enwiki 20260901: the job pagetable is not done") {
		t.Errorf("err is %v", err)
	}
}

func TestStreamFileResumes(t *testing.T) {
	data := strings.Repeat("0123456789", 100)
	var opens int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		opens++
		from := 0
		if rg := r.Header.Get("Range"); rg != "" {
			if _, err := fmt.Sscanf(rg, "bytes=%d-", &from); err != nil {
				http.Error(w, "bad range", http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", from, len(data)-1, len(data)))
		}
		// Send at most 300 bytes, then drop the connection.
		end := min(from+300, len(data))
		w.Header().Set("Content-Length", fmt.Sprint(len(data)-from))
		if from > 0 {
			w.WriteHeader(http.StatusPartialContent)
		}
		_, _ = w.Write([]byte(data[from:end]))
		w.(http.Flusher).Flush()
		if end < len(data) {
			panic(http.ErrAbortHandler)
		}
	}))
	defer srv.Close()
	c := dumpHost(t, map[string]string{})
	rc := c.StreamFile(context.Background(), srv.URL+"/f.sql.gz")
	defer func() { _ = rc.Close() }()
	got, err := io.ReadAll(rc)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != data {
		t.Fatalf("got %d bytes, want %d", len(got), len(data))
	}
	if opens != 4 {
		t.Errorf("the reader opened the URL %d times, want 4", opens)
	}
}
