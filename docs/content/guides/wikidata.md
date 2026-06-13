---
title: "Wikidata"
description: "Look up structured entities by id or article title, and run raw SPARQL against the Wikidata Query Service."
weight: 7
---

Wikidata is the structured knowledge graph behind Wikipedia. wiki reads it two
ways: entity lookups for a single item, and SPARQL for questions across the
whole graph.

## Entity lookups

Pass a Q-id (or P-id) to see an item's label, description, aliases, and claims:

```bash
wiki entity Q937              # Albert Einstein
wiki entity Q64 --lang de    # Berlin, labels in German
```

Restrict the claims to the properties you care about:

```bash
wiki entity Q937 --props P31,P569,P570   # instance-of, born, died
```

Do not know the id? Resolve it from a Wikipedia title with `--title` (wiki
follows the article's `wikibase_item`):

```bash
wiki entity "Albert Einstein" --title --props P569,P570
```

`-o json` gives you the flattened entity for a pipeline:

```bash
wiki entity Q937 -o json
```

## SPARQL

Run any query against the Wikidata Query Service. The query can be inline, read
from a file with `@path`, or read from stdin with `-`. Entity URIs in the
result are shortened to bare Q/P ids:

```bash
wiki sparql 'SELECT ?city ?pop WHERE { ?city wdt:P31 wd:Q515; wdt:P1082 ?pop } ORDER BY DESC(?pop) LIMIT 10'
```

From a file, rendered as CSV:

```bash
wiki sparql @capitals.rq -o csv
```

From stdin:

```bash
echo 'SELECT ?p WHERE { wd:Q937 wdt:P800 ?p }' | wiki sparql -
```

Each SELECT variable becomes a column, so the result drops straight into a
table, CSV, or JSONL just like every other command.
