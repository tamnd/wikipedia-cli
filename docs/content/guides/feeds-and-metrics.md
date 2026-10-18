---
title: "Feeds and metrics"
description: "Browse the daily featured feed and on-this-day events, list the most-viewed articles, and chart pageviews over time."
weight: 6
---

## The featured feed

Wikipedia curates a daily feed: today's featured article, the most-read pages,
the picture of the day, in-the-news stories, and on-this-day highlights.

```bash
wiki featured                 # today
wiki featured 2020-07-20      # a specific date
wiki featured -o jsonl        # the whole feed as structured data
```

The plain output is a digest, one line per item. The structured output is a
faithful copy of the REST feed: each article keeps its full page summary
(thumbnail, original image, extract and extract_html, wikibase item, namespace,
and desktop and mobile URLs), the most-read entries keep their per-day
`view_history`, the picture of the day keeps its license, credit, artist and
file page, the in-the-news stories keep the summaries of every article they
link, and the on-this-day highlights keep the summaries of their linked pages.
Nothing the feed returns is dropped:

```bash
wiki featured 2020-07-20 -o json | jq '.tfa.content_urls.mobile.page'
wiki featured 2020-07-20 -o json | jq '.image.license'
```

## On this day

Historical events for a calendar day, across all years:

```bash
wiki onthisday                       # today
wiki onthisday 07-20                 # July 20
wiki onthisday 07-20 --type births   # births only
wiki onthisday --type deaths -o jsonl
```

The `--type` slices are `all`, `selected`, `births`, `deaths`, `holidays`, and
`events`. The table shows the year, the event text, and the linked page titles;
`-o json` keeps the full page summary of every linked article, so you can pull
their extracts or thumbnails straight from the event.

## Most-viewed articles

The top articles for a day or a whole month. It defaults to yesterday, the most
recent complete day:

```bash
wiki top                       # yesterday
wiki top 2024-01-01 -n 25      # a specific day
wiki top 2024-01 -o jsonl      # a whole month
```

## Pageviews over time

A daily or monthly time series for one article. Defaults to the last 30 days:

```bash
wiki pageviews "Alan Turing"
wiki pageviews "Pi" --from 2024-01-01 --to 2024-03-31 --granularity monthly
wiki pageviews "Cat" --days 90 -o csv
```

Filter by access method or agent type when you need to:

```bash
wiki pageviews "Wikipedia" --access mobile-web --agent user
```

CSV output drops straight into a spreadsheet or a plotting script:

```bash
wiki pageviews "ChatGPT" --days 180 -o csv > views.csv
```
