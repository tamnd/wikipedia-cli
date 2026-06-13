package wiki

import (
	"fmt"
	"net/url"
	"strings"
)

// Site is a resolved target wiki: its host and the URL roots for the three
// per-wiki API surfaces. Build one with cfg.Site().
type Site struct {
	Host    string // e.g. "en.wikipedia.org"
	Project string // e.g. "wikipedia"
	Lang    string // e.g. "en" ("" for non-language projects)
}

// nonLangProjects map to a single host with no language subdomain.
var nonLangProjects = map[string]string{
	"wikidata": "www.wikidata.org",
	"commons":  "commons.wikimedia.org",
	"species":  "species.wikimedia.org",
	"meta":     "meta.wikimedia.org",
}

// langProjects map to a "{lang}.{domain}" host.
var langProjects = map[string]string{
	"wikipedia":   "wikipedia.org",
	"wiktionary":  "wiktionary.org",
	"wikibooks":   "wikibooks.org",
	"wikinews":    "wikinews.org",
	"wikiquote":   "wikiquote.org",
	"wikisource":  "wikisource.org",
	"wikiversity": "wikiversity.org",
	"wikivoyage":  "wikivoyage.org",
}

// Projects lists every project name Site understands, for discovery.
func Projects() []string {
	return []string{
		"wikipedia", "wiktionary", "wikibooks", "wikinews", "wikiquote",
		"wikisource", "wikiversity", "wikivoyage", "wikidata", "commons",
		"species", "meta",
	}
}

// Site resolves the configured Lang/Project/SiteHost into a concrete Site.
func (c Config) Site() (Site, error) {
	if c.SiteHost != "" {
		host := strings.TrimPrefix(strings.TrimPrefix(c.SiteHost, "https://"), "http://")
		host = strings.TrimSuffix(host, "/")
		if !c.AllowAnyHost && !isWikimediaHost(host) {
			return Site{}, fmt.Errorf("refusing non-Wikimedia host %q (use --allow-any-host to override)", host)
		}
		return Site{Host: host, Project: projectFromHost(host), Lang: langFromHost(host)}, nil
	}
	project := c.Project
	if project == "" {
		project = "wikipedia"
	}
	if host, ok := nonLangProjects[project]; ok {
		return Site{Host: host, Project: project}, nil
	}
	domain, ok := langProjects[project]
	if !ok {
		return Site{}, fmt.Errorf("unknown project %q (try one of: %s)", project, strings.Join(Projects(), ", "))
	}
	lang := c.Lang
	if lang == "" {
		lang = "en"
	}
	return Site{Host: lang + "." + domain, Project: project, Lang: lang}, nil
}

// APIURL returns the Action API endpoint with the given query parameters.
func (s Site) APIURL(params url.Values) string {
	return "https://" + s.Host + "/w/api.php?" + params.Encode()
}

// RestV1 returns a per-wiki REST v1 URL for the given path (no leading slash).
func (s Site) RestV1(path string) string {
	return "https://" + s.Host + "/api/rest_v1/" + path
}

// PageURL returns the canonical article URL for a title.
func (s Site) PageURL(title string) string {
	return "https://" + s.Host + "/wiki/" + titlePath(title)
}

// CoreProject returns the project slug used by the cross-wiki core API, or ""
// when the project is not addressable there.
func (s Site) CoreProject() (project, lang string, ok bool) {
	if s.Lang != "" && langProjects[s.Project] != "" {
		return s.Project, s.Lang, true
	}
	return "", "", false
}

// CoreURL builds an api.wikimedia.org core endpoint for this wiki.
func (s Site) CoreURL(path string) (string, bool) {
	project, lang, ok := s.CoreProject()
	if !ok {
		return "", false
	}
	return CoreAPIBase + "/" + project + "/" + lang + "/" + strings.TrimPrefix(path, "/"), true
}

// DumpWiki returns the dump database name for this site, e.g. "enwiki".
func (s Site) DumpWiki() string {
	if s.Lang == "" {
		return strings.ReplaceAll(strings.TrimSuffix(s.Host, ".wikimedia.org"), ".", "") + "wiki"
	}
	suffix := map[string]string{
		"wikipedia": "wiki", "wiktionary": "wiktionary", "wikibooks": "wikibooks",
		"wikinews": "wikinews", "wikiquote": "wikiquote", "wikisource": "wikisource",
		"wikiversity": "wikiversity", "wikivoyage": "wikivoyage",
	}[s.Project]
	if suffix == "" {
		suffix = "wiki"
	}
	return s.Lang + suffix
}

func projectFromHost(host string) string {
	for p, h := range nonLangProjects {
		if h == host {
			return p
		}
	}
	for p, d := range langProjects {
		if strings.HasSuffix(host, "."+d) {
			return p
		}
	}
	return ""
}

func langFromHost(host string) string {
	parts := strings.SplitN(host, ".", 2)
	if len(parts) == 2 && parts[0] != "www" && parts[0] != "commons" && parts[0] != "species" && parts[0] != "meta" {
		return parts[0]
	}
	return ""
}

// isWikimediaHost reports whether host belongs to the Wikimedia family. Used as
// an SSRF guard for --site and pasted URLs.
func isWikimediaHost(host string) bool {
	host = strings.ToLower(host)
	for _, suffix := range []string{
		".wikipedia.org", ".wikimedia.org", ".wiktionary.org", ".wikibooks.org",
		".wikinews.org", ".wikiquote.org", ".wikisource.org", ".wikiversity.org",
		".wikivoyage.org", ".wikidata.org", ".mediawiki.org",
	} {
		if strings.HasSuffix(host, suffix) {
			return true
		}
	}
	switch host {
	case "wikipedia.org", "wikimedia.org", "wikidata.org", "mediawiki.org":
		return true
	}
	return false
}
