package wiki

import (
	"compress/bzip2"
	"compress/gzip"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"io"
	"os"
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
	statusURL := fmt.Sprintf("%s/%s/%s/dumpstatus.json", DumpsBase, wiki, date)
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
				URL: DumpsBase + f.URL,
			})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, date, nil
}

var dumpDateRe = regexp.MustCompile(`href="(\d{8})/"`)

func (c *Client) latestDumpDate(ctx context.Context, wiki string) (string, error) {
	body, err := c.HTTP.GetBytes(ctx, fmt.Sprintf("%s/%s/", DumpsBase, wiki))
	if err != nil {
		return "", err
	}
	var dates []string
	for _, m := range dumpDateRe.FindAllStringSubmatch(string(body), -1) {
		dates = append(dates, m[1])
	}
	if len(dates) == 0 {
		return "", fmt.Errorf("no dated dumps found for %s", wiki)
	}
	sort.Strings(dates)
	return dates[len(dates)-1], nil
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
	Text      string `json:"text,omitempty"`
}

// StreamPages parses a local pages-articles XML dump (optionally bz2/gz) and
// calls fn for each page in constant memory. Return an error from fn to stop.
// namespace >= 0 filters to that namespace; withText includes the body.
func StreamPages(path string, namespace int, withText bool, fn func(DumpPage) error) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	var r io.Reader = f
	switch {
	case strings.HasSuffix(path, ".bz2"):
		r = bzip2.NewReader(f)
	case strings.HasSuffix(path, ".gz"):
		gz, err := gzip.NewReader(f)
		if err != nil {
			return err
		}
		defer func() { _ = gz.Close() }()
		r = gz
	}
	return streamPagesReader(r, namespace, withText, fn)
}

// errStop ends a stream early without surfacing as an error.
var errStop = fmt.Errorf("stop")

func streamPagesReader(r io.Reader, namespace int, withText bool, fn func(DumpPage) error) error {
	dec := xml.NewDecoder(r)
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		se, ok := tok.(xml.StartElement)
		if !ok || se.Name.Local != "page" {
			continue
		}
		var page struct {
			Title    string `xml:"title"`
			NS       int    `xml:"ns"`
			ID       int    `xml:"id"`
			Revision struct {
				ID        int    `xml:"id"`
				Timestamp string `xml:"timestamp"`
				Text      string `xml:"text"`
			} `xml:"revision"`
		}
		if err := dec.DecodeElement(&page, &se); err != nil {
			return err
		}
		if namespace >= 0 && page.NS != namespace {
			continue
		}
		dp := DumpPage{
			ID: page.ID, NS: page.NS, Title: page.Title,
			RevID: page.Revision.ID, Timestamp: page.Revision.Timestamp,
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
