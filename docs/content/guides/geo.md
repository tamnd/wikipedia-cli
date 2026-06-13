---
title: "Geo"
description: "Find articles near a coordinate or near another article."
weight: 6
---

Many Wikipedia articles carry coordinates. wiki lets you search by them.

## Near a coordinate

Give a latitude and longitude, as one `lat,lon` argument or as two, and a
radius in metres (up to 10000):

```bash
wiki geosearch 48.8584,2.2945 --radius 2000
wiki geosearch 51.5007 -0.1246 -n 20
```

Each result has the title, its coordinates, the distance in metres, and a URL:

```bash
wiki geosearch 40.7484,-73.9857 -o jsonl
```

## Near another article

When you have a place rather than numbers, search around its article:

```bash
wiki nearby "Eiffel Tower" --radius 1000
wiki nearby "Statue of Liberty" -n 15
```

## A small workflow

List the URLs of everything within 500 metres of a landmark:

```bash
wiki nearby "Colosseum" --radius 500 -o url
```
