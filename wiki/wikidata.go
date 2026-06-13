package wiki

import (
	"context"
	"net/url"
	"sort"
	"strconv"
	"strings"
)

// Entity is a flattened Wikidata entity.
type Entity struct {
	ID          string              `json:"id"`
	Label       string              `json:"label"`
	Description string              `json:"description,omitempty"`
	Aliases     []string            `json:"aliases,omitempty"`
	Claims      map[string][]string `json:"claims,omitempty"`
}

// wikidataClient returns a Client bound to www.wikidata.org, reusing the same
// HTTP client so politeness and caching carry over.
func (c *Client) wikidataClient() (*Client, error) {
	cfg := c.Cfg
	cfg.Project = "wikidata"
	cfg.SiteHost = ""
	site, err := cfg.Site()
	if err != nil {
		return nil, err
	}
	return &Client{Site: site, HTTP: c.HTTP, Cfg: cfg}, nil
}

// EntityByID fetches and flattens a Wikidata entity (Q… or P… id).
func (c *Client) EntityByID(ctx context.Context, id, lang string, props []string) (*Entity, error) {
	wd, err := c.wikidataClient()
	if err != nil {
		return nil, err
	}
	if lang == "" {
		lang = firstNonEmpty(c.Cfg.Lang, "en")
	}
	v := wd.actionParams()
	v.Set("action", "wbgetentities")
	v.Set("ids", id)
	v.Set("languages", lang)
	v.Set("props", "labels|descriptions|aliases|claims")
	var resp struct {
		apiError
		Entities map[string]wbEntity `json:"entities"`
	}
	if err := wd.HTTP.GetJSON(ctx, wd.Site.APIURL(v), ttlContent, &resp); err != nil {
		return nil, err
	}
	if err := resp.err(); err != nil {
		return nil, err
	}
	e, ok := resp.Entities[id]
	if !ok || e.Missing != nil {
		return nil, ErrNotFound
	}
	return e.flatten(id, lang, props), nil
}

// EntityByTitle resolves a Wikipedia title to its Wikidata entity, then fetches
// it. Uses pageprops.wikibase_item on the current wiki.
func (c *Client) EntityByTitle(ctx context.Context, title, lang string, props []string) (*Entity, error) {
	v := c.actionParams()
	v.Set("action", "query")
	v.Set("prop", "pageprops")
	v.Set("ppprop", "wikibase_item")
	v.Set("redirects", "1")
	v.Set("titles", title)
	var resp struct {
		apiError
		Query struct {
			Pages []struct {
				Missing   bool `json:"missing"`
				PageProps struct {
					Item string `json:"wikibase_item"`
				} `json:"pageprops"`
			} `json:"pages"`
		} `json:"query"`
	}
	if err := c.actionJSON(ctx, v, ttlContent, &resp); err != nil {
		return nil, err
	}
	if err := resp.err(); err != nil {
		return nil, err
	}
	if len(resp.Query.Pages) == 0 || resp.Query.Pages[0].Missing || resp.Query.Pages[0].PageProps.Item == "" {
		return nil, ErrNotFound
	}
	return c.EntityByID(ctx, resp.Query.Pages[0].PageProps.Item, lang, props)
}

type wbEntity struct {
	Missing *struct{} `json:"missing"`
	Labels  map[string]struct {
		Value string `json:"value"`
	} `json:"labels"`
	Descriptions map[string]struct {
		Value string `json:"value"`
	} `json:"descriptions"`
	Aliases map[string][]struct {
		Value string `json:"value"`
	} `json:"aliases"`
	Claims map[string][]struct {
		Mainsnak struct {
			Datavalue struct {
				Type  string `json:"type"`
				Value any    `json:"value"`
			} `json:"datavalue"`
		} `json:"mainsnak"`
	} `json:"claims"`
}

func (e wbEntity) flatten(id, lang string, props []string) *Entity {
	out := &Entity{ID: id, Claims: map[string][]string{}}
	if l, ok := e.Labels[lang]; ok {
		out.Label = l.Value
	} else {
		for _, l := range e.Labels {
			out.Label = l.Value
			break
		}
	}
	if d, ok := e.Descriptions[lang]; ok {
		out.Description = d.Value
	}
	for _, as := range e.Aliases[lang] {
		out.Aliases = append(out.Aliases, as.Value)
	}
	want := map[string]bool{}
	for _, p := range props {
		want[strings.ToUpper(strings.TrimSpace(p))] = true
	}
	for pid, claims := range e.Claims {
		if len(want) > 0 && !want[pid] {
			continue
		}
		for _, cl := range claims {
			out.Claims[pid] = append(out.Claims[pid], snakValue(cl.Mainsnak.Datavalue.Type, cl.Mainsnak.Datavalue.Value))
		}
	}
	return out
}

// snakValue renders a Wikidata datavalue into a short string.
func snakValue(typ string, val any) string {
	switch v := val.(type) {
	case string:
		return v
	case map[string]any:
		switch typ {
		case "wikibase-entityid":
			if id, ok := v["id"].(string); ok {
				return id
			}
		case "time":
			if t, ok := v["time"].(string); ok {
				return strings.TrimPrefix(t, "+")
			}
		case "quantity":
			if a, ok := v["amount"].(string); ok {
				return strings.TrimPrefix(a, "+")
			}
		case "globecoordinate":
			lat, _ := v["latitude"].(float64)
			lon, _ := v["longitude"].(float64)
			return formatFloat(lat) + "," + formatFloat(lon)
		case "monolingualtext":
			if t, ok := v["text"].(string); ok {
				return t
			}
		}
	}
	return ""
}

// SPARQLResult holds the column order and rows of a flattened SPARQL response.
type SPARQLResult struct {
	Vars []string
	Rows []map[string]string
}

// SPARQL runs a query against the Wikidata Query Service and flattens the JSON
// results into rows keyed by SELECT variable.
func (c *Client) SPARQL(ctx context.Context, query string) (*SPARQLResult, error) {
	v := url.Values{}
	v.Set("query", query)
	v.Set("format", "json")
	u := WikidataSPARQL + "?" + v.Encode()
	var resp struct {
		Head struct {
			Vars []string `json:"vars"`
		} `json:"head"`
		Results struct {
			Bindings []map[string]struct {
				Value string `json:"value"`
			} `json:"bindings"`
		} `json:"results"`
	}
	if err := c.HTTP.GetJSON(ctx, u, ttlSearch, &resp); err != nil {
		return nil, err
	}
	out := &SPARQLResult{Vars: resp.Head.Vars}
	for _, b := range resp.Results.Bindings {
		row := map[string]string{}
		for k, cell := range b {
			row[k] = shortenURI(cell.Value)
		}
		out.Rows = append(out.Rows, row)
	}
	if len(out.Vars) == 0 {
		// Fall back to sorted keys from the first row.
		seen := map[string]bool{}
		for _, r := range out.Rows {
			for k := range r {
				if !seen[k] {
					seen[k] = true
					out.Vars = append(out.Vars, k)
				}
			}
		}
		sort.Strings(out.Vars)
	}
	return out, nil
}

// shortenURI collapses a Wikidata entity URI to its bare Q/P id.
func shortenURI(s string) string {
	for _, prefix := range []string{
		"http://www.wikidata.org/entity/",
		"http://www.wikidata.org/prop/direct/",
	} {
		if rest, ok := strings.CutPrefix(s, prefix); ok {
			return rest
		}
	}
	return s
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func formatFloat(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}
