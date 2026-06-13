---
title: "wiki"
description: "A fast, friendly command line for Wikipedia. Read articles as text or Markdown, search full text, pull summaries, walk links and categories, query Wikidata, fetch metrics, and stream the public dumps, all from one binary."
heroTitle: "Wikipedia, from the command line"
heroLead: "wiki is a single pure-Go binary that puts Wikipedia and the wider Wikimedia world behind a tool that feels like curl. Read an article, search the full text, pull a summary, walk links and categories, query Wikidata with SPARQL, fetch pageview metrics, and stream the XML dumps, with no account and nothing to pay for."
heroPrimaryURL: "/getting-started/quick-start/"
heroPrimaryText: "Get started"
---

Working with Wikipedia from a script usually means juggling the MediaWiki
Action API, the REST endpoints, the Wikidata Query Service, and a pile of glue
code. wiki puts all of it behind one tool with sensible defaults, real output
formats, and pipelines that compose.

```bash
wiki read "Alan Turing"               # the readable article in your pager
wiki search "turing machine" -o url   # every matching title as a URL
wiki summary "Quantum computing"      # a one-paragraph summary
wiki entity Q937                       # the Wikidata entity for Einstein
```

It speaks to the public Wikimedia hosts over plain HTTPS, so there is nothing
to sign up for. The binary is pure Go with no runtime dependencies, and it is a
polite client out of the box: self-throttling, retrying with backoff, honouring
`Retry-After` and `maxlag`, and caching responses on disk.

## What you can do with it

- **Read articles.** `wiki read` renders clean plain text or Markdown, the
  server HTML, the raw wikitext, or a summary, and pages it for you. `wiki get`
  is the scriptable sibling that never pages and reads titles on stdin.
- **Search and discover.** Full-text `search` with CirrusSearch operators,
  prefix `suggest`, `random` articles, and `related` pages.
- **Walk the structure.** List a page's `links`, `backlinks`, `categories`,
  `media`, and `references`, find the same article in other `langs`, and read
  its `info`.
- **Follow the history.** Page `revisions` and a unified `diff` between any two.
- **Browse feeds and metrics.** The `featured` feed, `onthisday` events, the
  most-viewed `top` list, and a `pageviews` time series.
- **Map and link.** `geosearch` near a coordinate, `nearby` an article, and
  Wikidata `entity` and `sparql` queries.
- **Go bulk.** `dump` the public XML archives with resume, sha1 verification,
  and constant-memory streaming.

## Where to go next

- New here? Start with the [introduction](/getting-started/introduction/) for
  the mental model, then the [quick start](/getting-started/quick-start/).
- Want to install it? See [installation](/getting-started/installation/).
- Looking for a specific task? The [guides](/guides/) cover reading, searching,
  page structure, history, feeds and metrics, geo, Wikidata, and dumps.
- Need every flag? The [CLI reference](/reference/cli/) is the full surface.
