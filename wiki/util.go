package wiki

import (
	"net/url"
	"strconv"
	"strings"
)

// itoa is a tiny strconv.Itoa alias used by the renderer and feature methods.
func itoa(n int) string { return strconv.Itoa(n) }

// urlPathEscape percent-encodes a single path segment (e.g. a pageviews title).
func urlPathEscape(s string) string { return url.PathEscape(s) }

// NormalizeTitle turns a human-typed title into the canonical form the APIs use
// for display: underscores become spaces and surrounding whitespace is trimmed.
// The server's own normalization remains authoritative; this is for local use.
func NormalizeTitle(title string) string {
	t := strings.TrimSpace(strings.ReplaceAll(title, "_", " "))
	return t
}

// titlePath encodes a title for a /wiki/ or REST path: spaces become
// underscores and the result is percent-encoded (but '/' is kept readable).
func titlePath(title string) string {
	t := strings.ReplaceAll(strings.TrimSpace(title), " ", "_")
	// url.PathEscape escapes '/', which we want to keep for subpage titles.
	parts := strings.Split(t, "/")
	for i, p := range parts {
		parts[i] = url.PathEscape(p)
	}
	return strings.Join(parts, "/")
}

// ParseTarget accepts a bare title, a "Namespace:Title", or a full Wikipedia
// URL, and returns the title plus the host it was pasted from (empty when the
// input was not a URL). A leading/trailing whitespace is trimmed.
func ParseTarget(input string) (title, host string) {
	in := strings.TrimSpace(input)
	if strings.HasPrefix(in, "http://") || strings.HasPrefix(in, "https://") {
		if u, err := url.Parse(in); err == nil {
			host = u.Host
			// /wiki/Title  or  /w/index.php?title=Title
			if rest, ok := strings.CutPrefix(u.Path, "/wiki/"); ok {
				if dec, err := url.PathUnescape(rest); err == nil {
					rest = dec
				}
				return NormalizeTitle(rest), host
			}
			if t := u.Query().Get("title"); t != "" {
				return NormalizeTitle(t), host
			}
			return NormalizeTitle(strings.TrimPrefix(u.Path, "/")), host
		}
	}
	return NormalizeTitle(in), ""
}

// stripHTML removes HTML tags from a fragment (used for search snippets), and
// decodes the handful of entities the search API emits.
func stripHTML(s string) string {
	var b strings.Builder
	depth := false
	for _, r := range s {
		switch r {
		case '<':
			depth = true
		case '>':
			depth = false
		default:
			if !depth {
				b.WriteRune(r)
			}
		}
	}
	out := b.String()
	out = strings.ReplaceAll(out, "&quot;", "\"")
	out = strings.ReplaceAll(out, "&amp;", "&")
	out = strings.ReplaceAll(out, "&lt;", "<")
	out = strings.ReplaceAll(out, "&gt;", ">")
	out = strings.ReplaceAll(out, "&#39;", "'")
	out = strings.ReplaceAll(out, "&nbsp;", " ")
	return strings.Join(strings.Fields(out), " ")
}

// truncate shortens s to n runes, appending an ellipsis when it cuts.
func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}
