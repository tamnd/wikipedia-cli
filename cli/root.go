// Package cli builds the wiki command tree on top of the wiki library.
package cli

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/tamnd/wikipedia-cli/wiki"
)

// Build metadata, set via -ldflags at release time.
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

// cmdOut is where plain (non-row) command output is written. A var so tests can
// redirect it.
var cmdOut = os.Stdout

// App carries the resolved configuration and shared clients for a command run.
type App struct {
	Cfg     wiki.Config
	Cache   *wiki.Cache
	Out     *Output
	client  *wiki.Client
	cerr    error
	Limit   int
	yes     bool
	noPager bool
}

// globalFlags holds the persistent flag values before they are folded into Cfg.
type globalFlags struct {
	lang     string
	project  string
	site     string
	output   string
	fields   string
	limit    int
	template string
	noHeader bool
	dataDir  string
	rate     time.Duration
	retries  int
	timeout  time.Duration
	noCache  bool
	color    string
	quiet    bool
	verbose  int
	yes      bool
	ua       string
	allowAny bool
}

// Root builds the root command and its whole subtree.
func Root() *cobra.Command {
	g := &globalFlags{}
	app := &App{}

	root := &cobra.Command{
		Use:   "wiki",
		Short: "A fast, friendly command line for Wikipedia",
		Long: `wiki is the fastest way to work with Wikipedia from your terminal.

Read an article as text or Markdown, search full text, pull summaries, list
links, categories, media and references, walk revisions and diffs, query
Wikidata with SPARQL, fetch pageview metrics, browse the daily and on-this-day
feeds, find articles near a coordinate, and stream the public XML dumps, all
from one binary with no credentials.

Quick start:
  wiki read "Alan Turing"              read an article in your pager
  wiki search "turing machine"         full-text search
  wiki get "Pi" --text | wc -w         the article text, for pipelines
  wiki summary "Quantum computing"     a one-paragraph summary`,
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			app.init(g)
			return nil
		},
	}

	pf := root.PersistentFlags()
	def := wiki.DefaultConfig()
	pf.StringVarP(&g.lang, "lang", "l", envOr("WIKI_LANG", def.Lang), "wiki language subdomain")
	pf.StringVar(&g.project, "project", envOr("WIKI_PROJECT", def.Project), "Wikimedia project")
	pf.StringVar(&g.site, "site", os.Getenv("WIKI_SITE"), "explicit wiki host (overrides lang/project)")
	pf.StringVarP(&g.output, "output", "o", "auto", "table|json|jsonl|csv|tsv|url|raw")
	pf.StringVar(&g.fields, "fields", "", "comma-separated columns to show")
	pf.IntVarP(&g.limit, "limit", "n", 0, "max results (0 = API default)")
	pf.StringVar(&g.template, "template", "", "Go text/template applied per row")
	pf.BoolVar(&g.noHeader, "no-header", false, "omit the header row in table/csv output")
	pf.StringVar(&g.dataDir, "data-dir", def.DataDir, "root data directory")
	pf.DurationVar(&g.rate, "rate", def.Delay, "minimum delay between requests")
	pf.IntVar(&g.retries, "retries", def.Retries, "retry attempts on 429/5xx")
	pf.DurationVar(&g.timeout, "timeout", def.Timeout, "per-request timeout")
	pf.BoolVar(&g.noCache, "no-cache", false, "bypass on-disk caches")
	pf.StringVar(&g.color, "color", "auto", "color output: auto|always|never")
	pf.BoolVarP(&g.quiet, "quiet", "q", false, "suppress progress output")
	pf.CountVarP(&g.verbose, "verbose", "v", "increase verbosity (repeatable)")
	pf.BoolVarP(&g.yes, "yes", "y", false, "assume yes to prompts")
	pf.StringVar(&g.ua, "ua", "", "override the User-Agent")
	pf.BoolVar(&g.allowAny, "allow-any-host", false, "allow non-Wikimedia --site hosts")

	root.AddCommand(
		newReadCmd(app),
		newGetCmd(app),
		newSummaryCmd(app),
		newOpenCmd(app),
		newSearchCmd(app),
		newSuggestCmd(app),
		newRandomCmd(app),
		newRelatedCmd(app),
		newLinksCmd(app),
		newBacklinksCmd(app),
		newCategoriesCmd(app),
		newCategoryCmd(app),
		newMediaCmd(app),
		newReferencesCmd(app),
		newCiteCmd(app),
		newLangsCmd(app),
		newInfoCmd(app),
		newRevisionsCmd(app),
		newDiffCmd(app),
		newFeaturedCmd(app),
		newOnThisDayCmd(app),
		newTopCmd(app),
		newPageviewsCmd(app),
		newGeoSearchCmd(app),
		newNearbyCmd(app),
		newEntityCmd(app),
		newSPARQLCmd(app),
		newSitesCmd(app),
		newStatsCmd(app),
		newConvertCmd(app),
		newDumpCmd(app),
		newConfigCmd(app),
		newCacheCmd(app),
		newVersionCmd(),
	)
	return root
}

func (a *App) init(g *globalFlags) {
	cfg := wiki.DefaultConfig()
	cfg.Lang = g.lang
	cfg.Project = g.project
	cfg.SiteHost = g.site
	if g.dataDir != "" {
		cfg.DataDir = g.dataDir
		cfg.CacheDir = g.dataDir + "/cache"
	}
	cfg.Delay = g.rate
	cfg.Retries = g.retries
	cfg.Timeout = g.timeout
	cfg.AllowAnyHost = g.allowAny
	if g.ua != "" {
		cfg.UserAgent = g.ua
	} else if env := os.Getenv("WIKI_USER_AGENT"); env != "" {
		cfg.UserAgent = env
	}

	a.Cfg = cfg
	a.Limit = g.limit
	a.yes = g.yes
	a.Cache = wiki.NewCache(cfg.CacheDir, !g.noCache)
	a.client, a.cerr = wiki.New(cfg, a.Cache)
	a.Out = newOutput(g)
}

// Client returns the resolved wiki client or the resolution error (bad
// project/host). Commands that need network access call this first.
func (a *App) Client() (*wiki.Client, error) {
	if a.cerr != nil {
		return nil, usageErr(a.cerr.Error())
	}
	return a.client, nil
}

// exit code helpers ---------------------------------------------------------

type exitCoder interface{ ExitCode() int }

type codedError struct {
	err  error
	code int
}

func (e codedError) Error() string { return e.err.Error() }
func (e codedError) ExitCode() int { return e.code }

func noResults(msg string) error { return codedError{fmt.Errorf("%s", msg), 3} }
func usageErr(msg string) error  { return codedError{fmt.Errorf("%s", msg), 2} }
func notFound(msg string) error  { return codedError{fmt.Errorf("%s", msg), 4} }

// wrapErr maps a library error to the right exit code.
func wrapErr(err error) error {
	if err == nil {
		return nil
	}
	if err == wiki.ErrNotFound || wiki.NotFound(err) {
		return notFound(err.Error())
	}
	return err
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
