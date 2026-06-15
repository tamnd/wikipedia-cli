---
title: "CLI"
description: "Every wiki command, grouped by area, with its arguments and flags."
weight: 1
---

Run `wiki --help` for the top-level list and `wiki <command> --help` for any
command's own flags. This page is the full surface.

## Synopsis

```
wiki <command> <target> [flags]
```

The target is usually an article title: a bare word, a quoted phrase, an
underscore form, or a pasted Wikipedia URL (which selects the wiki for you).

## Reading

| Command | Description |
| --- | --- |
| `read <title>` | Read an article, paged plain text by default. Aliases: `show`, `cat`. |
| `get <title \| ->` | Fetch an article for pipelines; never paged; reads titles on stdin. |
| `summary <title>` | Print a one-paragraph summary. Alias: `tldr`. |
| `open <title>` | Open the article in your browser, or `--print` its URL. |

Shared render flags for `read` and `get`: `--text` (default), `--markdown`/`-m`,
`--html`, `--wikitext`, `--summary`, `--lead`, `--section N`, `--rev ID`.
`read` also takes `--no-pager`.

## Search and discovery

| Command | Description |
| --- | --- |
| `search <query>` | Full-text search; CirrusSearch operators pass through. Aliases: `s`, `find`. |
| `suggest <prefix>` | Prefix autocomplete. Aliases: `complete`, `opensearch`. |
| `random` | One or more random articles (`-n`, `-N` namespace). Alias: `rand`. |
| `related <title>` | Pages related to a title. |

## Page structure

| Command | Description |
| --- | --- |
| `links <title>` | Internal links, or `--external` URLs; `-N` namespace filter. |
| `backlinks <title>` | Pages that link here. Alias: `whatlinkshere`. |
| `categories <title>` | Categories the page belongs to. Alias: `cats`. |
| `category <name>` | Members of a category; `--type page\|subcat\|file`. |
| `media <title>` | Files used on a page; `--download`, `--out-dir`. Aliases: `images`, `files`. |
| `references <title>` | External sources cited. Alias: `refs`. |
| `cite <title>` | A citation for the article; `--format bibtex\|ris\|mla\|apa`. |
| `langs <title>` | The same article in other languages. Alias: `langlinks`. |
| `info <title>` | Page metadata. |
| `discover <seed>...` | Breadth-first walk of the graph from a page or category; `--follow content\|network\|cats\|all` or an edge list, `--depth`, `--fanout`. Aliases: `walk`, `graph`. |

## History

| Command | Description |
| --- | --- |
| `revisions <title>` | Revision history, newest first; `--user`. Aliases: `history`, `log`. |
| `diff <from> [to]` | Unified diff between revisions; `--to prev\|next\|cur\|<revid>`. |

## Feeds and metrics

| Command | Description |
| --- | --- |
| `featured [date]` | The daily featured feed. Aliases: `daily`, `tfa`. |
| `onthisday [date]` | Historical events; `--type all\|selected\|births\|deaths\|holidays\|events`. Alias: `otd`. |
| `top [date]` | Most-viewed articles for a day or month. |
| `pageviews <title>` | Pageview time series; `--from`, `--to`, `--days`, `--granularity`, `--access`, `--agent`. Aliases: `views`, `pv`. |

## Geo

| Command | Description |
| --- | --- |
| `geosearch <lat,lon>` | Articles near a coordinate; `--radius`. Alias: `geo`. |
| `nearby <title>` | Articles near another article; `--radius`. |

## Wikidata

| Command | Description |
| --- | --- |
| `entity <id \| title>` | A Wikidata entity; `--title`, `--props`, `--lang`. Alias: `wd`. |
| `sparql <query \| @file \| ->` | Run SPARQL against the Query Service. |

## Site and dumps

| Command | Description |
| --- | --- |
| `sites` | List known projects and example hosts. Aliases: `wikis`, `projects`. |
| `stats` | Statistics for the selected wiki. Alias: `siteinfo`. |
| `dump list` | Files of a dump; `--wiki`, `--date`. |
| `dump download <file\|job>` | Download with resume and sha1 verify; `--out-dir`. |
| `dump pages <file>` | Stream-parse a pages-articles dump; `--namespace`, `--text`. |
| `dump grep <pattern> <file>` | Emit pages matching a regexp; `--title-only`, `--text`. |
| `dump export [file]` | Convert a dump to Markdown/text; `--out-dir`, `--format`, `--download`, `--min-bytes`. |

## Utility

| Command | Description |
| --- | --- |
| `convert <file \| ->` | Convert HTML or wikitext to text, Markdown, or JSON, offline. |
| `config show` | Print the resolved configuration and paths. |
| `cache path\|info\|clear` | Inspect or clear the on-disk response cache. |
| `version` | Print the version; `--short`. |
| `completion <shell>` | Shell completion for bash, zsh, fish, or PowerShell. |

## Global flags

These persistent flags work on every command. See
[configuration](/reference/configuration/) for the full table and the matching
environment variables.

| Flag | Description |
| --- | --- |
| `-l, --lang` | Wiki language subdomain (default `en`). |
| `--project` | Wikimedia project (default `wikipedia`). |
| `--site` | Explicit wiki host; overrides `--lang`/`--project`. |
| `-o, --output` | `list\|table\|markdown\|json\|jsonl\|csv\|tsv\|url\|raw` (default `auto`: `list` on a terminal, `jsonl` when piped). |
| `--fields` | Comma-separated columns to keep, in order. |
| `-n, --limit` | Max results (0 = API default). |
| `--template` | Go text/template applied per row. |
| `--no-header` | Omit the header row (csv/tsv/markdown) or section heading (list). |
| `--data-dir` | Root data directory. |
| `--rate` | Minimum delay between requests. |
| `--retries` | Retry attempts on 429/5xx. |
| `--timeout` | Per-request timeout. |
| `--no-cache` | Bypass the on-disk cache. |
| `--color` | Color output: `auto\|always\|never` (honors `NO_COLOR`). |
| `-q, --quiet` | Suppress the progress spinner. |
| `-v, --verbose` | Increase verbosity (repeatable). |
| `-y, --yes` | Assume yes to prompts. |
| `--ua` | Override the User-Agent. |
| `--allow-any-host` | Allow non-Wikimedia `--site` hosts. |
