package wiki

import (
	"bufio"
	"compress/bzip2"
	"compress/gzip"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// DumpFile is one file produced by a dump job.
type DumpFile struct {
	Job  string `json:"job"`
	Name string `json:"name"`
	Size int64  `json:"size"`
	Sha1 string `json:"sha1,omitempty"`
	URL  string `json:"url"`
	// Status is the status of the job: "done" when the file is complete.
	Status string `json:"status,omitempty"`
}

// DumpList returns the files of a dump for a wiki and date. date may be a
// concrete "YYYYMMDD" or "latest"/"" to resolve the most recent dated dump.
func (c *Client) DumpList(ctx context.Context, wiki, date string) ([]DumpFile, string, error) {
	if wiki == "" {
		wiki = c.Site.DumpWiki()
	}
	if date == "" || date == "latest" {
		d, err := c.latestDumpDate(ctx, wiki)
		if err != nil {
			return nil, "", err
		}
		date = d
	}
	statusURL := fmt.Sprintf("%s/%s/%s/dumpstatus.json", c.Cfg.dumpsBase(), wiki, date)
	var status struct {
		Jobs map[string]struct {
			Status string `json:"status"`
			Files  map[string]struct {
				Size int64  `json:"size"`
				URL  string `json:"url"`
				Sha1 string `json:"sha1"`
			} `json:"files"`
		} `json:"jobs"`
	}
	if err := c.HTTP.GetJSON(ctx, statusURL, ttlDump, &status); err != nil {
		return nil, date, err
	}
	var out []DumpFile
	for job, j := range status.Jobs {
		for name, f := range j.Files {
			out = append(out, DumpFile{
				Job: job, Name: name, Size: f.Size, Sha1: f.Sha1,
				URL: c.Cfg.dumpsBase() + f.URL, Status: j.Status,
			})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, date, nil
}

var dumpDateRe = regexp.MustCompile(`href="(\d{8})/"`)

func (c *Client) latestDumpDate(ctx context.Context, wiki string) (string, error) {
	dates, err := c.dumpDates(ctx, wiki)
	if err != nil {
		return "", err
	}
	return dates[len(dates)-1], nil
}

// dumpDates returns the dates of the dumps of a wiki, oldest first.
func (c *Client) dumpDates(ctx context.Context, wiki string) ([]string, error) {
	body, err := c.HTTP.GetBytes(ctx, fmt.Sprintf("%s/%s/", c.Cfg.dumpsBase(), wiki))
	if err != nil {
		return nil, err
	}
	var dates []string
	for _, m := range dumpDateRe.FindAllStringSubmatch(string(body), -1) {
		dates = append(dates, m[1])
	}
	if len(dates) == 0 {
		return nil, fmt.Errorf("no dated dumps found for %s", wiki)
	}
	sort.Strings(dates)
	return dates, nil
}

// DumpTables are the SQL tables that the base build of a corpus reads: the
// page register, the redirects, the page properties, and the category, link,
// template, language and Wikidata usage tables.
var DumpTables = []string{
	"page", "redirect", "page_props", "categorylinks", "linktarget",
	"pagelinks", "templatelinks", "langlinks", "wbc_entity_usage",
}

// TableJob gives the dump job of a SQL table. The job name is the table name
// with no underscores and the suffix "table", so "page_props" gives
// "pagepropstable".
func TableJob(table string) string {
	return strings.ReplaceAll(table, "_", "") + "table"
}

// tablesBack is the number of dump dates that DumpTableFiles tries for
// "latest" before it stops.
const tablesBack = 3

// DumpTableFiles returns the file of each table, in the order of tables, for
// a wiki and a date. With the date "latest" or "", it takes the newest dump
// in which the job of each table is done, because the newest dump can still
// run. With a concrete date, a job that is not done is an error.
func (c *Client) DumpTableFiles(ctx context.Context, wiki, date string, tables []string) ([]DumpFile, string, error) {
	if wiki == "" {
		wiki = c.Site.DumpWiki()
	}
	dates := []string{date}
	if date == "" || date == "latest" {
		all, err := c.dumpDates(ctx, wiki)
		if err != nil {
			return nil, "", err
		}
		dates = nil
		for i := len(all) - 1; i >= 0 && len(dates) < tablesBack; i-- {
			dates = append(dates, all[i])
		}
	}
	var last error
	for _, d := range dates {
		files, _, err := c.DumpList(ctx, wiki, d)
		if err != nil {
			return nil, d, err
		}
		out, err := tableFiles(files, tables)
		if err == nil {
			return out, d, nil
		}
		last = fmt.Errorf("%s %s: %w", wiki, d, err)
	}
	return nil, "", last
}

// tableFiles picks the .sql.gz file of the job of each table.
func tableFiles(files []DumpFile, tables []string) ([]DumpFile, error) {
	var out []DumpFile
	for _, t := range tables {
		var f *DumpFile
		for i := range files {
			if files[i].Job == TableJob(t) && strings.HasSuffix(files[i].Name, "-"+t+".sql.gz") {
				f = &files[i]
				break
			}
		}
		switch {
		case f == nil:
			return nil, fmt.Errorf("the dump has no file for the table %s", t)
		case f.Status != "done":
			return nil, fmt.Errorf("the job %s is not done (status %q)", f.Job, f.Status)
		}
		out = append(out, *f)
	}
	return out, nil
}

// DownloadFile streams url to outDir, resuming a partial file by byte range and
// returning the final path. progress, if non-nil, is called with bytes copied.
func (c *Client) DownloadFile(ctx context.Context, url, outDir string, progress func(n int64)) (string, error) {
	if outDir == "" {
		outDir = c.Cfg.DownloadDir()
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return "", err
	}
	dest := filepath.Join(outDir, filepath.Base(url))
	var have int64
	if info, err := os.Stat(dest); err == nil {
		have = info.Size()
	}
	rangeHdr := ""
	if have > 0 {
		rangeHdr = fmt.Sprintf("bytes=%d-", have)
	}
	resp, err := c.HTTP.Open(ctx, url, rangeHdr)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	flag := os.O_CREATE | os.O_WRONLY
	if resp.StatusCode == 206 {
		flag |= os.O_APPEND
	} else {
		flag |= os.O_TRUNC
		have = 0
	}
	f, err := os.OpenFile(dest, flag, 0o644)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()

	buf := make([]byte, 1<<20)
	for {
		n, rerr := resp.Body.Read(buf)
		if n > 0 {
			if _, werr := f.Write(buf[:n]); werr != nil {
				return "", werr
			}
			have += int64(n)
			if progress != nil {
				progress(have)
			}
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			return "", rerr
		}
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}
	}
	return dest, nil
}

// VerifySha1 checks a file against an expected sha1 hex digest.
func VerifySha1(path, expected string) (bool, error) {
	if expected == "" {
		return true, nil
	}
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer func() { _ = f.Close() }()
	h := sha1.New()
	if _, err := io.Copy(h, f); err != nil {
		return false, err
	}
	return strings.EqualFold(hex.EncodeToString(h.Sum(nil)), expected), nil
}

// DumpPage is one page record streamed from a pages-articles XML dump.
type DumpPage struct {
	ID        int    `json:"id"`
	NS        int    `json:"ns"`
	Title     string `json:"title"`
	RevID     int    `json:"revid"`
	Timestamp string `json:"timestamp"`
	Redirect  bool   `json:"redirect,omitempty"`
	Text      string `json:"text,omitempty"`
}

// StreamPages parses a local pages-articles XML dump (optionally bz2/gz) and
// calls fn for each page in constant memory. Return an error from fn to stop.
// namespace >= 0 filters to that namespace; withText includes the body.
//
// For .bz2 dumps it prefers the parallel lbzip2/pbzip2 binaries when they are on
// PATH (several times faster than Go's single-threaded decompressor), falling
// back to the standard library otherwise.
func StreamPages(path string, namespace int, withText bool, fn func(DumpPage) error) error {
	rc, err := openDumpReader(path)
	if err != nil {
		return err
	}
	defer func() { _ = rc.Close() }()
	return streamPagesReader(rc, namespace, withText, fn)
}

// openDumpReader returns a decompressed reader for a dump file.
func openDumpReader(path string) (io.ReadCloser, error) {
	switch {
	case strings.HasSuffix(path, ".bz2"):
		return openBzip2(path)
	case strings.HasSuffix(path, ".gz"):
		f, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		gz, err := gzip.NewReader(bufio.NewReaderSize(f, 1<<20))
		if err != nil {
			_ = f.Close()
			return nil, err
		}
		return readCloser{Reader: gz, close: func() error { _ = gz.Close(); return f.Close() }}, nil
	default:
		return os.Open(path)
	}
}

// openBzip2 prefers parallel lbzip2/pbzip2, then falls back to the stdlib.
func openBzip2(path string) (io.ReadCloser, error) {
	for _, name := range []string{"lbzip2", "pbzip2", "bzip2"} {
		if _, err := exec.LookPath(name); err != nil {
			continue
		}
		cmd := exec.Command(name, "-dc", path)
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return nil, err
		}
		if err := cmd.Start(); err != nil {
			return nil, err
		}
		return readCloser{Reader: bufio.NewReaderSize(stdout, 1<<20), close: func() error {
			_ = stdout.Close()
			return cmd.Wait()
		}}, nil
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	r := bzip2.NewReader(bufio.NewReaderSize(f, 1<<20))
	return readCloser{Reader: r, close: f.Close}, nil
}

// readCloser adapts a reader plus a close func into an io.ReadCloser.
type readCloser struct {
	io.Reader
	close func() error
}

func (rc readCloser) Close() error { return rc.close() }

// errStop ends a stream early without surfacing as an error.
var errStop = fmt.Errorf("stop")

func streamPagesReader(r io.Reader, namespace int, withText bool, fn func(DumpPage) error) error {
	dec := xml.NewDecoder(r)
	// MediaWiki dumps occasionally carry non-UTF8 references and the compressed
	// stream can end abruptly; tolerate both rather than aborting the run.
	dec.Strict = false
	dec.CharsetReader = func(_ string, in io.Reader) (io.Reader, error) { return in, nil }
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			if strings.Contains(err.Error(), "unexpected EOF") {
				return nil
			}
			return err
		}
		se, ok := tok.(xml.StartElement)
		if !ok || se.Name.Local != "page" {
			continue
		}
		var page struct {
			Title    string    `xml:"title"`
			NS       int       `xml:"ns"`
			ID       int       `xml:"id"`
			Redirect *struct{} `xml:"redirect"`
			Revision struct {
				ID        int    `xml:"id"`
				Timestamp string `xml:"timestamp"`
				Text      string `xml:"text"`
			} `xml:"revision"`
		}
		if err := dec.DecodeElement(&page, &se); err != nil {
			continue // skip a malformed page rather than killing the stream
		}
		if namespace >= 0 && page.NS != namespace {
			continue
		}
		dp := DumpPage{
			ID: page.ID, NS: page.NS, Title: page.Title,
			RevID: page.Revision.ID, Timestamp: page.Revision.Timestamp,
			Redirect: page.Redirect != nil,
		}
		if withText {
			dp.Text = page.Revision.Text
		}
		if err := fn(dp); err != nil {
			if err == errStop {
				return nil
			}
			return err
		}
	}
}
