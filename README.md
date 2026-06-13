# wiki

A fast, friendly command line for Wikipedia.

`wiki` is a single pure-Go binary that puts Wikipedia and the wider Wikimedia
world behind a tool that feels like `curl`. Read an article as clean text or
Markdown, search the full text, pull a one-paragraph summary, walk links,
categories, media and references, follow revisions and diffs, query Wikidata
with SPARQL, fetch pageview metrics, browse the daily and on-this-day feeds,
find articles near a coordinate, and stream the public XML dumps, all with no
account and nothing to pay for.

```bash
wiki read "Alan Turing"               # read the article in your pager
wiki search "turing machine"          # full-text search
wiki get "Pi" --text | wc -w          # the article text, for pipelines
wiki summary "Quantum computing"      # a one-paragraph summary
```

It talks to the public Wikimedia APIs over plain HTTPS and is a polite client
by default: it rate-limits itself, retries with backoff, honours `Retry-After`
and `maxlag`, sends a descriptive User-Agent, and caches responses on disk. The
binary is pure Go with no runtime dependencies and no CGO.

## Install

```bash
# Go
go install github.com/tamnd/wikipedia-cli/cmd/wiki@latest

# Homebrew (once the tap is published)
brew install tamnd/tap/wiki

# Docker
docker run --rm ghcr.io/tamnd/wiki read "Alan Turing"
```

Release archives, `.deb`/`.rpm`/`.apk` packages, a Scoop manifest, checksums,
SBOMs and a cosign signature are attached to every
[GitHub release](https://github.com/tamnd/wikipedia-cli/releases).

## What you can do with it

- **Read.** `wiki read` renders an article as paged plain text by default, or
  `--markdown`, `--html`, `--wikitext`, or `--summary`. `wiki get` is the
  scriptable, never-paged sibling for pipelines.
- **Search & discover.** Full-text `search` (CirrusSearch operators pass
  through), prefix `suggest`, `random`, and `related` pages.
- **Walk structure.** `links`, `backlinks`, `categories`, `category` members,
  `media`, `references`, `langs` (interlanguage links), `info`, and `cite`.
- **Follow history.** `revisions` and a unified `diff` between any two
  revisions.
- **Browse feeds.** `featured` (today's featured article, most-read, picture of
  the day, in the news), `onthisday`, and the most-viewed `top` list.
- **Measure.** `pageviews` time series for an article and the `stats` for a
  whole wiki.
- **Map.** `geosearch` near a coordinate and `nearby` an article.
- **Query Wikidata.** `entity` lookups and raw `sparql` against the Query
  Service.
- **Go bulk.** `dump list`/`download`/`pages`/`grep` over the public XML dumps,
  with resume, sha1 verification, and constant-memory streaming.
- **Go offline.** `dump export` turns a whole dump into a corpus of clean
  Markdown (or text), one file per article, parsing wikitext and keeping code
  blocks, headings, and links while dropping template and reference chrome.

Every command shares one output contract: `-o table` (the default on a
terminal), `json`, `jsonl` (the default in a pipe), `csv`, `tsv`, `url`, and Go
`--template`. Pick columns with `--fields`, cap results with `-n`, and target
any wiki with `-l/--lang`, `--project`, or `--site`.

## Multiple wikis

```bash
wiki read "Berlin" -l de                       # German Wikipedia
wiki search "café" --project wiktionary -l fr  # French Wiktionary
wiki entity Q937                               # Wikidata
wiki read https://en.wikipedia.org/wiki/Cat    # a pasted URL just works
```

## Documentation

Full docs live at <https://wikipedia-cli.tamnd.com>. Start with the
[introduction](https://wikipedia-cli.tamnd.com/getting-started/introduction/)
and the
[quick start](https://wikipedia-cli.tamnd.com/getting-started/quick-start/),
or jump to the
[CLI reference](https://wikipedia-cli.tamnd.com/reference/cli/).

## Building from source

```bash
git clone https://github.com/tamnd/wikipedia-cli
cd wikipedia-cli
make build          # builds ./bin/wiki
make test
```

Requires Go 1.26+.

## License

[Apache-2.0](LICENSE).

`wiki` is an independent tool and is not affiliated with or endorsed by the
Wikimedia Foundation. Article content is licensed by its authors under
[CC BY-SA](https://creativecommons.org/licenses/by-sa/4.0/); please respect the
Wikimedia [API etiquette](https://www.mediawiki.org/wiki/API:Etiquette) and
attribute your sources.
