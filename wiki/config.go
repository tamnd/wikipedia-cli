package wiki

import (
	"os"
	"path/filepath"
	"time"
)

// Build/runtime constants for the library.
const (
	// UserAgent identifies the client politely to the Wikimedia hosts. Wikimedia
	// asks every client to send a descriptive User-Agent with contact info; this
	// default carries the project URL. Override with Config.UserAgent.
	UserAgent = "wiki/1.0 (+https://github.com/tamnd/wikipedia-cli)"

	// WikidataSPARQL is the Wikidata Query Service endpoint.
	WikidataSPARQL = "https://query.wikidata.org/sparql"

	// MetricsBase is the analytics/pageviews REST root.
	MetricsBase = "https://wikimedia.org/api/rest_v1/metrics"

	// CoreAPIBase is the cross-wiki REST "core" root.
	CoreAPIBase = "https://api.wikimedia.org/core/v1"

	// DumpsBase is the published XML dumps host.
	DumpsBase = "https://dumps.wikimedia.org"
)

// Defaults for the polite HTTP client.
const (
	DefaultTimeout = 60 * time.Second
	DefaultRetries = 4
	DefaultDelay   = 150 * time.Millisecond
	// DefaultMaxlag is sent to the Action API so we back off when replica lag is
	// high. 5 seconds is the value the Wikimedia docs recommend for read clients.
	DefaultMaxlag = 5
)

// Config controls library behaviour. The zero value is not usable; call
// DefaultConfig and adjust.
type Config struct {
	Lang      string // language subdomain, e.g. "en"
	Project   string // "wikipedia", "wiktionary", "commons", "wikidata", ...
	SiteHost  string // explicit host; overrides Lang/Project when set
	DataDir   string
	CacheDir  string
	Timeout   time.Duration
	Delay     time.Duration
	Retries   int
	Maxlag    int
	UserAgent string
	// AllowAnyHost disables the Wikimedia-host allowlist (SSRF guard). Off by
	// default so a hostile "title"/URL cannot point the client elsewhere.
	AllowAnyHost bool
}

// DefaultConfig returns a Config rooted at ~/data/wiki, reading English
// Wikipedia, with the polite client defaults.
func DefaultConfig() Config {
	return Config{
		Lang:      "en",
		Project:   "wikipedia",
		DataDir:   dataDir(),
		CacheDir:  cacheDir(),
		Timeout:   DefaultTimeout,
		Delay:     DefaultDelay,
		Retries:   DefaultRetries,
		Maxlag:    DefaultMaxlag,
		UserAgent: UserAgent,
	}
}

// DownloadDir is where dump files and downloaded media land.
func (c Config) DownloadDir() string { return filepath.Join(c.DataDir, "downloads") }

// dataDir is the root for everything wiki writes: the cache and downloads. It
// defaults to ~/data/wiki so all state lives under one predictable tree.
// WIKI_DATA_DIR overrides it.
func dataDir() string {
	if d := os.Getenv("WIKI_DATA_DIR"); d != "" {
		return d
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "data", "wiki")
}

// cacheDir holds the small cached API responses. It sits under the data dir by
// default so the whole footprint is one tree; WIKI_CACHE_DIR overrides.
func cacheDir() string {
	if d := os.Getenv("WIKI_CACHE_DIR"); d != "" {
		return d
	}
	return filepath.Join(dataDir(), "cache")
}

// ConfigDir returns the directory holding the optional config file.
func ConfigDir() string {
	if d := os.Getenv("XDG_CONFIG_HOME"); d != "" {
		return filepath.Join(d, "wiki")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "wiki")
}
