package dump

import (
	"bufio"
	"compress/bzip2"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"
)

// The pages-articles-multistream dump is a series of bz2 streams. Each
// stream holds about 100 pages. The index file gives, for each page, the
// byte offset of its stream in the dump, the page id and the title, as
// "offset:pageid:title" on one line. So a reader can read the pages that it
// needs and skip the other streams.

// IndexEntry is one line of the index of a multistream dump.
type IndexEntry struct {
	Offset int64
	PageID int64
	Title  string
}

// ReadIndex reads the index of a multistream dump. The index can be bz2 or
// plain text. fn gets each entry in the order of the file.
func ReadIndex(path string, fn func(IndexEntry) error) error {
	f, err := os.Open(path) // #nosec G304 -- the caller names the index
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	var r io.Reader = bufio.NewReaderSize(f, 1<<20)
	if strings.HasSuffix(path, ".bz2") {
		r = bzip2.NewReader(r)
	}
	return readIndex(r, fn)
}

func readIndex(r io.Reader, fn func(IndexEntry) error) error {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64<<10), 1<<20)
	line := 0
	for sc.Scan() {
		line++
		l := sc.Text()
		a, rest, ok1 := strings.Cut(l, ":")
		b, title, ok2 := strings.Cut(rest, ":")
		if !ok1 || !ok2 {
			return fmt.Errorf("index line %d: want offset:pageid:title", line)
		}
		off, err := strconv.ParseInt(a, 10, 64)
		if err != nil {
			return fmt.Errorf("index line %d: %w", line, err)
		}
		id, err := strconv.ParseInt(b, 10, 64)
		if err != nil {
			return fmt.Errorf("index line %d: %w", line, err)
		}
		if err := fn(IndexEntry{Offset: off, PageID: id, Title: title}); err != nil {
			return err
		}
	}
	return sc.Err()
}

// Stream is one bz2 stream of a multistream dump, from the byte Start to the
// byte End. End is -1 for the last stream, which goes to the end of the file.
type Stream struct {
	Start, End int64
}

// StreamsOf gives the streams of the index that hold one or more of the
// pages, in the order of the dump. It also gives the ids of the pages that
// the index does not have.
func StreamsOf(indexPath string, pages map[int64]bool) ([]Stream, []int64, error) {
	var (
		out  []Stream
		cur  int64 = -1
		need bool
		seen = map[int64]bool{}
	)
	err := ReadIndex(indexPath, func(e IndexEntry) error {
		if e.Offset != cur {
			if need {
				out = append(out, Stream{Start: cur, End: e.Offset})
			}
			cur, need = e.Offset, false
		}
		if pages[e.PageID] {
			need = true
			seen[e.PageID] = true
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	if need {
		out = append(out, Stream{Start: cur, End: -1})
	}
	var missing []int64
	for id := range pages {
		if !seen[id] {
			missing = append(missing, id)
		}
	}
	slices.Sort(missing)
	return out, missing, nil
}

// Page is one page of a pages-articles dump, with its one revision.
type Page struct {
	ID        int64
	NS        int
	Title     string
	Redirect  string // the target of a redirect, or ""
	RevID     int64
	ParentID  int64
	Timestamp string
	SHA1      string
	Text      string
}

type xmlPage struct {
	Title    string `xml:"title"`
	NS       int    `xml:"ns"`
	ID       int64  `xml:"id"`
	Redirect *struct {
		Title string `xml:"title,attr"`
	} `xml:"redirect"`
	Revision struct {
		ID        int64  `xml:"id"`
		ParentID  int64  `xml:"parentid"`
		Timestamp string `xml:"timestamp"`
		SHA1      string `xml:"sha1"`
		Text      string `xml:"text"`
	} `xml:"revision"`
}

// ReadStream reads the pages of one stream of the dump.
func ReadStream(f io.ReaderAt, size int64, s Stream) ([]Page, error) {
	end := s.End
	if end < 0 {
		end = size
	}
	if s.Start < 0 || end < s.Start || end > size {
		return nil, fmt.Errorf("the stream %d to %d is not in the file of %d bytes", s.Start, end, size)
	}
	r := bzip2.NewReader(bufio.NewReaderSize(io.NewSectionReader(f, s.Start, end-s.Start), 1<<16))
	dec := xml.NewDecoder(r)
	var out []Page
	for {
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			return out, nil
		}
		// A stream is a part of one XML document. The first stream opens
		// <mediawiki> and does not close it, and the last stream closes it.
		// So a syntax error between two pages is the edge of the stream.
		var se *xml.SyntaxError
		if errors.As(err, &se) {
			return out, nil
		}
		if err != nil {
			return nil, fmt.Errorf("the stream at %d: %w", s.Start, err)
		}
		start, ok := tok.(xml.StartElement)
		if !ok || start.Name.Local != "page" {
			continue
		}
		var p xmlPage
		if err := dec.DecodeElement(&p, &start); err != nil {
			return nil, fmt.Errorf("the stream at %d: %w", s.Start, err)
		}
		pg := Page{ID: p.ID, NS: p.NS, Title: p.Title, RevID: p.Revision.ID, ParentID: p.Revision.ParentID, Timestamp: p.Revision.Timestamp, SHA1: p.Revision.SHA1, Text: p.Revision.Text}
		if p.Redirect != nil {
			pg.Redirect = p.Redirect.Title
		}
		out = append(out, pg)
	}
}

// ReadPages reads the streams of the dump with the number of workers, and
// calls fn for each page of the set, in the order of the dump. fn runs on
// one goroutine at a time. An error of fn or of a stream stops the read.
func ReadPages(dumpPath string, streams []Stream, pages map[int64]bool, workers int, fn func(Page) error) error {
	f, err := os.Open(dumpPath) // #nosec G304 -- the caller names the dump
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	st, err := f.Stat()
	if err != nil {
		return err
	}
	workers = max(workers, 1)
	type result struct {
		pages []Page
		err   error
	}
	// Each stream has its own channel, and the channels are read in order,
	// so fn sees the pages in the order of the dump. The queue holds at most
	// 2 * workers streams.
	queue := make(chan chan result, 2*workers)
	jobs := make(chan struct {
		s   Stream
		out chan result
	})
	done := make(chan struct{})
	var wg sync.WaitGroup
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				ps, err := ReadStream(f, st.Size(), j.s)
				j.out <- result{ps, err}
			}
		}()
	}
	go func() {
		defer close(jobs)
		defer close(queue)
		for _, s := range streams {
			out := make(chan result, 1)
			select {
			case queue <- out:
			case <-done:
				return
			}
			select {
			case jobs <- struct {
				s   Stream
				out chan result
			}{s, out}:
			case <-done:
				return
			}
		}
	}()
	stop := func(err error) error {
		close(done)
		for range queue {
		}
		wg.Wait()
		return err
	}
	for out := range queue {
		r := <-out
		if r.err != nil {
			return stop(r.err)
		}
		for _, p := range r.pages {
			if !pages[p.ID] {
				continue
			}
			if err := fn(p); err != nil {
				return stop(err)
			}
		}
	}
	wg.Wait()
	return nil
}
