---
title: "Quick start"
description: "A guided first run: read an article, search, summarise, and compose a small pipeline."
weight: 3
---

This page takes you from a fresh install to a working pipeline in a few
minutes. Nothing here needs an account.

## Read an article

```bash
wiki read "Alan Turing"
```

The article opens in your pager as clean plain text. Press `q` to quit. Want
Markdown instead?

```bash
wiki read "Alan Turing" --markdown
```

Other forms are a flag away: `--html`, `--wikitext`, `--summary`, `--lead`
(the intro only), and `--section N`.

## Get a summary

```bash
wiki summary "Quantum computing"
```

One clean paragraph, the same one Wikipedia shows in its preview cards.

## Search

```bash
wiki search "turing machine"
```

You get a table of matching titles with a snippet. CirrusSearch operators pass
straight through:

```bash
wiki search "incategory:Physics quantum"
wiki search "intitle:learning insource:python"
```

## Switch wikis

```bash
wiki summary "Berlin" -l de                    # German Wikipedia
wiki read "café" --project wiktionary -l fr    # French Wiktionary
```

A pasted URL picks the wiki for you:

```bash
wiki read https://fr.wikipedia.org/wiki/Paris
```

## Compose a pipeline

Output is JSONL when piped, so commands chain naturally. Search, then fetch the
summary of each hit:

```bash
wiki search "climate change" -n 5 -o jsonl | wiki get - --summary
```

Just want URLs?

```bash
wiki links "Alan Turing" -o url | head
```

Pull a column with `--fields` and feed it to anything:

```bash
wiki search "physics" -n 10 --fields title -o csv
```

## Look something up in Wikidata

```bash
wiki entity Q937 --props P569,P570        # Einstein's birth and death dates
wiki sparql 'SELECT ?c ?p WHERE { ?c wdt:P31 wd:Q515; wdt:P1082 ?p } ORDER BY DESC(?p) LIMIT 5'
```

## Where to go next

- The [guides](/guides/) walk through each area in depth: reading, searching,
  page structure, history, feeds and metrics, geo, Wikidata, and dumps.
- The [CLI reference](/reference/cli/) lists every command and flag.
- [Output formats](/reference/output/) explains tables, JSON, CSV, URLs, and
  templates.
