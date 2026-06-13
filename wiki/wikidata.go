package wiki

import (
	"context"
	"encoding/json"
	"net/url"
	"sort"
	"strconv"
	"strings"
)

// Entity is a Wikidata entity with its full structure preserved. The JSON
// encoding mirrors the wbgetentities response, so labels, descriptions,
// aliases, statements (with qualifiers, references and ranks), sitelinks and
// the entity datatype all survive a round trip.
type Entity struct {
	ID           string                 `json:"id"`
	PageID       int                    `json:"pageid,omitempty"`
	NS           int                    `json:"ns,omitempty"`
	Title        string                 `json:"title,omitempty"`
	Type         string                 `json:"type,omitempty"`
	Datatype     string                 `json:"datatype,omitempty"`
	Labels       map[string]TermValue   `json:"labels,omitempty"`
	Descriptions map[string]TermValue   `json:"descriptions,omitempty"`
	Aliases      map[string][]TermValue `json:"aliases,omitempty"`
	Claims       map[string][]Statement `json:"claims,omitempty"`
	Sitelinks    map[string]Sitelink    `json:"sitelinks,omitempty"`
	LastRevID    int                    `json:"lastrevid,omitempty"`
	Modified     string                 `json:"modified,omitempty"`

	// missing is set by wbgetentities for an unknown id; never emitted.
	missing *string
}

// TermValue is a language-tagged label, description or alias.
type TermValue struct {
	Language string `json:"language"`
	Value    string `json:"value"`
}

// Statement is one claim on an entity, with its rank, qualifiers and
// references intact.
type Statement struct {
	ID              string            `json:"id,omitempty"`
	Type            string            `json:"type,omitempty"`
	Rank            string            `json:"rank,omitempty"`
	Mainsnak        Snak              `json:"mainsnak"`
	Qualifiers      map[string][]Snak `json:"qualifiers,omitempty"`
	QualifiersOrder []string          `json:"qualifiers-order,omitempty"`
	References      []Reference       `json:"references,omitempty"`
}

// Snak is a property-value assertion. Datavalue is kept as raw JSON so the
// full value structure (time precision, quantity bounds, coordinate globe,
// monolingual language) survives; ValueString renders a short form on demand.
type Snak struct {
	SnakType  string     `json:"snaktype"`
	Property  string     `json:"property"`
	Hash      string     `json:"hash,omitempty"`
	Datatype  string     `json:"datatype,omitempty"`
	Datavalue *Datavalue `json:"datavalue,omitempty"`
}

// Datavalue is the typed value of a snak.
type Datavalue struct {
	Type  string          `json:"type"`
	Value json.RawMessage `json:"value"`
}

// Reference is a citation attached to a statement.
type Reference struct {
	Hash       string            `json:"hash,omitempty"`
	Snaks      map[string][]Snak `json:"snaks,omitempty"`
	SnaksOrder []string          `json:"snaks-order,omitempty"`
}

// Sitelink connects an entity to a page on a Wikimedia site.
type Sitelink struct {
	Site   string   `json:"site"`
	Title  string   `json:"title"`
	Badges []string `json:"badges,omitempty"`
	URL    string   `json:"url,omitempty"`
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

// EntityByID fetches a Wikidata entity (Q… or P… id) in full. props, when set,
// restricts the claims to those property ids; everything else is always kept.
func (c *Client) EntityByID(ctx context.Context, id, lang string, props []string) (*Entity, error) {
	wd, err := c.wikidataClient()
	if err != nil {
		return nil, err
	}
	v := wd.actionParams()
	v.Set("action", "wbgetentities")
	v.Set("ids", id)
	// No languages filter: keep every language so the JSON is a full record.
	// lang only chooses which language the flattened display fields use.
	v.Set("props", "labels|descriptions|aliases|claims|sitelinks/urls|datatype")
	var resp struct {
		apiError
		Entities map[string]*Entity `json:"entities"`
	}
	if err := wd.HTTP.GetJSON(ctx, wd.Site.APIURL(v), ttlContent, &resp); err != nil {
		return nil, err
	}
	if err := resp.err(); err != nil {
		return nil, err
	}
	e, ok := resp.Entities[id]
	if !ok || e == nil || e.missing != nil {
		return nil, ErrNotFound
	}
	e.restrictClaims(props)
	return e, nil
}

// UnmarshalJSON detects the missing marker (wbgetentities emits "missing":"")
// while decoding the rest of the entity normally.
func (e *Entity) UnmarshalJSON(b []byte) error {
	type alias Entity
	aux := struct {
		Missing *string `json:"missing"`
		*alias
	}{alias: (*alias)(e)}
	if err := json.Unmarshal(b, &aux); err != nil {
		return err
	}
	e.missing = aux.Missing
	return nil
}

// restrictClaims drops every claim whose property id is not in props. An empty
// props keeps all claims.
func (e *Entity) restrictClaims(props []string) {
	if len(props) == 0 || e.Claims == nil {
		return
	}
	want := map[string]bool{}
	for _, p := range props {
		want[strings.ToUpper(strings.TrimSpace(p))] = true
	}
	for pid := range e.Claims {
		if !want[pid] {
			delete(e.Claims, pid)
		}
	}
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

// LabelFor returns the label in lang, falling back to any available language.
func (e *Entity) LabelFor(lang string) string { return pickTerm(e.Labels, lang) }

// DescriptionFor returns the description in lang, falling back to any language.
func (e *Entity) DescriptionFor(lang string) string { return pickTerm(e.Descriptions, lang) }

// AliasesFor returns the aliases in lang, or none if that language has no set.
func (e *Entity) AliasesFor(lang string) []string {
	var out []string
	for _, a := range e.Aliases[lang] {
		out = append(out, a.Value)
	}
	return out
}

func pickTerm(m map[string]TermValue, lang string) string {
	if t, ok := m[lang]; ok {
		return t.Value
	}
	if t, ok := m["en"]; ok {
		return t.Value
	}
	// Deterministic fallback: the lowest language code present.
	langs := make([]string, 0, len(m))
	for l := range m {
		langs = append(langs, l)
	}
	sort.Strings(langs)
	if len(langs) > 0 {
		return m[langs[0]].Value
	}
	return ""
}

// ValueString renders a snak's value into a short string for table output.
// novalue/somevalue snaks render as "(no value)" / "(unknown value)".
func (s Snak) ValueString() string {
	switch s.SnakType {
	case "novalue":
		return "(no value)"
	case "somevalue":
		return "(unknown value)"
	}
	if s.Datavalue == nil {
		return ""
	}
	return datavalueString(s.Datavalue.Type, s.Datavalue.Value)
}

func datavalueString(typ string, raw json.RawMessage) string {
	switch typ {
	case "string":
		var str string
		if json.Unmarshal(raw, &str) == nil {
			return str
		}
	case "wikibase-entityid":
		var v struct {
			ID string `json:"id"`
		}
		if json.Unmarshal(raw, &v) == nil {
			return v.ID
		}
	case "time":
		var v struct {
			Time string `json:"time"`
		}
		if json.Unmarshal(raw, &v) == nil {
			return strings.TrimPrefix(v.Time, "+")
		}
	case "quantity":
		var v struct {
			Amount string `json:"amount"`
		}
		if json.Unmarshal(raw, &v) == nil {
			return strings.TrimPrefix(v.Amount, "+")
		}
	case "globecoordinate":
		var v struct {
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
		}
		if json.Unmarshal(raw, &v) == nil {
			return formatFloat(v.Latitude) + "," + formatFloat(v.Longitude)
		}
	case "monolingualtext":
		var v struct {
			Text string `json:"text"`
		}
		if json.Unmarshal(raw, &v) == nil {
			return v.Text
		}
	}
	// Unknown type: return the raw JSON so nothing is silently lost.
	return strings.TrimSpace(string(raw))
}

// SPARQLBinding is one cell of a SPARQL result: its RDF term kind, value, and
// optional language tag or datatype.
type SPARQLBinding struct {
	Type     string `json:"type"`
	Value    string `json:"value"`
	XMLLang  string `json:"xml:lang,omitempty"`
	Datatype string `json:"datatype,omitempty"`
}

// SPARQLResult holds the column order and rows of a SPARQL response. Each row
// maps a SELECT variable to its full binding.
type SPARQLResult struct {
	Vars []string                   `json:"vars"`
	Rows []map[string]SPARQLBinding `json:"rows"`
}

// SPARQL runs a query against the Wikidata Query Service and returns the rows
// keyed by SELECT variable, preserving each binding's type, language and
// datatype.
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
			Bindings []map[string]SPARQLBinding `json:"bindings"`
		} `json:"results"`
	}
	if err := c.HTTP.GetJSON(ctx, u, ttlSearch, &resp); err != nil {
		return nil, err
	}
	out := &SPARQLResult{Vars: resp.Head.Vars, Rows: resp.Results.Bindings}
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

// ShortenURI collapses a Wikidata entity or property URI to its bare Q/P id for
// compact display. The full URI is always kept in the structured output.
func ShortenURI(s string) string {
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

func formatFloat(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}
