package wiki

import (
	"context"
	"encoding/json"
	"net/url"
	"testing"
)

func TestProjectPages(t *testing.T) {
	pages := []string{
		`{"continue":{"wppcontinue":"276|1063","continue":"-||"},"query":{"projects":{
"Statistics":[{"pageid":612,"ns":0,"title":"Arithmetic mean","assessment":{"class":"C","importance":"Top"}}],
"Mathematics":[{"pageid":308,"ns":0,"title":"Aristotle","assessment":{"class":"GA","importance":"High"}},{"pageid":9,"ns":14,"title":"Category:Group theory","assessment":{"class":"Category","importance":"NA"}}]}}}`,
		`{"batchcomplete":true,"query":{"projects":{"Mathematics":[{"pageid":634,"ns":0,"title":"Analysis of variance","assessment":{"class":"C","importance":""}}]}}}`,
	}
	var calls []url.Values
	get := func(v url.Values, out any) error {
		c := url.Values{}
		for k, x := range v {
			c[k] = append([]string(nil), x...)
		}
		calls = append(calls, c)
		return json.Unmarshal([]byte(pages[len(calls)-1]), out)
	}
	var got []ProjectPage
	err := projectPages(context.Background(), get, url.Values{}, []string{"Mathematics", "Statistics"}, 0, 0, func(p ProjectPage) error {
		got = append(got, p)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) != 2 || calls[0].Get("wppprojects") != "Mathematics|Statistics" || calls[1].Get("wppcontinue") != "276|1063" {
		t.Fatalf("calls %v", calls)
	}
	want := []string{"Mathematics/Aristotle/GA/High", "Statistics/Arithmetic mean/C/Top", "Mathematics/Analysis of variance/C/"}
	if len(got) != len(want) {
		t.Fatalf("got %+v", got)
	}
	for i, p := range got {
		if s := p.Project + "/" + p.Title + "/" + p.Class + "/" + p.Importance; s != want[i] {
			t.Errorf("page %d is %s, want %s", i, s, want[i])
		}
	}

	calls = nil
	got = nil
	if err := projectPages(context.Background(), get, url.Values{}, []string{"Mathematics"}, -1, 2, func(p ProjectPage) error {
		got = append(got, p)
		return nil
	}); err != nil || len(got) != 2 || len(calls) != 1 || got[1].NS != 14 {
		t.Fatalf("with a limit: %+v, %d calls, %v", got, len(calls), err)
	}
}

func TestProjectPagesError(t *testing.T) {
	get := func(v url.Values, out any) error {
		return json.Unmarshal([]byte(`{"error":{"code":"badvalue","info":"bad project"}}`), out)
	}
	err := projectPages(context.Background(), get, url.Values{}, []string{"Nope"}, 0, 0, func(ProjectPage) error { return nil })
	if err == nil || err.Error() != "badvalue: bad project" {
		t.Fatalf("err %v", err)
	}
}
