---
title: "Resource URIs"
description: "Use wiki as a database/sql-style driver so a host program can address Wikipedia pages and categories as wikipedia:// URIs."
weight: 10
---

`wiki` is a command line, but the `wiki` Go package is also a small driver that
makes Wikipedia addressable as a resource URI. A host program registers it the
way a program registers a database driver with `database/sql`, then dereferences
`wikipedia://` URIs without knowing anything about the MediaWiki API.

The host that does this today is [ant](https://github.com/tamnd/ant), a single
binary that puts one URI namespace over a family of site tools. The examples
below use `ant`; any program that links the package gets the same behaviour.

## Mounting the driver

A host enables the driver with one blank import, exactly like `import _
"github.com/lib/pq"`:

```go
import _ "github.com/tamnd/wikipedia-cli/wiki"
```

The package's `init` registers a domain with the scheme `wikipedia` (alias
`wiki`) for the hosts `en.wikipedia.org` and `wikipedia.org`. The standalone
`wiki` binary does not use any of this, so the CLI is unchanged.

## Addressing pages and categories

A URI is `scheme://authority/id`. The driver serves two record types:

| URI                                   | What it is                         |
| ------------------------------------- | ---------------------------------- |
| `wikipedia://page/<Title>`            | an article summary (id is the title) |
| `wikipedia://category/<Name>`         | a category page (id is the bare name, no `Category:` prefix) |

```bash
ant get wikipedia://page/Alan_Turing         # the article summary record
ant cat wikipedia://page/Alan_Turing         # just the extract text (the body)
ant url wikipedia://page/Alan_Turing         # the live https URL
ant get wikipedia://category/Computability    # a category's metadata
```

A title with spaces works either way; `Alan_Turing` and `Alan Turing` name the
same page, and a pasted URL resolves too:

```bash
ant get "wikipedia://page/Alan Turing"
ant resolve https://en.wikipedia.org/wiki/Alan_Turing   # -> wikipedia://page/Alan%20Turing
```

## Walking the graph

`ls` lists the members of a collection, and every member is itself an
addressable page URI, so a host can follow the graph:

```bash
ant ls wikipedia://page/Alan_Turing           # the articles this page links to
ant ls wikipedia://category/Computability      # the articles in the category
ant export wikipedia://page/Alan_Turing --follow 1 --to ./corpus
```

Page links are article-namespace links; category members are the articles in the
category (not its subcategories or files). Both come back as page summaries
keyed by title.

## One wiki

The driver speaks the default English Wikipedia, where a bare title is
unambiguous. To read another language or project, or to query Wikidata, use the
`wiki` binary and its `-l/--lang`, `--project`, and `--site` flags as described
in [Multiple wikis](/getting-started/quick-start/). The driver and the binary
share the same on-disk cache, so a page fetched one way is warm for the other.
