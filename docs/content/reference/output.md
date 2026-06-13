---
title: "Output formats"
description: "The shared output contract: table, json, jsonl, csv, tsv, url, raw, and Go templates, plus column selection and limits."
weight: 3
---

Every command renders through one output layer, so what you learn once applies
everywhere. Set the format with `-o/--output`.

## Formats

| Format | Best for |
| --- | --- |
| `table` | Reading at a terminal (the default there). |
| `jsonl` | Pipelines; one JSON object per line (the default in a pipe). |
| `json` | A single JSON array, for tools that want one document. |
| `csv` | Spreadsheets and data tools. |
| `tsv` | Tab-separated, for `cut` and friends. |
| `url` | Just the URL column, one per line. |
| `raw` | The first column, unadorned (also used by `--template`). |

## auto

The default is `auto`: a table when stdout is a terminal, JSONL when it is a
pipe or file. So the same command reads well by eye and composes well in a
script with no extra flags:

```bash
wiki search "physics"            # a table on screen
wiki search "physics" | cat      # JSONL into the pipe
```

## Picking columns

`--fields` selects and orders columns by name, in any format:

```bash
wiki search "volcano" --fields title,description -o csv
wiki revisions "Pi" --fields revid,user,timestamp
```

`--no-header` drops the header row from `table` and `csv`/`tsv`, which is handy
when feeding another command:

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
