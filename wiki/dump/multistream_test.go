package dump

import (
	"errors"
	"os"
	"slices"
	"strings"
	"testing"
)

const (
	testDump  = "testdata/multistream.xml.bz2"
	testIndex = "testdata/multistream-index.txt.bz2"
)

func TestReadIndex(t *testing.T) {
	for _, path := range []string{testIndex, "testdata/multistream-index.txt"} {
		var got []IndexEntry
		if err := ReadIndex(path, func(e IndexEntry) error { got = append(got, e); return nil }); err != nil {
			t.Fatal(err)
		}
		want := []IndexEntry{{137, 10, "Coset"}, {137, 11, "Talk:Coset"}, {429, 12, "Cosets"}}
		if !slices.Equal(got, want) {
			t.Errorf("%s: got %v, want %v", path, got, want)
		}
	}
	if err := readIndex(strings.NewReader("12:x\n"), func(IndexEntry) error { return nil }); err == nil {
		t.Error("a line with no title gives no error")
	}
}

func TestStreamsOf(t *testing.T) {
	ss, missing, err := StreamsOf(testIndex, map[int64]bool{10: true, 12: true, 99: true})
	if err != nil {
		t.Fatal(err)
	}
	if want := []Stream{{137, 429}, {429, -1}}; !slices.Equal(ss, want) {
		t.Errorf("streams %v, want %v", ss, want)
	}
	if !slices.Equal(missing, []int64{99}) {
		t.Errorf("missing %v", missing)
	}
	ss, _, _ = StreamsOf(testIndex, map[int64]bool{11: true})
	if want := []Stream{{137, 429}}; !slices.Equal(ss, want) {
		t.Errorf("streams %v, want %v", ss, want)
	}
}

func TestReadPages(t *testing.T) {
	set := map[int64]bool{10: true, 12: true}
	ss, _, err := StreamsOf(testIndex, set)
	if err != nil {
		t.Fatal(err)
	}
	for _, workers := range []int{1, 4} {
		var got []Page
		if err := ReadPages(testDump, ss, set, workers, func(p Page) error { got = append(got, p); return nil }); err != nil {
			t.Fatal(err)
		}
		if len(got) != 2 {
			t.Fatalf("got %d pages", len(got))
		}
		p := got[0]
		if p.ID != 10 || p.Title != "Coset" || p.RevID != 1001 || p.ParentID != 1000 || p.SHA1 != "abc" || p.Timestamp != "2026-09-01T10:00:00Z" || p.Text != "A '''coset''' of <math>H</math> is a set." || p.Redirect != "" {
			t.Errorf("page 10: %+v", p)
		}
		if got[1].ID != 12 || got[1].Redirect != "Coset" {
			t.Errorf("page 12: %+v", got[1])
		}
	}
	stop := errors.New("stop")
	if err := ReadPages(testDump, ss, set, 2, func(Page) error { return stop }); !errors.Is(err, stop) {
		t.Errorf("the error of fn: %v", err)
	}
}

func TestReadStreamBounds(t *testing.T) {
	f, err := os.Open(testDump)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	if _, err := ReadStream(f, 674, Stream{500, 400}); err == nil {
		t.Error("a stream that ends before it starts gives no error")
	}
	if _, err := ReadStream(f, 674, Stream{0, 137}); err != nil {
		t.Errorf("the header stream: %v", err)
	}
}
