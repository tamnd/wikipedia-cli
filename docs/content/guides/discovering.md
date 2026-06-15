---
title: "Discovering"
description: "Walk the graph of pages and categories breadth first, following links, backlinks, categories, members, and subcategories, one record per node."
weight: 4
---

Every structure command answers one question about one object: a page's links,
its categories, the members of a category. `discover` chains them. From a seed it
follows the object's links outward, and from each neighbor it follows theirs, hop
by hop, streaming one record per node as the node is reached.

```bash
wiki discover "Alan Turing"
```

A seed is anything `wiki` can resolve: an article title or URL, or a category
name or URL. Pass several to walk from all of them at once.

## The graph

There are two kinds of node, and five edges between them:

| Kind | What it is |
|---|---|
| `page` | an article |
| `category` | a category |

| Edge | From to | What it follows |
|---|---|---|
| `links` | page to page | the article's outgoing internal links |
| `backlinks` | page to page | what links here |
| `categories` | page to category | the categories the article belongs to |
| `members` | category to page | the articles in the category |
| `subcats` | category to category | the category's subcategories |

You rarely name edges one at a time. `--follow` takes a **preset**:

| Preset | Expands to | Walk shape |
|---|---|---|
| `content` *(default)* | `links` + `categories` + `members` + `subcats` | the obvious forward neighbors: a page's links and categories, a category's members and subcategories |
| `network` | `links` + `backlinks` | a page's outgoing links and what links back to it |
| `cats` | `categories` + `members` + `subcats` | the category tree above and below a page |
| `all` | every edge | the whole reachable neighborhood |

```bash
wiki discover "Alan Turing"                        # content (the default)
wiki discover "Alan Turing" --follow network       # links and backlinks
wiki discover "Category:Computer scientists" --follow cats --depth 2
wiki discover "Alan Turing" --follow all --depth 2
```

`--follow` also takes a single edge name, or a comma-separated mix of presets and
edges, so you can be exact:

```bash
wiki discover "Alan Turing" --follow backlinks     # only what-links-here
wiki discover "Alan Turing" --follow links,categories
```

Preset names and edge names are deliberately disjoint, so no `--follow` token is
ever ambiguous.

## Bounding the walk

Three independent limits keep a walk finite, so an unbounded `discover` always
terminates instead of spidering the whole wiki:

- `--depth` is how many hops to follow (default `1`; `0` emits only the seeds).
- `--fanout` caps neighbors per edge (default `25`; `0` means unlimited).
- `-n` caps the total nodes streamed (default `500`).

```bash
wiki discover "Alan Turing" --depth 2 --fanout 10 -n 200
```

A page links to hundreds of others, so a deep walk fans out fast. Raise `--depth`
one hop at a time and lean on `--fanout` and `-n` to keep a walk the size you want.

## Reading the output

Each row is a node tagged with how it was reached: how deep, by which edge, the
title and a one-line gloss. The first row is the seed at depth 0; the rest are
its neighbors, each tagged with the edge it was reached by:

```
depth  via         kind      title
0                  page      Alan Turing
1      links       page      Turing machine
1      categories  category  1912 births
```

The full typed record rides along for `-o json` and `-o jsonl`, and `-o url`
prints one link per node:

```bash
wiki discover "Alan Turing"             # the readable table
wiki discover "Alan Turing" -o jsonl    # one lossless object per line
wiki discover "Alan Turing" -o url      # one URL per node, to pipe onward
```

`wiki` keeps no local database, so `discover` streams to stdout. To keep a walk,
redirect it, and reshape it with ordinary tools:

```bash
wiki discover "Alan Turing" --depth 2 -o jsonl > turing-graph.jsonl

# the distinct article titles two hops out
wiki discover "Pi" --depth 2 -o jsonl \
  | jq -r 'select(.kind=="page").page.title' | sort -u
```

## When an edge is gated

Wikipedia's API is uniformly open: there are no scrape tiers and no per-IP
content gates, so every edge is reachable. The only runtime friction is rate
limiting, which the HTTP client absorbs with backoff. The walk still treats a
failure at the seed differently from one deeper in:

- A **seed** that cannot be fetched fails the walk, like any failed single read.
- An edge that fails **deeper** in the walk becomes a one-line note on stderr
  (`wiki: note: ...`) and the walk carries on with the other edges. `-q`
  silences the notes.

## Discover or the structure commands?

`discover` does not replace the focused reads, it composes them. Reach for the
single-purpose command when you want one slice of one page; reach for `discover`
when you want that slice and what it links to, hop after hop.

- [`links`](/guides/structure/), `backlinks`, `categories`, and the category
  reads each return one edge of one object, exactly and completely.
- `discover` follows those same edges outward across many objects, deduping as it
  goes, and streams the result as one graph.
