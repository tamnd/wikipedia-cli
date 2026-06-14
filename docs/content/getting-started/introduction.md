---
title: "Introduction"
description: "What wiki is, the Wikimedia data it reads, and the mental model behind the command tree."
weight: 1
---

## What wiki is

wiki is a command-line tool for reading Wikipedia and the wider Wikimedia
world. It is one pure-Go binary with no runtime dependencies. You point it at a
title, a query, or a coordinate, and it renders the answer as text for you or
as data for a pipeline.

It is **read-only**. wiki never edits, creates, or deletes anything; it has no
login and stores no credentials. That keeps it safe to run anywhere and simple
to reason about.

## The data behind it

Wikimedia exposes its content through several public surfaces, and wiki picks
the right one for each job so you do not have to:

- **The MediaWiki Action API** (`/w/api.php`) on each wiki: search, links,
  categories, revisions, geosearch, site statistics, and more.
- **The REST API** (`/api/rest_v1/`) on each wiki: clean page summaries,
  related pages, and the daily featured and on-this-day feeds.
- **The Wikimedia metrics API**: per-article pageview time series and the
  most-viewed lists.
- **The Wikidata Query Service** (`query.wikidata.org/sparql`): SPARQL over the
  structured knowledge graph.
- **The published XML dumps** (`dumps.wikimedia.org`): the whole encyclopedia
  as downloadable, streamable files.

You never choose an endpoint. You choose a command.

## The mental model

Every command follows the same shape:

```
wiki <command> <target> [flags]
```

The **target** is usually an article title. A bare title, a title with spaces
in quotes, an underscore form, or even a pasted Wikipedia URL all work, and a
URL automatically selects the right wiki:

```bash
wiki read "Alan Turing"
wiki read Alan_Turing
wiki read https://de.wikipedia.org/wiki/Berlin
```

The **wiki** you talk to is English Wikipedia by default. Change the language
with `-l/--lang`, the project with `--project`, or give an explicit host with
`--site`:

```bash
wiki summary "Berlin" -l de                    # German Wikipedia
wiki read "café" --project wiktionary -l fr    # French Wiktionary
```

The **output** is a readable list when you are at a terminal and JSONL when you
pipe, so commands read well by eye and compose well in scripts. On a terminal
the list is colored and a slow read shows a small progress spinner. Switch the
format any time with `-o`:

```bash
wiki search "physics" -o table
wiki search "physics" -o json
wiki links "Pi" -o url | head
```

## Being a good citizen

wiki is polite by default. It throttles itself, retries failed requests with
backoff, honours the server's `Retry-After` and `maxlag` signals, and sends a
descriptive User-Agent. Responses are cached on disk so re-running a command is
instant and easy on the servers. See
[configuration](/reference/configuration/) to tune any of it.

Next: [install the binary](/getting-started/installation/).
