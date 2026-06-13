---
title: "Feeds and metrics"
description: "Browse the daily featured feed and on-this-day events, list the most-viewed articles, and chart pageviews over time."
weight: 5
---

## The featured feed

Wikipedia curates a daily feed: today's featured article, the most-read pages,
the picture of the day, in-the-news stories, and on-this-day highlights.

```bash
wiki featured                 # today
wiki featured 2020-07-20      # a specific date
wiki featured -o jsonl        # the whole feed as structured data
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
`events`.

## Most-viewed articles

The top articles for a day or a whole month. It defaults to yesterday, the most
recent complete day:

```bash
wiki top                       # yesterday
wiki top 2024-01-01 -n 25      # a specific day
wiki top 2024-01 -o jsonl      # a whole month
```

In JSON each row keeps the URL-path `article` key alongside the readable title,
and the project, access method and the year, month and day the list was drawn
from, so a row is self-describing.

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

Each point in the JSON series keeps its full request context: the project,
article, access method, agent and granularity, plus the raw API timestamp next
to the formatted date.
