---
title: "Page structure"
description: "Explore the graph around a page: links, backlinks, categories and their members, media, references, interlanguage links, info, and citations."
weight: 3
---

Every article is a node in a graph. These commands walk the edges.

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

Backlinks carry the page id of each linking page in their structured output.

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

In JSON, each category a page belongs to keeps its sort key, sort-key prefix,
timestamp and whether it is a hidden category; each category member keeps its
page id.

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

Each link keeps the language code, the autonym (the language's name in its own
script) and the langname (its name in this wiki's language), plus the URL.

## Page info

Metadata about a page: id, length, last touched, content model, language, and
whether it is a redirect:

```bash
wiki info "Alan Turing"
wiki info "Alan Turing" -o json
```

The JSON keeps the full info set: namespace, revision ids, watchers, the display
title and language variant titles, the canonical, full and edit URLs, and the
protection rules (type, level, expiry) as an array.

## Citations

Generate a citation for the article itself in your preferred format:

```bash
wiki cite "Alan Turing"                  # BibTeX by default
wiki cite "Alan Turing" --format ris
wiki cite "Alan Turing" --format apa
wiki cite "Alan Turing" --format mla
```
