---
title: "Configuration"
description: "Global flags, environment variables, the paths wiki uses, and the polite-client settings."
weight: 2
---

wiki needs no configuration to run. Everything below has a sensible default;
override it with a flag or an environment variable when you want to.

## Choosing a wiki

| Flag | Env | Default | Description |
| --- | --- | --- | --- |
| `-l, --lang` | `WIKI_LANG` | `en` | Language subdomain, e.g. `de`, `fr`, `ja`. |
| `--project` | `WIKI_PROJECT` | `wikipedia` | Project: `wikipedia`, `wiktionary`, `wikibooks`, `wikinews`, `wikiquote`, `wikisource`, `wikiversity`, `wikivoyage`, `wikidata`, `commons`, `species`, `meta`. |
| `--site` | `WIKI_SITE` | (none) | An explicit host like `en.wikipedia.org`; overrides lang and project. |

Run `wiki sites` to see every project and an example host.

A pasted Wikipedia URL as a command's argument selects the wiki automatically,
regardless of these settings.

## The polite client

wiki is built to be easy on the Wikimedia servers. These control how:

| Flag | Default | Description |
| --- | --- | --- |
| `--rate` | `150ms` | Minimum delay between requests. |
| `--retries` | `4` | Retry attempts on `429` and `5xx`, with backoff. |
| `--timeout` | `60s` | Per-request timeout. |
| `--ua` | built-in | The User-Agent sent on every request. |

The client also honours the server's `Retry-After` header and sends
`maxlag=5` to the Action API so it backs off when database replicas are behind.
You can set a contact-carrying User-Agent with `--ua` or `WIKI_USER_AGENT`,
which the Wikimedia [API etiquette](https://www.mediawiki.org/wiki/API:Etiquette)
appreciates for heavier use.

## Caching

Responses are cached on disk with a per-kind TTL, so re-running a command is
instant and gentle on the servers. Random results and downloads are never
cached.

| Flag | Description |
| --- | --- |
| `--no-cache` | Bypass the cache for this run. |

Inspect or clear it:

```bash
wiki cache path
wiki cache info
wiki cache clear
```

## Paths

All state lives under one tree so the footprint is predictable.

| Path | Env | Default |
| --- | --- | --- |
| Data dir | `WIKI_DATA_DIR` | `~/data/wiki` |
| Cache dir | `WIKI_CACHE_DIR` | `<data-dir>/cache` |
| Downloads | (none) | `<data-dir>/downloads` |
| Config dir | `XDG_CONFIG_HOME` | `~/.config/wiki` |

`--data-dir` moves the whole tree at once. See the resolved values any time:

```bash
wiki config show
wiki config show -o json
```

## Output

The output contract is shared by every command and documented in full on the
[output formats](/reference/output/) page. The headline flags are `-o` for the
format, `--fields` to pick columns, `-n` to cap results, and `--template` for
Go templates.

## Security

wiki only talks to Wikimedia hosts. A `--site` value or a pasted URL pointing
somewhere else is refused unless you pass `--allow-any-host`, which guards
against a hostile input redirecting the client. The tool is read-only and sends
no credentials.
