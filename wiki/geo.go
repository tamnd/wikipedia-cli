package wiki

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
)

// GeoResult is an article near a coordinate. It keeps every field the
// geosearch list returns under the widened gsprop set: the page id and
// namespace, whether this is the page's primary coordinate, and the feature
// type, name, dimension, country, region and globe.
type GeoResult struct {
	Pageid  int             `json:"pageid,omitempty"`
	NS      int             `json:"ns,omitempty"`
	Title   string          `json:"title"`
	Lat     float64         `json:"lat"`
	Lon     float64         `json:"lon"`
	Dist    float64         `json:"dist"` // metres
	Primary bool            `json:"primary,omitempty"`
	Type    string          `json:"type,omitempty"`
	Name    string          `json:"name,omitempty"`
	Dim     json.RawMessage `json:"dim,omitempty"` // number or string, kept verbatim
	Country string          `json:"country,omitempty"`
	Region  string          `json:"region,omitempty"`
	Globe   string          `json:"globe,omitempty"`
	URL     string          `json:"url"`
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
	v.Set("gsprop", geoProps)
	return c.geoQuery(ctx, v, limit)
}

// geoProps is the full set of properties the geosearch list can return for each
// result, requested so the structured output is lossless.
const geoProps = "type|name|dim|country|region|globe"

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
	v.Set("gsprop", geoProps)
	return c.geoQuery(ctx, v, limit)
}

func (c *Client) geoQuery(ctx context.Context, v url.Values, limit int) ([]GeoResult, error) {
	var resp struct {
		apiError
		Query struct {
			GeoSearch []struct {
				Pageid  int             `json:"pageid"`
				NS      int             `json:"ns"`
				Title   string          `json:"title"`
				Lat     float64         `json:"lat"`
				Lon     float64         `json:"lon"`
				Dist    float64         `json:"dist"`
				Primary json.RawMessage `json:"primary"`
				Type    string          `json:"type"`
				Name    string          `json:"name"`
				Dim     json.RawMessage `json:"dim"`
				Country string          `json:"country"`
				Region  string          `json:"region"`
				Globe   string          `json:"globe"`
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
		out = append(out, GeoResult{
			Pageid: g.Pageid, NS: g.NS, Title: g.Title,
			Lat: g.Lat, Lon: g.Lon, Dist: g.Dist,
			Primary: g.Primary != nil,
			Type:    g.Type, Name: g.Name, Dim: g.Dim,
			Country: g.Country, Region: g.Region, Globe: g.Globe,
			URL: c.Site.PageURL(g.Title),
		})
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out, nil
}
