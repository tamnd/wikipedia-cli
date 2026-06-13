package wiki

import "time"

// Per-surface cache TTLs. They balance freshness against politeness; see the
// spec's caching table. dayTTL is generous; callers that must see edits within
// minutes pass a short TTL or run with --no-cache.
const (
	ttlSiteInfo  = 7 * 24 * time.Hour
	ttlContent   = 24 * time.Hour
	ttlSearch    = time.Hour
	ttlHistory   = 10 * time.Minute
	ttlFeed      = 6 * time.Hour
	ttlPageviews = 24 * time.Hour
	ttlDump      = time.Hour
)
