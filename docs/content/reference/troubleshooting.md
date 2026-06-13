---
title: "Troubleshooting"
description: "Exit codes and fixes for the situations you are most likely to hit."
weight: 4
---

## Exit codes

wiki returns a precise exit code so scripts can branch on the result:

| Code | Meaning |
| --- | --- |
| `0` | Success. |
| `1` | A runtime error (network failure, server error, unexpected response). |
| `2` | A usage error (bad flag or argument). |
| `3` | The request succeeded but there were no results. |
| `4` | The page or entity was not found. |
| `5` | Rate-limited after exhausting retries. |

```bash
if wiki summary "Some Page" >/dev/null 2>&1; then
  echo "exists"
else
  echo "exit $?"
fi
```

## "No matches" or "not found"

A `3` (no results) means the query ran but matched nothing; a `4` (not found)
means the exact title or id does not exist. Try `search` to find the right
title, or `suggest` to autocomplete it:

```bash
wiki search "the title I half remember"
wiki suggest "Quantu"
```

Remember that titles are case- and punctuation-sensitive after the first
letter. A pasted URL is the most reliable target.

## Wrong wiki or language

If results come from the wrong place, check `-l/--lang` and `--project`, or use
`wiki config show` to see what is resolved. A pasted Wikipedia URL overrides
all of these and is the surest way to hit a specific wiki.

## "refusing non-Wikimedia host"

`--site` and pasted URLs are restricted to Wikimedia hosts as a safety guard.
If you really mean to point wiki at another MediaWiki install, add
`--allow-any-host`.

## Slow or throttled

The client is deliberately polite and may pause between requests. For a large
job, that is expected. If you are being rate-limited (exit `5`), raise
`--rate`, lower `-n`, and set a contact-carrying `--ua`. The on-disk cache
makes repeated runs instant, so avoid `--no-cache` unless you need fresh data.

## Stale results

Responses are cached with a TTL. To force a refresh for one run, add
`--no-cache`; to clear everything, run `wiki cache clear`.

## A dump download stopped partway

`wiki dump download` resumes. Re-run the same command and it continues from the
bytes already on disk, then verifies the sha1 when one is published.

## Filing an issue

If something looks wrong, re-run with `-v` for more detail and include the
command, the output, and `wiki version` in an
[issue](https://github.com/tamnd/wikipedia-cli/issues).
