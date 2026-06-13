---
title: "Dumps"
description: "List, download, and stream-parse the public Wikimedia XML dumps with resume, sha1 verification, and constant memory."
weight: 8
---

When you need the whole encyclopedia rather than one page, the public XML dumps
are the way. wiki lists them, downloads them safely, and streams them without
loading them into memory.

## List a dump

The files of a dump for a wiki and date. It defaults to the selected wiki's
most recent complete dump:

```bash
wiki dump list
wiki dump list --wiki enwiki --date latest
wiki dump list --wiki dewiki -o jsonl
```

Each row has the job, file name, size, sha1, and URL. The structured output
also carries the md5, the job's status (`done`, `waiting`, and so on) and the
time the job was last updated, so `-o json` is a faithful copy of the relevant
slice of `dumpstatus.json`. The `--wiki` name is a dump database name like
`enwiki` or `simplewiki`; without it, wiki derives one from `--project` and
`-l`.

## Download a file

Download by file name or by job. The transfer resumes a partial file and
verifies the sha1 from the dump status when one is published:

```bash
wiki dump download enwiki-latest-pages-articles1.xml-p1p41242.bz2
wiki dump download metahistory7zdump --out-dir ~/dumps
```

Progress prints to stderr and the final path prints to stdout, so it composes
in a script.

## Stream pages

Parse a local `pages-articles` dump into records in constant memory. bzip2 and
gzip are handled for you, so you can stream a compressed file directly. For
bzip2, wiki uses the parallel `lbzip2` or `pbzip2` binary when one is on PATH
(several times faster) and falls back to the standard library otherwise, so it
always works with no setup:

```bash
wiki dump pages enwiki-latest-pages-articles1.xml.bz2 --namespace 0 -n 100 -o jsonl
wiki dump pages simplewiki-latest-pages-articles.xml.bz2 --text -n 1 -o json
```

`--namespace` filters to one namespace (0 is articles), `-n` caps the count,
and `--text` includes the page body.

A `pages-articles` dump carries one current revision per page, but the same
parser handles a `pages-meta-history` dump, where each page carries its whole
revision list. The structured output keeps every revision in order, each with
its id, parent id, timestamp, contributor (a registered user's name and id, or
an anonymous edit's IP), minor flag, comment, content model and format, origin,
sha1, and byte length, along with the page's redirect target and edit/move
restrictions. The table view shows the page id, namespace, title, revision
count, and the latest revision's id and timestamp; `-o json` keeps all of it,
so you can reconstruct the full page history from the JSON:

```bash
wiki dump pages enwiki-latest-pages-meta-history1.xml.bz2 -n 1 --text -o json
wiki dump pages dump.xml.bz2 -n 1 -o json | jq '.revisions[].contributor'
```

## Grep a dump

Stream a dump and emit only the pages whose title or text matches a regular
expression:

```bash
wiki dump grep '(?i)quantum' simplewiki-latest-pages-articles.xml.bz2 -n 20
wiki dump grep '^List of' enwiki-latest-pages-articles1.xml.bz2 --title-only
```

Add `--text` to include the matching page bodies in the output.

## Export to Markdown

Turn a whole dump into a corpus of clean Markdown (or plain text), one file per
article. The wikitext is parsed and converted: headings, lists, bold/italic,
internal and external links, and fenced code blocks with their language are
preserved, while templates, infoboxes, references, tables, and File/Category
chrome are dropped. Redirects and non-article namespaces are skipped.

```bash
# Convert a local dump into a sharded Markdown tree.
wiki dump export simplewiki-latest-pages-articles.xml.bz2 --out-dir ./md

# One step: fetch the latest simple-English dump and export it.
wiki dump export --download --wiki simplewiki --out-dir ./md

# Sample 50 articles to stdout to eyeball quality.
wiki dump export simplewiki-latest-pages-articles.xml.bz2 -n 50 > sample.md

# Plain-text corpus instead of Markdown.
wiki dump export dump.xml.bz2 --format text --out-dir ./txt
```

With `--out-dir`, files land in hash-sharded subdirectories (`DIR/<aa>/<title>.md`)
so no single directory holds hundreds of thousands of files; without it, the
articles stream to stdout. `--min-bytes` skips stubs, `-N/--namespace` picks the
namespace (0 = articles), and `-n` bounds the run for sampling.

It is quick: on a fast laptop the whole Simple English Wikipedia (281,727
articles, 333 MB compressed) converts to Markdown in under a minute, in constant
memory. See the spec's benchmark section for details across machines.

## A complete flow

Find the latest simple-English dump file, download it, and pull the first
article that mentions a term:

```bash
file=$(wiki dump list --wiki simplewiki --fields name -o csv --no-header | grep pages-articles.xml.bz2 | head -1)
wiki dump download "$file" --wiki simplewiki --out-dir ~/dumps
wiki dump grep '(?i)volcano' ~/dumps/"$file" -n 1 --text -o json
```
