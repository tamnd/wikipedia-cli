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

Each row has the job, file name, size, sha1, and URL. The `--wiki` name is a
dump database name like `enwiki` or `simplewiki`; without it, wiki derives one
from `--project` and `-l`.

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
gzip are handled in pure Go, so you can stream a compressed file directly:

```bash
wiki dump pages enwiki-latest-pages-articles1.xml.bz2 --namespace 0 -n 100 -o jsonl
wiki dump pages simplewiki-latest-pages-articles.xml.bz2 --text -n 1 -o json
```

`--namespace` filters to one namespace (0 is articles), `-n` caps the count,
and `--text` includes the page body.

## Grep a dump

Stream a dump and emit only the pages whose title or text matches a regular
expression:

```bash
wiki dump grep '(?i)quantum' simplewiki-latest-pages-articles.xml.bz2 -n 20
wiki dump grep '^List of' enwiki-latest-pages-articles1.xml.bz2 --title-only
```

Add `--text` to include the matching page bodies in the output.

## A complete flow

Find the latest simple-English dump file, download it, and pull the first
article that mentions a term:

```bash
file=$(wiki dump list --wiki simplewiki --fields name -o csv --no-header | grep pages-articles.xml.bz2 | head -1)
wiki dump download "$file" --wiki simplewiki --out-dir ~/dumps
wiki dump grep '(?i)volcano' ~/dumps/"$file" -n 1 --text -o json
```
