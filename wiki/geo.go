package wiki

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// GeoResult is an article near a coordinate.
type GeoResult struct {
	Title string  `json:"title"`
	Lat   float64 `json:"lat"`
	Lon   float64 `json:"lon"`
	Dist  float64 `json:"dist"` // metres
	URL   string  `json:"url"`
}

// GeoSearch returns articles within radius metres of (lat, lon).
func (c *Client) GeoSearch(ctx context.Context, lat, lon float64, radius, limit int) ([]GeoResult, error) {
	if radius <= 0 {
		radius = 1000
	}
	if radius > 10000 {
		radius = 10000
	}
	v := c.actionParams()
	v.Set("action", "query")
	v.Set("list", "geosearch")
	v.Set("gscoord", fmt.Sprintf("%g|%g", lat, lon))
	v.Set("gsradius", strconv.Itoa(radius))
	v.Set("gslimit", limitParam(limit))
	return c.geoQuery(ctx, v, limit)
}

// GeoNear returns articles near another article's coordinates.
func (c *Client) GeoNear(ctx context.Context, title string, radius, limit int) ([]GeoResult, error) {
	if radius <= 0 {
		radius = 1000
	}
	if radius > 10000 {
		radius = 10000
	}
	v := c.actionParams()
	v.Set("action", "query")
	v.Set("list", "geosearch")
	v.Set("gspage", title)
	v.Set("gsradius", strconv.Itoa(radius))
	v.Set("gslimit", limitParam(limit))
	return c.geoQuery(ctx, v, limit)
}

func (c *Client) geoQuery(ctx context.Context, v url.Values, limit int) ([]GeoResult, error) {
	var resp struct {
		apiError
		Query struct {
			GeoSearch []struct {
				Title string  `json:"title"`
				Lat   float64 `json:"lat"`
				Lon   float64 `json:"lon"`
				Dist  float64 `json:"dist"`
			} `json:"geosearch"`
		} `json:"query"`
	}
	if err := c.HTTP.GetJSON(ctx, c.Site.APIURL(v), ttlContent, &resp); err != nil {
		return nil, err
	}
	if err := resp.err(); err != nil {
		return nil, err
	}
	var out []GeoResult
	for _, g := range resp.Query.GeoSearch {
		out = append(out, GeoResult{Title: g.Title, Lat: g.Lat, Lon: g.Lon, Dist: g.Dist, URL: c.Site.PageURL(g.Title)})
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out, nil
}
