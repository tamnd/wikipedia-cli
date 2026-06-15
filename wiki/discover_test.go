package wiki

import (
	"context"
	"testing"
)

// fakeGraph is an in-memory grapher: the walk's bounds, ordering, dedup, and
// note-on-failure path are exercised over it with no network.
type fakeGraph struct {
	pages    map[string]*Summary         // title -> summary (for GetSummary)
	cats     map[string]*PageInfo        // bare name -> info (for Info)
	links    map[string][]Link           // title -> outgoing links
	backlnk  map[string][]Link           // title -> backlinks
	pagecats map[string][]Category       // title -> categories
	members  map[string][]CategoryMember // bare name -> page members
	subcats  map[string][]CategoryMember // bare name -> subcats
	fail     map[string]error            // "method:arg" -> error to return
}

func (f *fakeGraph) GetSummary(_ context.Context, title string) (*Summary, error) {
	if err := f.fail["summary:"+title]; err != nil {
		return nil, err
	}
	if s, ok := f.pages[title]; ok {
		return s, nil
	}
	return nil, ErrNotFound
}

func (f *fakeGraph) Info(_ context.Context, title string) (*PageInfo, error) {
	name, _ := cutCategory(title)
	if err := f.fail["info:"+name]; err != nil {
		return nil, err
	}
	if p, ok := f.cats[name]; ok {
		return p, nil
	}
	return nil, ErrNotFound
}

func (f *fakeGraph) Links(_ context.Context, title string, _, limit int) ([]Link, error) {
	if err := f.fail["links:"+title]; err != nil {
		return nil, err
	}
	return capLinks(f.links[title], limit), nil
}

func (f *fakeGraph) Backlinks(_ context.Context, title string, _, limit int) ([]Link, error) {
	if err := f.fail["backlinks:"+title]; err != nil {
		return nil, err
	}
	return capLinks(f.backlnk[title], limit), nil
}

func (f *fakeGraph) Categories(_ context.Context, title string, limit int) ([]Category, error) {
	if err := f.fail["categories:"+title]; err != nil {
		return nil, err
	}
	cats := f.pagecats[title]
	if limit > 0 && len(cats) > limit {
		cats = cats[:limit]
	}
	return cats, nil
}

func (f *fakeGraph) CategoryMembers(_ context.Context, name, memberType string, limit int) ([]CategoryMember, error) {
	if err := f.fail[memberType+":"+name]; err != nil {
		return nil, err
	}
	var src []CategoryMember
	switch memberType {
	case "subcat":
		src = f.subcats[name]
	default:
		src = f.members[name]
	}
	if limit > 0 && len(src) > limit {
		src = src[:limit]
	}
	return src, nil
}

func capLinks(ls []Link, limit int) []Link {
	if limit > 0 && len(ls) > limit {
		return ls[:limit]
	}
	return ls
}

// newFakeGraph builds a tiny corpus:
//
//	Alan Turing  --links-->  Turing machine, Cryptography
//	             --cats-->   Category:Computer scientists
//	             <-backlinks- Computer science
//	Turing machine --links--> Alan Turing (cycle), Cryptography
//	Computer science --links--> Alan Turing
//	Category:Computer scientists --members--> Alan Turing, Grace Hopper
//	                             --subcats--> Category:British computer scientists
//	Category:British computer scientists --members--> Alan Turing
func newFakeGraph() *fakeGraph {
	page := func(title string) *Summary {
		return &Summary{Title: title, Extract: title + " extract.", URL: "https://en.wikipedia.org/wiki/" + title}
	}
	link := func(title string) Link {
		return Link{Title: title, NS: 0, URL: "https://en.wikipedia.org/wiki/" + title}
	}
	member := func(title, typ string) CategoryMember {
		return CategoryMember{Title: title, Type: typ, URL: "https://en.wikipedia.org/wiki/" + title}
	}
	return &fakeGraph{
		pages: map[string]*Summary{
			"Alan Turing":      page("Alan Turing"),
			"Turing machine":   page("Turing machine"),
			"Cryptography":     page("Cryptography"),
			"Computer science": page("Computer science"),
			"Grace Hopper":     page("Grace Hopper"),
		},
		cats: map[string]*PageInfo{
			"Computer scientists":         {Title: "Category:Computer scientists", URL: "https://en.wikipedia.org/wiki/Category:Computer scientists"},
			"British computer scientists": {Title: "Category:British computer scientists", URL: "https://en.wikipedia.org/wiki/Category:British computer scientists"},
		},
		links: map[string][]Link{
			"Alan Turing":      {link("Turing machine"), link("Cryptography")},
			"Turing machine":   {link("Alan Turing"), link("Cryptography")},
			"Computer science": {link("Alan Turing")},
		},
		backlnk: map[string][]Link{
			"Alan Turing": {link("Computer science")},
		},
		pagecats: map[string][]Category{
			"Alan Turing": {{Title: "Category:Computer scientists", NS: 14, URL: "https://en.wikipedia.org/wiki/Category:Computer scientists"}},
		},
		members: map[string][]CategoryMember{
			"Computer scientists":         {member("Alan Turing", "page"), member("Grace Hopper", "page")},
			"British computer scientists": {member("Alan Turing", "page")},
		},
		subcats: map[string][]CategoryMember{
			"Computer scientists": {member("Category:British computer scientists", "subcat")},
		},
		fail: map[string]error{},
	}
}

func TestParseEdges(t *testing.T) {
	t.Run("empty is the default", func(t *testing.T) {
		got, err := ParseEdges("")
		if err != nil {
			t.Fatal(err)
		}
		if got.String() != DefaultEdges().String() {
			t.Errorf("empty = %q, want default %q", got, DefaultEdges())
		}
	})
	t.Run("preset expands", func(t *testing.T) {
		got, err := ParseEdges("cats")
		if err != nil {
			t.Fatal(err)
		}
		for _, e := range []Edge{EdgeCategories, EdgeMembers, EdgeSubcats} {
			if !got.Has(e) {
				t.Errorf("cats missing %s", e)
			}
		}
		if got.Has(EdgeLinks) || got.Has(EdgeBacklinks) {
			t.Errorf("cats should not include link edges: %q", got)
		}
	})
	t.Run("mixed list of edges and presets", func(t *testing.T) {
		// "links" is an edge and must NOT be shadowed by a preset of the same
		// name; the presets are deliberately disjoint from the edge names.
		got, err := ParseEdges("links,cats")
		if err != nil {
			t.Fatal(err)
		}
		if !got.Has(EdgeLinks) {
			t.Errorf("links edge missing: %q", got)
		}
		if !got.Has(EdgeMembers) {
			t.Errorf("cats preset not expanded: %q", got)
		}
		if got.Has(EdgeBacklinks) {
			t.Errorf("unexpected backlinks: %q", got)
		}
	})
	t.Run("unknown token errors", func(t *testing.T) {
		if _, err := ParseEdges("nope"); err == nil {
			t.Error("want error for unknown token")
		}
	})
}

func TestEdgeTargetAndSource(t *testing.T) {
	cases := []struct {
		e        Edge
		src, dst NodeKind
	}{
		{EdgeLinks, KindPage, KindPage},
		{EdgeBacklinks, KindPage, KindPage},
		{EdgeCategories, KindPage, KindCategory},
		{EdgeMembers, KindCategory, KindPage},
		{EdgeSubcats, KindCategory, KindCategory},
	}
	for _, c := range cases {
		if got := c.e.source(); got != c.src {
			t.Errorf("%s.source() = %s, want %s", c.e, got, c.src)
		}
		if got := c.e.Target(); got != c.dst {
			t.Errorf("%s.Target() = %s, want %s", c.e, got, c.dst)
		}
	}
}

func TestParseSeed(t *testing.T) {
	cases := []struct {
		in   string
		kind NodeKind
		ref  string
	}{
		{"Alan Turing", KindPage, "Alan Turing"},
		{"Alan_Turing", KindPage, "Alan Turing"},
		{"https://en.wikipedia.org/wiki/Earth", KindPage, "Earth"},
		{"Category:Physics", KindCategory, "Physics"},
		{"https://en.wikipedia.org/wiki/Category:Physics", KindCategory, "Physics"},
	}
	for _, c := range cases {
		s, err := ParseSeed(c.in)
		if err != nil {
			t.Errorf("ParseSeed(%q) error: %v", c.in, err)
			continue
		}
		if s.Kind != c.kind || s.Ref != c.ref {
			t.Errorf("ParseSeed(%q) = %s/%q, want %s/%q", c.in, s.Kind, s.Ref, c.kind, c.ref)
		}
	}
	if _, err := ParseSeed("   "); err == nil {
		t.Error("ParseSeed(blank) = nil error, want error")
	}
}

// walkAll runs a walk and collects the emitted nodes.
func walkAll(t *testing.T, g grapher, seeds []Seed, opts WalkOptions) []*Node {
	t.Helper()
	var got []*Node
	err := NewWalker(g).Walk(context.Background(), seeds, opts, func(n *Node) error {
		got = append(got, n)
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
	return got
}

func titlesByKind(nodes []*Node, kind NodeKind) []string {
	var out []string
	for _, n := range nodes {
		if n.Kind != kind {
			continue
		}
		out = append(out, n.Endpoint())
	}
	return out
}

func TestWalkContentFromPage(t *testing.T) {
	g := newFakeGraph()
	nodes := walkAll(t, g, []Seed{{Kind: KindPage, Ref: "Alan Turing"}}, WalkOptions{
		Depth: 1, Edges: DefaultEdges(),
	})
	// Seed first, at depth 0.
	if len(nodes) == 0 || nodes[0].Endpoint() != "Alan Turing" || nodes[0].Depth != 0 {
		t.Fatalf("first node = %+v, want Alan Turing at depth 0", nodes[0])
	}
	pages := titlesByKind(nodes, KindPage)
	if !contains(pages, "Turing machine") || !contains(pages, "Cryptography") {
		t.Errorf("pages = %v, want the linked articles", pages)
	}
	cats := titlesByKind(nodes, KindCategory)
	if !contains(cats, "Computer scientists") {
		t.Errorf("categories = %v, want Computer scientists", cats)
	}
	// The seed's content has no backlinks edge by default.
	for _, n := range nodes {
		if n.Via == EdgeBacklinks {
			t.Errorf("content should not follow backlinks, got %+v", n)
		}
	}
}

func TestWalkFromCategory(t *testing.T) {
	g := newFakeGraph()
	nodes := walkAll(t, g, []Seed{{Kind: KindCategory, Ref: "Computer scientists"}}, WalkOptions{
		Depth: 1, Edges: edgePresets["cats"],
	})
	if nodes[0].Kind != KindCategory || nodes[0].Endpoint() != "Computer scientists" {
		t.Fatalf("seed = %+v, want the category", nodes[0])
	}
	pages := titlesByKind(nodes, KindPage)
	if !contains(pages, "Alan Turing") || !contains(pages, "Grace Hopper") {
		t.Errorf("members = %v, want the category's pages", pages)
	}
	if !contains(titlesByKind(nodes, KindCategory), "British computer scientists") {
		t.Errorf("missing the subcategory")
	}
}

func TestWalkNetworkFollowsBacklinks(t *testing.T) {
	g := newFakeGraph()
	nodes := walkAll(t, g, []Seed{{Kind: KindPage, Ref: "Alan Turing"}}, WalkOptions{
		Depth: 1, Edges: edgePresets["network"],
	})
	var sawBacklink bool
	for _, n := range nodes {
		if n.Via == EdgeBacklinks && n.Endpoint() == "Computer science" {
			sawBacklink = true
		}
	}
	if !sawBacklink {
		t.Errorf("network preset did not follow backlinks: %v", titlesByKind(nodes, KindPage))
	}
}

func TestWalkDedup(t *testing.T) {
	g := newFakeGraph()
	nodes := walkAll(t, g, []Seed{{Kind: KindPage, Ref: "Alan Turing"}}, WalkOptions{
		Depth: 2, Edges: edgePresets["all"],
	})
	seen := map[string]int{}
	for _, n := range nodes {
		seen[string(n.Kind)+":"+n.Endpoint()]++
	}
	for k, c := range seen {
		if c != 1 {
			t.Errorf("node %s emitted %d times, want 1", k, c)
		}
	}
}

func TestWalkBudgetStops(t *testing.T) {
	g := newFakeGraph()
	nodes := walkAll(t, g, []Seed{{Kind: KindPage, Ref: "Alan Turing"}}, WalkOptions{
		Depth: 3, Max: 3, Edges: edgePresets["all"],
	})
	if len(nodes) != 3 {
		t.Errorf("emitted %d nodes, want exactly the budget of 3", len(nodes))
	}
}

func TestWalkFanoutCaps(t *testing.T) {
	g := newFakeGraph()
	nodes := walkAll(t, g, []Seed{{Kind: KindPage, Ref: "Alan Turing"}}, WalkOptions{
		Depth: 1, Fanout: 1, Edges: newEdgeSet(EdgeLinks),
	})
	// Seed plus at most one linked page (fanout 1 on the links edge).
	if len(nodes) != 2 {
		t.Errorf("emitted %d nodes, want 2 (seed + 1 link)", len(nodes))
	}
}

func TestWalkDepthZeroSeedsOnly(t *testing.T) {
	g := newFakeGraph()
	nodes := walkAll(t, g, []Seed{{Kind: KindPage, Ref: "Alan Turing"}}, WalkOptions{
		Depth: 0, Edges: edgePresets["all"],
	})
	if len(nodes) != 1 || nodes[0].Endpoint() != "Alan Turing" {
		t.Errorf("depth 0 = %v, want only the seed", titlesByKind(nodes, KindPage))
	}
}

func TestWalkSeedNotFoundFatal(t *testing.T) {
	g := newFakeGraph()
	err := NewWalker(g).Walk(context.Background(), []Seed{{Kind: KindPage, Ref: "Nonexistent"}}, WalkOptions{
		Depth: 1, Edges: DefaultEdges(),
	}, func(*Node) error { return nil })
	if err == nil {
		t.Error("a missing seed should fail the walk")
	}
}

func TestWalkDeeperErrorDegrades(t *testing.T) {
	g := newFakeGraph()
	// The links edge of the seed fails; the categories edge must still be
	// followed, and the failure should be a note, not a fatal error.
	g.fail["links:Alan Turing"] = ErrNotFound
	var notes int
	nodes := walkAll(t, g, []Seed{{Kind: KindPage, Ref: "Alan Turing"}}, WalkOptions{
		Depth: 1, Edges: DefaultEdges(),
		Note: func(string) { notes++ },
	})
	if !contains(titlesByKind(nodes, KindCategory), "Computer scientists") {
		t.Errorf("categories should survive a failed links edge: %v", nodes)
	}
	if contains(titlesByKind(nodes, KindPage), "Turing machine") {
		t.Errorf("the failed links edge should yield no link nodes")
	}
	if notes == 0 {
		t.Error("a failed edge should produce a note")
	}
}

func TestWalkMultipleSeeds(t *testing.T) {
	g := newFakeGraph()
	nodes := walkAll(t, g, []Seed{
		{Kind: KindPage, Ref: "Alan Turing"},
		{Kind: KindPage, Ref: "Grace Hopper"},
	}, WalkOptions{Depth: 0, Edges: DefaultEdges()})
	pages := titlesByKind(nodes, KindPage)
	if !contains(pages, "Alan Turing") || !contains(pages, "Grace Hopper") {
		t.Errorf("both seeds should be emitted: %v", pages)
	}
}

func contains(ss []string, want string) bool {
	for _, s := range ss {
		if s == want {
			return true
		}
	}
	return false
}
