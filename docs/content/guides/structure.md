---
title: "Page structure"
description: "Explore the graph around a page: links, backlinks, categories and their members, media, references, interlanguage links, info, citations, and a breadth-first walk that chains them all."
weight: 3
---

Every article is a node in a graph. These commands walk the edges, one edge at a
time; [`discover`](#discover-walk-the-graph) walks many at once.

## Links and backlinks

The internal links on a page, or the external URLs it cites:

```bash
wiki links "Alan Turing"                 # internal wiki links
wiki links "Alan Turing" --external      # external URLs
wiki links "Alan Turing" -N 0            # only article-namespace links
```

What links *here*, the reverse direction:

```bash
wiki backlinks "Turing machine"
wiki backlinks "Turing machine" -o url | head
```

## Categories

The categories a page belongs to:

```bash
wiki categories "Alan Turing"
```

The members of a category (the `Category:` prefix is optional):

```bash
wiki category "British computer scientists" -n 100 -o url
wiki category Physics --type subcat       # subcategories only
wiki category Physics --type file         # files only
```

## Media

The files used on a page, with URL, MIME type, dimensions, size, and license:

```bash
wiki media "Alan Turing"
```

Download them all to a directory:

```bash
wiki media "Alan Turing" --download --out-dir imgs/
```

## References

The external sources cited by an article:

```bash
wiki references "Climate change"
wiki references "Climate change" -o url
```

## Interlanguage links

The same article in other languages:

```bash
wiki langs "Alan Turing"
wiki langs "Alan Turing" -o jsonl | head
```

## Page info

Metadata about a page: id, length, last touched, content model, language, and
whether it is a redirect:

```bash
wiki info "Alan Turing"
wiki info "Alan Turing" -o json
```

## Citations

Generate a citation for the article itself in your preferred format:

```bash
wiki cite "Alan Turing"                  # BibTeX by default
wiki cite "Alan Turing" --format ris
wiki cite "Alan Turing" --format apa
wiki cite "Alan Turing" --format mla
```

## Discover: walk the graph

Each command above answers one question about one page. `discover` chains them:
from a seed it follows the page's edges, then each neighbor's edges, breadth
first, streaming one row per object it reaches. Aliases: `walk`, `graph`.

```bash
wiki discover "Alan Turing"              # the obvious neighbors, one hop out
```

By default this follows a page's links and categories (and a category's members
and subcategories), one hop. The first row is the seed at depth 0; the rest are
its neighbors at depth 1, each tagged with the edge it was reached by:

```
depth  via         kind      title
0                  page      Alan Turing
1      links       page      Turing machine
1      categories  category  1912 births
```

`--follow` chooses which edges to walk. It takes a preset or a comma-separated
edge list:

| Preset    | Walks |
| --- | --- |
| `content` | a page's links and categories, a category's members and subcategories (the default) |
| `network` | a page's outgoing links and its backlinks |
| `cats`    | the category system: categories, members, and subcategories |
| `all`     | every edge |

The five edges are `links`, `backlinks`, `categories`, `members`, `subcats`.
Name one directly to follow just that edge:

```bash
wiki discover "Turing machine" --follow backlinks      # what links here, walked
wiki discover "Quantum computing" --follow network     # both link directions
wiki discover "Category:Turing Award laureates" --follow cats --depth 2
wiki discover "Pi" --follow links,categories           # mix edges and presets
```

`--depth` is how many hops to follow (default 1; `0` emits only the seeds).
`--fanout` caps neighbors per edge (default 25). The walk streams and stops after
`-n` nodes (default 500), so a bare walk always terminates.

`wiki` keeps no local database, so `discover` streams to stdout. To keep a walk,
pipe it:

```bash
wiki discover "Alan Turing" --depth 2 -o jsonl > turing-graph.jsonl

# the distinct article titles two hops out
wiki discover "Pi" --depth 2 -o jsonl \
  | jq -r 'select(.kind=="page").page.title' | sort -u
```

A seed that cannot be fetched fails the command, like any single read. A failure
deeper in the walk (one flaky edge) is a one-line note on stderr, and the walk
carries on with the rest of the graph.
