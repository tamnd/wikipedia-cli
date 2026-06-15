package wiki

import (
	"context"
	"fmt"
	"strings"
)

// discover.go is the breadth-first graph walker. The reads each answer one
// question about one object: a page's summary, the links on a page, the members
// of a category. The walker chains them. From a seed page or category it follows
// that object's links outward, hop by hop, emitting one node as it is reached.
//
// It talks to a small grapher interface, the exact subset of *Client it needs,
// never to *Client directly, so the BFS is tested hermetically over a fake
// in-memory graph with no network: the bounds, the ordering, the dedup, and the
// note-on-failure degradation are unit tests, not integration tests.
//
// Wikipedia's API is uniformly open: there are no scrape tiers (X) and no
// per-IP content gates (YouTube Restricted Mode), so every edge is reachable and
// nothing is dropped up front. The only runtime friction is rate-limiting, which
// the HTTP client already absorbs with backoff. A seed that cannot be fetched is
// fatal, matching a single read; a deeper failure becomes a one-line note and
// the walk continues on the rest of the graph.

// NodeKind is the kind of object a walk visits.
type NodeKind string

const (
	KindPage     NodeKind = "page"
	KindCategory NodeKind = "category"
)

// Edge is the public link vocabulary: what the user types in --follow and what a
// node reports as the edge it arrived by.
type Edge string

const (
	EdgeLinks      Edge = "links"      // page -> page (outgoing internal links, ns 0)
	EdgeBacklinks  Edge = "backlinks"  // page -> page (what links here, ns 0)
	EdgeCategories Edge = "categories" // page -> category (the page's categories)
	EdgeMembers    Edge = "members"    // category -> page (articles in the category)
	EdgeSubcats    Edge = "subcats"    // category -> category (subcategories)
)

// allEdges lists every edge in a stable display order.
var allEdges = []Edge{EdgeLinks, EdgeBacklinks, EdgeCategories, EdgeMembers, EdgeSubcats}

var knownEdges = func() map[Edge]bool {
	m := make(map[Edge]bool, len(allEdges))
	for _, e := range allEdges {
		m[e] = true
	}
	return m
}()

// source reports which node kind an edge departs from.
func (e Edge) source() NodeKind {
	switch e {
	case EdgeMembers, EdgeSubcats:
		return KindCategory
	default:
		return KindPage
	}
}

// Target reports which node kind an edge arrives at.
func (e Edge) Target() NodeKind {
	switch e {
	case EdgeCategories, EdgeSubcats:
		return KindCategory
	default:
		return KindPage
	}
}

// EdgeSet is a set of edges to follow.
type EdgeSet map[Edge]bool

func newEdgeSet(es ...Edge) EdgeSet {
	s := make(EdgeSet, len(es))
	for _, e := range es {
		s[e] = true
	}
	return s
}

// Has reports whether the set contains e.
func (s EdgeSet) Has(e Edge) bool { return s[e] }

// List returns the set's edges in the canonical display order.
func (s EdgeSet) List() []Edge {
	out := make([]Edge, 0, len(s))
	for _, e := range allEdges {
		if s[e] {
			out = append(out, e)
		}
	}
	return out
}

// String renders the set as a comma-separated list in canonical order.
func (s EdgeSet) String() string { return joinEdges(s.List()) }

func (s EdgeSet) clone() EdgeSet {
	out := make(EdgeSet, len(s))
	for e := range s {
		out[e] = true
	}
	return out
}

// edgePresets bundles the edges by intent. Preset names are kept disjoint from
// edge names, so no --follow token is ambiguous: ParseEdges resolves presets
// first, and a name that is both would shadow the same-named edge.
var edgePresets = map[string]EdgeSet{
	"content": newEdgeSet(EdgeLinks, EdgeCategories, EdgeMembers, EdgeSubcats),
	"network": newEdgeSet(EdgeLinks, EdgeBacklinks),
	"cats":    newEdgeSet(EdgeCategories, EdgeMembers, EdgeSubcats),
	"all":     newEdgeSet(allEdges...),
}

// presetNames lists the presets in display order for help and errors.
var presetNames = []string{"content", "network", "cats", "all"}

// DefaultEdges is the edge set used when --follow is empty: the content preset,
// the obvious forward neighbors of whatever was seeded.
func DefaultEdges() EdgeSet { return edgePresets["content"].clone() }

// EdgeHelp is the one-line catalogue of presets and edges, shared by the flag
// help and the parse error so the vocabulary lives in exactly one place.
func EdgeHelp() string {
	return "presets " + strings.Join(presetNames, "|") +
		"; edges " + edgeNames()
}

func edgeNames() string {
	names := make([]string, len(allEdges))
	for i, e := range allEdges {
		names[i] = string(e)
	}
	return strings.Join(names, "|")
}

// ParseEdges turns a --follow spec into an edge set. An empty spec is the
// default content preset; otherwise it is a comma-separated mix of preset names
// and edge names. An unknown token is an error that names the full catalogue.
func ParseEdges(spec string) (EdgeSet, error) {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return DefaultEdges(), nil
	}
	out := make(EdgeSet)
	for _, tok := range strings.Split(spec, ",") {
		tok = strings.TrimSpace(tok)
		if tok == "" {
			continue
		}
		if preset, ok := edgePresets[tok]; ok {
			for e := range preset {
				out[e] = true
			}
			continue
		}
		if knownEdges[Edge(tok)] {
			out[Edge(tok)] = true
			continue
		}
		return nil, fmt.Errorf("unknown edge or preset %q (%s)", tok, EdgeHelp())
	}
	if len(out) == 0 {
		return DefaultEdges(), nil
	}
	return out, nil
}

func joinEdges(es []Edge) string {
	parts := make([]string, len(es))
	for i, e := range es {
		parts[i] = string(e)
	}
	return strings.Join(parts, ",")
}

// CategoryNode is the category record a walk emits: the bare name, the full
// "Category:Name" title, and the live URL. A category reached through an edge is
// pre-built from the edge data, so it costs no extra fetch; a category seed is
// validated through Info.
type CategoryNode struct {
	Name  string `json:"name"`
	Title string `json:"title"`
	URL   string `json:"url"`
}

// Node is one object reached by a walk, tagged with how it was reached.
type Node struct {
	Kind     NodeKind      `json:"kind"`
	Depth    int           `json:"depth"`
	Via      Edge          `json:"via,omitempty"`
	Parent   string        `json:"parent,omitempty"`
	Page     *Summary      `json:"page,omitempty"`
	Category *CategoryNode `json:"category,omitempty"`
}

// Endpoint is the node's stable identity within a walk: a page's title or a
// category's bare name.
func (n *Node) Endpoint() string {
	switch n.Kind {
	case KindPage:
		if n.Page != nil {
			return n.Page.Title
		}
	case KindCategory:
		if n.Category != nil {
			return n.Category.Name
		}
	}
	return ""
}

// nodeKey collapses aliases so the same object reached two ways is visited once.
func nodeKey(kind NodeKind, ref string) string {
	switch kind {
	case KindCategory:
		return "c:" + NormalizeTitle(ref)
	default:
		return "p:" + NormalizeTitle(ref)
	}
}

// Seed is a starting point for a walk.
type Seed struct {
	Kind NodeKind
	Ref  string // page title, or bare category name
}

// ParseSeed classifies any accepted reference into a seed, reusing the domain's
// own Classify so discover reads a string exactly as get and ls do.
func ParseSeed(ref string) (Seed, error) {
	typ, id, err := Domain{}.Classify(ref)
	if err != nil {
		return Seed{}, err
	}
	switch typ {
	case "category":
		return Seed{Kind: KindCategory, Ref: id}, nil
	default:
		return Seed{Kind: KindPage, Ref: id}, nil
	}
}

// WalkOptions configures a walk.
type WalkOptions struct {
	Depth  int          // hops to follow from each seed (0 = seeds only)
	Max    int          // total node budget, the hard stop
	Fanout int          // max neighbors per edge (0 = bounded only by Max)
	Edges  EdgeSet      // which edges to follow
	Note   func(string) // surface a one-line advisory (never fatal)
}

// grapher is the exact subset of *Client the walker needs. *Client is the
// production grapher; a test supplies a fake in-memory graph.
type grapher interface {
	GetSummary(ctx context.Context, title string) (*Summary, error)
	Info(ctx context.Context, title string) (*PageInfo, error)
	Links(ctx context.Context, title string, namespace, limit int) ([]Link, error)
	Backlinks(ctx context.Context, title string, namespace, limit int) ([]Link, error)
	Categories(ctx context.Context, title string, limit int) ([]Category, error)
	CategoryMembers(ctx context.Context, name, memberType string, limit int) ([]CategoryMember, error)
}

var _ grapher = (*Client)(nil)

// Walker runs a breadth-first walk over a grapher.
type Walker struct{ g grapher }

// NewWalker builds a walker over any grapher (the client, or a test fake).
func NewWalker(g grapher) *Walker { return &Walker{g: g} }

// Walk runs a breadth-first walk from the seeds and calls emit for each node.
func (c *Client) Walk(ctx context.Context, seeds []Seed, opts WalkOptions, emit func(*Node) error) error {
	return NewWalker(c).Walk(ctx, seeds, opts, emit)
}

// frontier is a queued item: a reference plus how it was reached. A page or
// category reached through an edge carries its pre-built record and skips the
// per-pop fetch; a seed (or anything we only have a bare ref for) is hydrated
// when it is popped.
type frontier struct {
	kind   NodeKind
	ref    string
	depth  int
	via    Edge
	parent string
	page   *Summary
	cat    *CategoryNode
}

func (f frontier) key() string { return nodeKey(f.kind, f.ref) }

// Walk is the breadth-first traversal.
func (w *Walker) Walk(ctx context.Context, seeds []Seed, opts WalkOptions, emit func(*Node) error) error {
	if opts.Edges == nil {
		opts.Edges = DefaultEdges()
	}
	visited := make(map[string]bool)
	queue := make([]frontier, 0, len(seeds))
	for _, s := range seeds {
		queue = append(queue, frontier{kind: s.Kind, ref: s.Ref, depth: 0})
	}

	emitted := 0
	for len(queue) > 0 {
		if err := ctx.Err(); err != nil {
			return err
		}
		f := queue[0]
		queue = queue[1:]
		if visited[f.key()] {
			continue
		}
		visited[f.key()] = true

		node, err := w.hydrate(ctx, f)
		if err != nil {
			if f.depth == 0 {
				return err // a seed that cannot be fetched fails the walk
			}
			note(opts, err)
			continue
		}
		// Collapse the resolved identity too, so a redirect or a normalization
		// that lands on an already-seen node does not re-emit.
		visited[nodeKey(node.Kind, node.Endpoint())] = true

		if err := emit(node); err != nil {
			return err
		}
		emitted++
		if opts.Max > 0 && emitted >= opts.Max {
			return nil
		}
		if f.depth >= opts.Depth {
			continue
		}
		queue = append(queue, w.neighbors(ctx, node, opts)...)
	}
	return nil
}

// hydrate turns a frontier item into a node, fetching only what was not already
// carried in from the edge that produced it.
func (w *Walker) hydrate(ctx context.Context, f frontier) (*Node, error) {
	switch f.kind {
	case KindPage:
		if f.page != nil {
			return &Node{Kind: KindPage, Depth: f.depth, Via: f.via, Parent: f.parent, Page: f.page}, nil
		}
		s, err := w.g.GetSummary(ctx, f.ref)
		if err != nil {
			return nil, err
		}
		return &Node{Kind: KindPage, Depth: f.depth, Via: f.via, Parent: f.parent, Page: s}, nil
	case KindCategory:
		if f.cat != nil {
			return &Node{Kind: KindCategory, Depth: f.depth, Via: f.via, Parent: f.parent, Category: f.cat}, nil
		}
		info, err := w.g.Info(ctx, "Category:"+f.ref)
		if err != nil {
			return nil, err
		}
		return &Node{
			Kind: KindCategory, Depth: f.depth, Via: f.via, Parent: f.parent,
			Category: &CategoryNode{Name: f.ref, Title: info.Title, URL: info.URL},
		}, nil
	}
	return nil, fmt.Errorf("unknown node kind %q", f.kind)
}

// neighbors expands a node into the next frontier, honoring the edge set and the
// per-edge fanout. Every neighbor is pre-built from the list it came in on, so a
// hop costs the list calls, not a fetch per neighbor.
func (w *Walker) neighbors(ctx context.Context, node *Node, opts WalkOptions) []frontier {
	limit := opts.Fanout
	if limit <= 0 {
		limit = opts.Max
	}
	depth := node.Depth + 1
	parent := node.Endpoint()
	var out []frontier

	switch node.Kind {
	case KindPage:
		title := node.Page.Title
		if opts.Edges.Has(EdgeLinks) {
			links, err := w.g.Links(ctx, title, 0, limit)
			note(opts, err)
			for _, l := range pageLinks(links, limit) {
				out = append(out, frontier{kind: KindPage, ref: l.Title, depth: depth, via: EdgeLinks, parent: parent, page: stubSummary(l.Title, l.URL)})
			}
		}
		if opts.Edges.Has(EdgeBacklinks) {
			links, err := w.g.Backlinks(ctx, title, 0, limit)
			note(opts, err)
			for _, l := range pageLinks(links, limit) {
				out = append(out, frontier{kind: KindPage, ref: l.Title, depth: depth, via: EdgeBacklinks, parent: parent, page: stubSummary(l.Title, l.URL)})
			}
		}
		if opts.Edges.Has(EdgeCategories) {
			cats, err := w.g.Categories(ctx, title, limit)
			note(opts, err)
			for _, c := range cats {
				name, ok := cutCategory(c.Title)
				if !ok {
					continue
				}
				out = append(out, frontier{kind: KindCategory, ref: name, depth: depth, via: EdgeCategories, parent: parent, cat: &CategoryNode{Name: name, Title: c.Title, URL: c.URL}})
			}
		}
	case KindCategory:
		name := node.Category.Name
		if opts.Edges.Has(EdgeMembers) {
			members, err := w.g.CategoryMembers(ctx, name, "page", limit)
			note(opts, err)
			for _, m := range members {
				if m.Title == "" {
					continue
				}
				out = append(out, frontier{kind: KindPage, ref: m.Title, depth: depth, via: EdgeMembers, parent: parent, page: stubSummary(m.Title, m.URL)})
			}
		}
		if opts.Edges.Has(EdgeSubcats) {
			subs, err := w.g.CategoryMembers(ctx, name, "subcat", limit)
			note(opts, err)
			for _, m := range subs {
				bare, ok := cutCategory(m.Title)
				if !ok {
					continue
				}
				out = append(out, frontier{kind: KindCategory, ref: bare, depth: depth, via: EdgeSubcats, parent: parent, cat: &CategoryNode{Name: bare, Title: m.Title, URL: m.URL}})
			}
		}
	}
	return out
}

// pageLinks keeps only the article links with a title and applies the fanout, so
// a namespace stray or a blank entry never becomes a node.
func pageLinks(links []Link, limit int) []Link {
	out := links[:0:0]
	for _, l := range links {
		if l.Title == "" {
			continue
		}
		out = append(out, l)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out
}

// stubSummary is the lightweight page record a neighbor carries: enough to emit
// it and to expand it (every page expansion takes only a title), no extra fetch.
func stubSummary(title, url string) *Summary {
	return &Summary{Title: title, Type: "page", URL: url}
}

// note surfaces a non-fatal advisory through the option hook, ignoring a nil
// error so the call sites stay terse.
func note(opts WalkOptions, err error) {
	if err == nil || opts.Note == nil {
		return
	}
	opts.Note(err.Error())
}
