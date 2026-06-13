package wiki

import (
	"encoding/json"
	"testing"
)

// A trimmed but representative wbgetentities entity: a statement with a
// qualifier, a reference, a rank, and a typed time datavalue, plus a sitelink
// with a badge. Every one of these must survive the decode.
const sampleEntity = `{
  "id": "Q42",
  "type": "item",
  "lastrevid": 123456,
  "modified": "2024-01-02T03:04:05Z",
  "labels": {"en": {"language": "en", "value": "Douglas Adams"}, "de": {"language": "de", "value": "Douglas Adams"}},
  "descriptions": {"en": {"language": "en", "value": "English writer"}},
  "aliases": {"en": [{"language": "en", "value": "Douglas Noel Adams"}]},
  "claims": {
    "P569": [{
      "id": "Q42$abc",
      "type": "statement",
      "rank": "normal",
      "mainsnak": {
        "snaktype": "value",
        "property": "P569",
        "datatype": "time",
        "datavalue": {"type": "time", "value": {"time": "+1952-03-11T00:00:00Z", "precision": 11, "calendarmodel": "http://www.wikidata.org/entity/Q1985727"}}
      },
      "qualifiers": {"P1326": [{"snaktype": "value", "property": "P1326", "datatype": "time", "datavalue": {"type": "time", "value": {"time": "+1952-00-00T00:00:00Z"}}}]},
      "qualifiers-order": ["P1326"],
      "references": [{"hash": "ref1", "snaks": {"P143": [{"snaktype": "value", "property": "P143", "datavalue": {"type": "wikibase-entityid", "value": {"id": "Q328"}}}]}, "snaks-order": ["P143"]}]
    }],
    "P31": [{"mainsnak": {"snaktype": "somevalue", "property": "P31", "datatype": "wikibase-item"}}]
  },
  "sitelinks": {"enwiki": {"site": "enwiki", "title": "Douglas Adams", "badges": ["Q17437798"], "url": "https://en.wikipedia.org/wiki/Douglas_Adams"}}
}`

func TestEntityDecodePreservesFullStructure(t *testing.T) {
	var e Entity
	if err := json.Unmarshal([]byte(sampleEntity), &e); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if e.missing != nil {
		t.Fatalf("entity marked missing")
	}
	if got := e.LabelFor("de"); got != "Douglas Adams" {
		t.Errorf("LabelFor(de) = %q", got)
	}
	if e.LastRevID != 123456 || e.Modified == "" {
		t.Errorf("revision metadata dropped: %d %q", e.LastRevID, e.Modified)
	}

	st := e.Claims["P569"][0]
	if st.Rank != "normal" || st.ID != "Q42$abc" {
		t.Errorf("statement rank/id dropped: %+v", st)
	}
	if len(st.Qualifiers["P1326"]) != 1 {
		t.Errorf("qualifiers dropped")
	}
	if len(st.References) != 1 || st.References[0].Hash != "ref1" {
		t.Errorf("references dropped: %+v", st.References)
	}
	if got := st.Mainsnak.ValueString(); got != "1952-03-11T00:00:00Z" {
		t.Errorf("time value = %q", got)
	}
	// The raw datavalue must still carry precision and calendarmodel.
	if !json.Valid(st.Mainsnak.Datavalue.Value) {
		t.Fatalf("datavalue not valid json")
	}
	var dv map[string]any
	_ = json.Unmarshal(st.Mainsnak.Datavalue.Value, &dv)
	if dv["precision"] == nil || dv["calendarmodel"] == nil {
		t.Errorf("time precision/calendarmodel dropped: %v", dv)
	}

	// somevalue snaks must be distinguishable, not empty.
	if got := e.Claims["P31"][0].Mainsnak.ValueString(); got != "(unknown value)" {
		t.Errorf("somevalue snak = %q", got)
	}

	sl := e.Sitelinks["enwiki"]
	if sl.Title != "Douglas Adams" || len(sl.Badges) != 1 || sl.URL == "" {
		t.Errorf("sitelink dropped: %+v", sl)
	}
}

func TestEntityMissingMarker(t *testing.T) {
	var e Entity
	if err := json.Unmarshal([]byte(`{"id":"Q0","missing":""}`), &e); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if e.missing == nil {
		t.Errorf("missing marker not detected")
	}
}

func TestSPARQLBindingPreservesTypeAndLang(t *testing.T) {
	const body = `{
      "head": {"vars": ["item", "label", "pop"]},
      "results": {"bindings": [
        {"item": {"type": "uri", "value": "http://www.wikidata.org/entity/Q515"},
         "label": {"type": "literal", "xml:lang": "en", "value": "city"},
         "pop": {"type": "literal", "datatype": "http://www.w3.org/2001/XMLSchema#integer", "value": "1000"}}
      ]}
    }`
	var resp struct {
		Head struct {
			Vars []string `json:"vars"`
		} `json:"head"`
		Results struct {
			Bindings []map[string]SPARQLBinding `json:"bindings"`
		} `json:"results"`
	}
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	b := resp.Results.Bindings[0]
	if b["item"].Type != "uri" {
		t.Errorf("binding type dropped: %+v", b["item"])
	}
	if b["label"].XMLLang != "en" {
		t.Errorf("xml:lang dropped: %+v", b["label"])
	}
	if b["pop"].Datatype == "" {
		t.Errorf("datatype dropped: %+v", b["pop"])
	}
	if ShortenURI(b["item"].Value) != "Q515" {
		t.Errorf("ShortenURI = %q", ShortenURI(b["item"].Value))
	}
}
