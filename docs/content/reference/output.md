---
title: "Output formats"
description: "The shared output contract: list, table, markdown, json, jsonl, csv, tsv, url, raw, and Go templates, plus color, progress, column selection, and limits."
weight: 3
---

Every command renders through one output layer, so what you learn once applies
everywhere. Set the format with `-o/--output`.

## Formats

| Format | Best for |
| --- | --- |
| `list` | Reading at a terminal (the default there): each record is a short section, a heading then its fields below. |
| `table` | Scanning one column down many rows: a rounded, aligned grid that shrinks to your terminal width. |
| `markdown` | A GitHub-flavored pipe table to paste into an issue, a PR, or a README. |
| `jsonl` | Pipelines; one JSON object per line (the default in a pipe). |
| `json` | A single JSON array, for tools that want one document. |
| `csv` | Spreadsheets and data tools. |
| `tsv` | Tab-separated, for `cut` and friends. |
| `url` | Just the URL column, one per line. |
| `raw` | The first column, unadorned (also used by `--template`). |

The `list` and `table` views are two takes on the same rows. `list` puts one
record in front of you at a time, which reads better when a record carries many
fields; `table` lines many records up in a grid, which is better when you want
to compare one column across rows. Reach for `-o table` for the latter.

## auto

The default is `auto`: the readable `list` when stdout is a terminal, `jsonl`
when it is a pipe or file. So the same command reads well by eye and composes
well in a script with no extra flags:

```bash
wiki search "physics"            # a list on screen
wiki search "physics" | cat      # JSONL into the pipe
```

## Color

On a terminal the output is colored: a bold pink heading and aligned cyan keys
in `list`, a bold header and dimmed borders in `table`, and syntax-highlighted
`json`/`jsonl`. Color is on for an interactive terminal and off when piped, so
machine-read output stays plain. `--color` (`auto|always|never`) overrides it,
and the [`NO_COLOR`](https://no-color.org) convention is honored. With color
off, `list` emits literal GitHub-flavored markdown you can paste as-is.

## Progress

A read can wait on the network before it has anything to show. When the terminal
is interactive, wiki prints a small spinner to standard error while it waits and
clears it the moment the first result is ready. It only ever writes to standard
error, so a pipe like `wiki search physics | jq` and a redirect like
`wiki search physics > out.jsonl` never see it; standard output stays clean.
`--quiet` turns it off.

## Picking columns

`--fields` selects and orders columns by name. It applies to the column-shaped
formats (`list`, `table`, `markdown`, `csv`, `tsv`); the JSON formats always
carry the full record:

```bash
wiki search "volcano" --fields title,description -o csv
wiki revisions "Pi" --fields revid,user,timestamp
```

`--no-header` drops the header row from `csv`, `tsv`, and `markdown`, and the
section heading from `list`, which is handy when feeding another command:

```bash
wiki search "physics" -n 1 --fields title -o csv --no-header
```

## Limiting results

`-n/--limit` caps how many rows come back. `0` means the API default:

```bash
wiki search "cat" -n 5
wiki category Physics -n 100
```

## URLs

`-o url` prints just the URL of each row, perfect for piping into a fetcher or
`xargs`:

```bash
wiki links "Alan Turing" -o url | head
wiki search "turing" -o url | xargs -I{} echo open {}
```

## Templates

`--template` runs a Go [text/template](https://pkg.go.dev/text/template) over
each row's underlying value, for fully custom lines. It implies `raw` output:

```bash
wiki search "physics" --template '{{.Title}} => {{.URL}}'
wiki revisions "Pi" --template '{{.Timestamp}} {{.User}}'
```

The fields available are the JSON fields of each record; see the structured
output of a command (`-o json`) to discover them.

## Exit codes

Commands set a precise exit code so scripts can branch on the outcome. They are
listed on the [troubleshooting](/reference/troubleshooting/) page.
