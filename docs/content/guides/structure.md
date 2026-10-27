---
title: "Page structure"
description: "Explore the graph around a page: links, backlinks, categories and their members, media, references, interlanguage links, info, and citations."
weight: 3
---

Every article is a node in a graph. These commands walk the edges, one edge at a
time; [`discover`](/guides/discovering/) walks many at once.

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

## WikiProject assessments

A WikiProject tags the talk pages of the pages in its area, and gives each page a class and an importance. `wiki assess` lists the pages that one or more projects tag. It follows the list to the end, so the output of a large project has all of its pages. A page that two projects tag gives one row for each project.

```bash
wiki assess Mathematics -o tsv > mathematics.tsv
wiki assess Statistics "Philosophy/Logic" Cryptography -o jsonl
wiki assess Mathematics --ns 14 -n 20     # the categories that the project tags
```

The API gives the subject page of each tagged talk page. `--ns` keeps the pages of one namespace. The default is 0, the articles, and `--ns -1` keeps all of them. On 27 October 2026, `wiki assess Mathematics` gave 29,858 articles in about one minute.

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
first, streaming one row per object it reaches. The [Discovering](/guides/discovering/)
guide covers the walk in full: the edge presets, the depth and fanout bounds, and
how the output reads.

```bash
wiki discover "Alan Turing"              # the obvious neighbors, one hop out
```
