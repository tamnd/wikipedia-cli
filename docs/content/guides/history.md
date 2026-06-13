---
title: "History and diffs"
description: "Walk a page's revision history and compare any two revisions as a unified diff."
weight: 4
---

## Revision history

List the edits to a page, newest first, with id, timestamp, author, byte size,
and edit summary:

```bash
wiki revisions "Go (programming language)" -n 20
```

Filter to one author:

```bash
wiki revisions "Climate change" --user Jimbo -o jsonl
```

The output carries everything you need to drill in: the `revid` of any row is
what `diff` and `read --rev` take.

## Diffs

Compare two revisions. Give one id and you get the diff against its parent:

```bash
wiki diff 123456789
```

Give two ids to compare them directly, or use `--to` with `prev`, `next`, or
`cur`:

```bash
wiki diff 123456789 123460000
wiki diff 123456789 --to next
```

For humans, the diff prints as familiar `+`/`-` lines, and `-o csv` gives one
row per line with its operation. With `-o json` you get the whole compare
result: both endpoints (their page ids, revision ids, namespaces and titles),
the raw HTML diff body, and the parsed lines, so nothing the API returned is
lost:

```bash
wiki diff 123456789 -o json
wiki diff 123456789 -o csv      # one row per changed line
```

## A small workflow

Find the most recent edit to a page and see what it changed:

```bash
rev=$(wiki revisions "Pi" -n 1 --fields revid -o csv --no-header)
wiki diff "$rev" --to prev
```
