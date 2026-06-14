package wiki

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/tamnd/any-cli/kit"
	"github.com/tamnd/any-cli/kit/errs"
)

// domain.go exposes Wikipedia as a kit Domain: a driver that a multi-domain host
// (ant) enables with a single blank import,
//
//	import _ "github.com/tamnd/wikipedia-cli/wiki"
//
// exactly as a database/sql program enables a driver with `import _
// "github.com/lib/pq"`. The init below registers it; the host then dereferences
// wikipedia:// URIs by routing to the operations Register installs. The standalone
// wiki binary does not use any of this, so the CLI is unchanged.
//
// The driver speaks one wiki: English Wikipedia (en.wikipedia.org). The standalone
// binary still reaches every project and language through its own flags; a host
// addresses the default encyclopedia, where a bare title is unambiguous.
func init() { kit.Register(Domain{}) }

// Domain is the Wikipedia driver. It carries no state; the per-run client is built
// by the factory Register hands kit.
type Domain struct{}

// Info describes the scheme, the hostnames a pasted link is matched against, and
// the identity a host reuses for help and version.
func (Domain) Info() kit.DomainInfo {
	return kit.DomainInfo{
		Scheme:  "wikipedia",
		Aliases: []string{"wiki"},
		Hosts:   []string{"en.wikipedia.org", "wikipedia.org"},
		Identity: kit.Identity{
			Binary: "wiki",
			Short:  "Read Wikipedia articles and categories",
			Site:   "en.wikipedia.org",
			Repo:   "https://github.com/tamnd/wikipedia-cli",
		},
	}
}

// Register installs the client factory and every Wikipedia operation onto app. A
// resolver op (Single) names its own record type and answers `ant get`; a List op
// enumerates a parent resource's members and answers `ant ls`.
func (Domain) Register(app *kit.App) {
	app.SetClient(newClient)

	// Resolver ops: one record per id, the home of `ant get wikipedia://<type>/<id>`.
	kit.Handle(app, kit.OpMeta{Name: "page", Group: "read", Single: true,
		Summary: "Fetch an article summary by title or URL", URIType: "page", Resolver: true,
		Args: []kit.Arg{{Name: "title", Help: "article title or wiki URL"}}}, getPage)
	kit.Handle(app, kit.OpMeta{Name: "category", Group: "read", Single: true,
		Summary: "Fetch a category page's metadata", URIType: "category", Resolver: true,
		Args: []kit.Arg{{Name: "name", Help: "category name (without the Category: prefix) or URL"}}}, getCategory)

	// List ops: members of a parent resource, the home of `ant ls`. Both emit page
	// summaries so every listed member is itself an addressable wikipedia://page/.
	kit.Handle(app, kit.OpMeta{Name: "links", Group: "read", List: true,
		Summary: "List the articles a page links to", URIType: "page",
		Args: []kit.Arg{{Name: "title", Help: "article title or wiki URL"}}}, listPageLinks)
	kit.Handle(app, kit.OpMeta{Name: "members", Group: "read", List: true,
		Summary: "List the articles in a category", URIType: "category",
		Args: []kit.Arg{{Name: "name", Help: "category name or URL"}}}, listCategoryMembers)

	// Search is not URI-addressable on its own, but each hit names a page, so it
	// rounds out the surface a host can drive and mints as wikipedia://page/.
	kit.Handle(app, kit.OpMeta{Name: "search", Group: "read", URIType: "page",
		Summary: "Search articles by text",
		Args:    []kit.Arg{{Name: "query", Help: "search terms", Variadic: true}}}, search)
}

// newClient builds the Wikipedia client from the host-resolved config, reusing the
// same data dir the standalone binary uses so the page cache is shared. It pins the
// default English Wikipedia; the standalone binary is the place to pick another
// project or language.
func newClient(_ context.Context, cfg kit.Config) (any, error) {
	wcfg := DefaultConfig()
	if cfg.DataDir != "" {
		wcfg.DataDir = cfg.DataDir
	}
	if cfg.CacheDir != "" {
		wcfg.CacheDir = cfg.CacheDir
	}
	if cfg.UserAgent != "" {
		wcfg.UserAgent = cfg.UserAgent
	}
	if cfg.Timeout > 0 {
		wcfg.Timeout = cfg.Timeout
	}
	cache := NewCache(wcfg.CacheDir, !cfg.NoCache)
	return New(wcfg, cache)
}

// categoryPage is the addressable record for a Wikipedia category. A category is
// a page in the Category: namespace; the URI keys it by its bare name, so the
// record carries the bare Name as its id and the full Category: title alongside.
type categoryPage struct {
	Name    string `json:"name" kit:"id"`
	Title   string `json:"title"`
	Pageid  int    `json:"pageid,omitempty"`
	Length  int    `json:"length,omitempty"`
	Touched string `json:"touched,omitempty"`
	URL     string `json:"url"`
}

// --- inputs ---

type titleRef struct {
	Ref    string  `kit:"arg" help:"article title or wiki URL"`
	Client *Client `kit:"inject"`
}

type listRef struct {
	Ref    string  `kit:"arg" help:"title, category, or wiki URL"`
	Limit  int     `kit:"flag,inherit" help:"max results"`
	Client *Client `kit:"inject"`
}

type queryRef struct {
	Query  []string `kit:"arg,variadic" help:"search terms"`
	Limit  int      `kit:"flag,inherit" help:"max results"`
	Client *Client  `kit:"inject"`
}

// --- handlers ---

func getPage(ctx context.Context, in titleRef, emit func(*Summary) error) error {
	s, err := in.Client.GetSummary(ctx, idToTitle(in.Ref))
	if err != nil {
		return mapErr(err)
	}
	return emit(s)
}

func getCategory(ctx context.Context, in titleRef, emit func(*categoryPage) error) error {
	name := categoryName(in.Ref)
	info, err := in.Client.Info(ctx, "Category:"+name)
	if err != nil {
		return mapErr(err)
	}
	return emit(&categoryPage{
		Name:    name,
		Title:   info.Title,
		Pageid:  info.Pageid,
		Length:  info.Length,
		Touched: info.Touched,
		URL:     info.URL,
	})
}

func listPageLinks(ctx context.Context, in listRef, emit func(*Summary) error) error {
	links, err := in.Client.Links(ctx, idToTitle(in.Ref), 0, in.Limit)
	if err != nil {
		return mapErr(err)
	}
	for _, l := range links {
		if l.Title == "" {
			continue
		}
		if err := emit(stub(l.Title, l.URL)); err != nil {
			return err
		}
	}
	return nil
}

func listCategoryMembers(ctx context.Context, in listRef, emit func(*Summary) error) error {
	members, err := in.Client.CategoryMembers(ctx, categoryName(in.Ref), "page", in.Limit)
	if err != nil {
		return mapErr(err)
	}
	for _, m := range members {
		if m.Title == "" {
			continue
		}
		if err := emit(stub(m.Title, m.URL)); err != nil {
			return err
		}
	}
	return nil
}

func search(ctx context.Context, in queryRef, emit func(SearchResult) error) error {
	res, err := in.Client.Search(ctx, strings.Join(in.Query, " "), in.Limit)
	if err != nil {
		return mapErr(err)
	}
	for _, r := range res {
		if err := emit(r); err != nil {
			return err
		}
	}
	return nil
}

// --- Resolver: the URI-native string functions ---

// Classify turns any accepted input into the canonical (type, id), so `ant
// resolve` and `ant url` need no network. A Category: title becomes a category;
// everything else is a page keyed by its title.
func (Domain) Classify(input string) (uriType, id string, err error) {
	title, _ := ParseTarget(input)
	if title == "" {
		return "", "", errs.Usage("unrecognized Wikipedia reference: %q", input)
	}
	if name, ok := cutCategory(title); ok {
		return "category", name, nil
	}
	return "page", title, nil
}

// Locate is the inverse: the live page URL for a (type, id).
func (Domain) Locate(uriType, id string) (string, error) {
	site, err := DefaultConfig().Site()
	if err != nil {
		return "", err
	}
	switch uriType {
	case "page":
		return site.PageURL(idToTitle(id)), nil
	case "category":
		return site.PageURL(categoryTitle(id)), nil
	default:
		return "", errs.Usage("wikipedia has no resource type %q", uriType)
	}
}

// --- helpers ---

// idToTitle turns a URI id back into a human title: a path id uses the article's
// own title verbatim, so this only normalizes underscores the way the API does.
func idToTitle(id string) string { return NormalizeTitle(id) }

// categoryName extracts the bare category name (no "Category:" prefix) from any
// accepted reference, so the members API gets the form it wants.
func categoryName(s string) string {
	title, _ := ParseTarget(s)
	if name, ok := cutCategory(title); ok {
		return name
	}
	return title
}

// categoryTitle is the full "Category:Name" page title for a bare name or URL.
func categoryTitle(s string) string {
	return "Category:" + categoryName(s)
}

// cutCategory reports whether a title names a category and returns the bare name.
func cutCategory(title string) (name string, ok bool) {
	if rest, found := strings.CutPrefix(title, "Category:"); found {
		return strings.TrimSpace(rest), true
	}
	return "", false
}

// stub builds the lightweight page summary a list op emits for a linked or member
// article: enough to mint its wikipedia://page/ URI and follow it, no extra fetch.
func stub(title, url string) *Summary {
	return &Summary{Title: title, Type: "page", URL: url}
}

// mapErr converts a library error into the kit error kind that carries the right
// exit code, so a host renders the same not-found and rate-limited outcomes the
// standalone binary does.
func mapErr(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, ErrNotFound), NotFound(err):
		return errs.NotFound("%s", err.Error())
	default:
		var he *HTTPError
		if errors.As(err, &he) && he.Status == http.StatusTooManyRequests {
			return errs.RateLimited("%s", err.Error())
		}
		return err
	}
}
