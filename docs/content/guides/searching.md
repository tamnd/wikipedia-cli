---
title: "Searching and discovering"
description: "Full-text search with CirrusSearch operators, prefix suggestions, random articles, and related pages."
weight: 2
---

## Full-text search

```bash
wiki search "turing machine"
```

You get a table of titles with a short description and snippet. In a pipe it is
JSONL, so it composes:

```bash
wiki search "climate change" -n 50 -o jsonl | wiki get - --summary
wiki search "physics" -o url | head
```

### CirrusSearch operators

Wikipedia's search operators pass straight through, so you can scope a query
precisely:

```bash
wiki search "incategory:Physics quantum"
wiki search "intitle:learning insource:python"
wiki search "morelike:Albert Einstein"
```

Cap the result count with `-n` and pick columns with `--fields`:

```bash
wiki search "volcano" -n 20 --fields title,description -o csv
```

The JSON and JSONL output carries everything the search API returns for a hit,
not just the columns shown in the table: the page id, the URL-safe `key`, the
`matched_title` when the match came from a redirect, and a `thumbnail` object
(url, mimetype, width, height) when the page has one.

```bash
wiki search "Alan Turing" -n 1 -o json
```

## Prefix suggestions

`suggest` is autocomplete: give a prefix, get the titles that start with it.
It is what shell completion uses under the hood.

```bash
wiki suggest "Quantu"
wiki suggest "New Yor" -n 5 -o url
```

## Random articles

```bash
wiki random                 # one random article
wiki random -n 5            # five of them
wiki random -N 14           # from a specific namespace
```

Random results are never cached, so each run is genuinely fresh.

## Related pages

Find articles the reader of one page is likely to want next:

```bash
wiki related "Alan Turing"
wiki related "Quantum computing" -o jsonl
```

Like search, `related` and `random` keep the page id, namespace and thumbnail in
their structured output.

## Putting it together

Discover, then read. Search for a topic, take the first hit, and read its lead:

```bash
wiki search "general relativity" -n 1 -o jsonl | wiki get - --lead
```
