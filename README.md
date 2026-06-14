# wiki

[![CI](https://github.com/tamnd/wikipedia-cli/actions/workflows/ci.yml/badge.svg)](https://github.com/tamnd/wikipedia-cli/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/tamnd/wikipedia-cli)](https://github.com/tamnd/wikipedia-cli/releases/latest)
[![Go Reference](https://pkg.go.dev/badge/github.com/tamnd/wikipedia-cli.svg)](https://pkg.go.dev/github.com/tamnd/wikipedia-cli)
[![Go Report Card](https://goreportcard.com/badge/github.com/tamnd/wikipedia-cli)](https://goreportcard.com/report/github.com/tamnd/wikipedia-cli)
[![License](https://img.shields.io/github/license/tamnd/wikipedia-cli)](./LICENSE)

A command line for Wikipedia. `wiki` reads public Wikipedia and Wikimedia data
and prints clean, pipeable records. One pure-Go binary, no API key, no login.

[Install](#install) • [Commands](#commands) • [Usage](#usage) • [Resource URIs](#use-it-as-a-resource-uri-driver)

![wiki searching and reading Wikipedia from the command line](docs/static/demo.gif)

It talks to the public Wikimedia APIs over plain HTTPS. Every request is paced,
retried on transient failures, and sent with an honest User-Agent. Responses are
cached on disk so re-running a command is instant and easy on the servers.

`wiki` is an independent tool. It is not affiliated with or endorsed by the
Wikimedia Foundation. Article content is licensed by its authors under
[CC BY-SA 4.0](https://creativecommons.org/licenses/by-sa/4.0/); please respect
the [Wikimedia API etiquette](https://www.mediawiki.org/wiki/API:Etiquette) and
attribute your sources.

## Install

```bash
go install github.com/tamnd/wikipedia-cli/cmd/wiki@latest
```

Or grab a prebuilt binary from the [releases](https://github.com/tamnd/wikipedia-cli/releases),
or run the container image:

```bash
docker run --rm ghcr.io/tamnd/wiki:latest read "Alan Turing"
```

Shell completion is built in: `wiki completion bash|zsh|fish|powershell`.

## Commands

| Command | Reads |
| --- | --- |
| `wiki read <title>` | an article as paged text, or `--markdown`, `--html`, `--wikitext` |
| `wiki get <title \| ->` | the same for pipelines; reads titles on stdin |
| `wiki summary <title>` | a one-paragraph extract |
| `wiki open <title>` | open the article in the browser, or `--print` its URL |
| `wiki search <query>` | full-text search; CirrusSearch operators pass through |
| `wiki suggest <prefix>` | prefix autocomplete |
| `wiki random` | one or more random articles |
| `wiki related <title>` | pages a reader is likely to want next |
| `wiki links <title>` | internal links; `--external` for URLs |
| `wiki backlinks <title>` | pages that link here |
| `wiki categories <title>` | categories the page belongs to |
| `wiki category <name>` | members of a category; `--type page\|subcat\|file` |
| `wiki media <title>` | files used on a page; `--download` to save them |
| `wiki references <title>` | external sources cited |
| `wiki cite <title>` | a formatted citation; `--format bibtex\|ris\|mla\|apa` |
| `wiki langs <title>` | the same article in other languages |
| `wiki info <title>` | page metadata |
| `wiki revisions <title>` | revision history, newest first; `--user` |
| `wiki diff <from> [to]` | unified diff between revisions |
| `wiki featured [date]` | the daily featured content |
| `wiki onthisday [date]` | historical events; `--type all\|births\|deaths\|holidays` |
| `wiki top [date]` | most-viewed articles for a day or month |
| `wiki pageviews <title>` | per-article pageview time series |
| `wiki geosearch <lat,lon>` | articles near a coordinate; `--radius` |
| `wiki nearby <title>` | articles near another article; `--radius` |
| `wiki entity <id \| title>` | a Wikidata entity; `--props`, `--lang` |
| `wiki sparql <query \| @file \| ->` | SPARQL against the Wikidata Query Service |
| `wiki dump list` | files of a dump; `--wiki`, `--date` |
| `wiki dump download <file>` | download with resume and sha1 verify |
| `wiki dump pages <file>` | stream-parse a pages-articles dump |
| `wiki dump grep <pattern> <file>` | pages matching a regexp |
| `wiki dump export [file]` | convert a dump to a Markdown or text corpus |
| `wiki sites` | known Wikimedia projects and example hosts |
| `wiki stats` | statistics for the selected wiki |
| `wiki convert <file \| ->` | convert HTML or wikitext offline |

Full reference and guides live at [wikipedia-cli.tamnd.com](https://wikipedia-cli.tamnd.com).

## Usage

```bash
wiki read "Alan Turing"
wiki search "turing machine"
wiki summary "Quantum computing"
wiki info "Mars"
wiki entity Q937 --props P569,P570       # Einstein's birth and death dates
wiki sparql 'SELECT ?c ?p WHERE { ?c wdt:P31 wd:Q515; wdt:P1082 ?p } ORDER BY DESC(?p) LIMIT 5'
wiki geosearch 51.5074,-0.1278 --radius 1000
wiki dump list --wiki dewiki
```

Records come out as a list (the default on a terminal), table, markdown, JSON,
JSONL, CSV, TSV, url, or raw. The output is colored on a terminal and a progress
spinner marks slow reads; pipe it and the bytes are plain:

```bash
wiki info "Mars" -o table
wiki search "physics" -n 10 --fields title,description -o csv
wiki links "Alan Turing" -o url | head
wiki revisions "Climate change" --user Jimbo -o jsonl
wiki search "quantum" -n 5 -o jsonl | jq .title
```

Switch wikis with `-l/--lang`, `--project`, or `--site`. A pasted URL picks the
right wiki automatically:

```bash
wiki read "Berlin" -l de                        # German Wikipedia
wiki search "café" --project wiktionary -l fr   # French Wiktionary
wiki read https://de.wikipedia.org/wiki/Berlin  # URL auto-selects
```

### Global flags

```
-o, --output      list|table|markdown|json|jsonl|csv|tsv|url|raw   (auto: list on a TTY, jsonl when piped)
    --fields      comma-separated columns to include
    --no-header   omit the heading in list, or the header row in csv/tsv/markdown
    --template    Go text/template applied per record
-n, --limit       max records (0 = command default)
-l, --lang        wiki language subdomain (default en)
    --project     Wikimedia project (default wikipedia)
    --site        explicit wiki host; overrides --lang and --project
-q, --quiet       suppress the progress spinner
    --color       auto|always|never  (color on a terminal, off when piped)
    --rate        min spacing between requests (default 150ms)
    --timeout     per-request timeout (default 1m)
    --retries     retry attempts on 429/5xx (default 4)
    --no-cache    bypass the on-disk response cache
    --ua          override the User-Agent
```

## Exit codes

```
0  success, at least one record
1  error
2  usage error
3  no results (a valid empty response)
4  not found (the article or entity does not exist)
```

## Use it as a resource-URI driver

The `wiki` package ships a driver that makes Wikipedia pages addressable as
resource URIs, the way a database driver registers with `database/sql`. A host
program such as [ant](https://github.com/tamnd/ant) blank-imports the package:

```go
import _ "github.com/tamnd/wikipedia-cli/wiki"
```

Then `ant` (or any program that links the package) dereferences `wikipedia://`
URIs:

```bash
ant get wikipedia://page/Alan_Turing       # the article summary
ant cat wikipedia://page/Alan_Turing       # just the extract text
ant ls  wikipedia://page/Alan_Turing       # the articles it links to
ant ls  wikipedia://category/Computability # the articles in a category
ant url wikipedia://page/Alan_Turing       # back to the live URL
```

Every listed link and category member is itself a `wikipedia://page/` URI, so a
host can walk the graph. See the [driver guide](https://wikipedia-cli.tamnd.com/guides/resource-uris/)
for the record shapes and the full URI grammar.

## Development

```
cmd/wiki/    thin main entry point
cli/         cobra commands, output rendering, and the progress spinner
wiki/        HTTP client, API calls, models, and the wikipedia:// driver
docs/        documentation site (Hugo, tago-doks theme)
```

```bash
make build   # ./bin/wiki
make test    # go test ./...
make vet     # go vet ./...
```

Requires Go 1.26+.

## Releasing

Push a version tag and GitHub Actions runs GoReleaser, which builds the
archives, Linux packages, the multi-arch GHCR image, checksums, SBOMs, and a
cosign signature:

```bash
git tag -a v0.3.0 -m "v0.3.0"
git push --tags
```

The image tag carries no `v` prefix (`ghcr.io/tamnd/wiki:0.3.0`).

## License

Apache-2.0. See [LICENSE](LICENSE).
