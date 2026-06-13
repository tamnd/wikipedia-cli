package wiki

import "testing"

func TestParseTarget(t *testing.T) {
	cases := []struct {
		in        string
		wantTitle string
		wantHost  string
	}{
		{"Alan Turing", "Alan Turing", ""},
		{"Alan_Turing", "Alan Turing", ""},
		{"  Cat  ", "Cat", ""},
		{"https://en.wikipedia.org/wiki/Alan_Turing", "Alan Turing", "en.wikipedia.org"},
		{"https://de.wikipedia.org/wiki/Berlin", "Berlin", "de.wikipedia.org"},
		{"https://en.wikipedia.org/w/index.php?title=Pi&oldid=1", "Pi", "en.wikipedia.org"},
		{"https://en.wikipedia.org/wiki/Go_(programming_language)", "Go (programming language)", "en.wikipedia.org"},
	}
	for _, tc := range cases {
		title, host := ParseTarget(tc.in)
		if title != tc.wantTitle || host != tc.wantHost {
			t.Errorf("ParseTarget(%q) = (%q,%q), want (%q,%q)", tc.in, title, host, tc.wantTitle, tc.wantHost)
		}
	}
}

func TestNormalizeTitle(t *testing.T) {
	if got := NormalizeTitle("Alan_Turing"); got != "Alan Turing" {
		t.Errorf("got %q", got)
	}
}

func TestStripHTML(t *testing.T) {
	in := `A <span class="x">quantum</span> &amp; <b>computer</b>`
	if got := stripHTML(in); got != "A quantum & computer" {
		t.Errorf("stripHTML = %q", got)
	}
}

func TestTruncate(t *testing.T) {
	if got := truncate("hello world", 5); got != "hell…" {
		t.Errorf("truncate = %q", got)
	}
	if got := truncate("hi", 5); got != "hi" {
		t.Errorf("truncate = %q", got)
	}
}

func TestTitlePath(t *testing.T) {
	if got := titlePath("Alan Turing"); got != "Alan_Turing" {
		t.Errorf("titlePath = %q", got)
	}
	if got := titlePath("AC/DC"); got != "AC/DC" {
		t.Errorf("titlePath = %q", got)
	}
}
