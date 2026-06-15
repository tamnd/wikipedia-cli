---
title: "History and diffs"
description: "Walk a page's revision history and compare any two revisions as a unified diff."
weight: 5
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

For humans, the diff prints as familiar `+`/`-` lines. In a structured format
each line becomes a row with its operation, so you can post-process it:

```bash
wiki diff 123456789 -o jsonl
```

## A small workflow

Find the most recent edit to a page and see what it changed:

```bash
rev=$(wiki revisions "Pi" -n 1 --fields revid -o csv --no-header)
wiki diff "$rev" --to prev
```
