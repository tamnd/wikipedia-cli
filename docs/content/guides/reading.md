---
title: "Reading articles"
description: "Render any article as text, Markdown, HTML, wikitext, or a summary, by section or revision, for reading or for pipelines."
weight: 1
---

Reading is the heart of wiki. There are two commands: `read` for humans and
`get` for pipelines. They share the same rendering flags.

## read vs get

`wiki read` is built for a terminal. It renders clean plain text by default and
pages it through `$PAGER` (or `less`):

```bash
wiki read "Alan Turing"
```

`wiki get` is the scriptable sibling. It never pages, writes straight to
stdout, and reads titles on stdin with `-`:

```bash
wiki get "Alan Turing" --text | wc -w
wiki get "Pi" --html > pi.html
```

## Choosing the form

Both commands take the same flags to pick what you get back:

| Flag | What you get |
| --- | --- |
| `--text` | Clean plain text (the default) |
| `--markdown`, `-m` | Markdown with headings, links, and emphasis |
| `--html` | The server-rendered article HTML |
| `--wikitext` | The raw wikitext source |
| `--summary` | A one-paragraph summary |
| `--lead` | The lead section only, as text |
| `--section N` | Only section index `N` |
| `--rev ID` | A specific revision's content |

```bash
wiki read "Go (programming language)" --markdown
wiki read "Pi" --lead
wiki read "Pi" --section 2
wiki get "Cat" --rev 123456789 --wikitext
```

## Summaries

`wiki summary` is a shortcut for the one-paragraph extract, printed plain for
reading and as structured data in a pipe:

```bash
wiki summary "Quantum computing"
wiki summary "Quantum computing" -o json
```

The `-o json` output is the full REST summary: the display and normalized
titles, the namespace, the wikibase item, the thumbnail and original image, the
desktop and mobile URLs, the revision stamp, the HTML extract and the
coordinates, all preserved exactly as the API returned them.

## Targets: titles and URLs

The argument can be a bare title, a quoted title with spaces, an underscore
form, or a pasted Wikipedia URL. A URL selects the right wiki automatically:

```bash
wiki read "Alan Turing"
wiki read Alan_Turing
wiki read https://de.wikipedia.org/wiki/Berlin
```

## Other languages and projects

```bash
wiki read "Berlin" -l de                       # German Wikipedia
wiki read "café" --project wiktionary -l fr    # French Wiktionary
```

## Open in the browser

When you would rather read on the web:

```bash
wiki open "Alan Turing"           # open in your default browser
wiki open "Alan Turing" --print   # just print the URL
```

## Pipelines

Because `get` reads stdin and emits structured data, search results flow
straight in:

```bash
wiki search "turing" -n 5 -o jsonl | wiki get - --summary
```

See [output formats](/reference/output/) for everything you can do with `-o`.
