package render

import (
	"testing"

	"lazypost/internal/collection"
)

func TestApply(t *testing.T) {
	vars := map[string]string{"host": "https://api.test", "id": "42"}

	cases := []struct {
		in, want string
	}{
		{"{{host}}/users", "https://api.test/users"},
		{"{{ host }}/users/{{id}}", "https://api.test/users/42"},
		{"{{host}}{{id}}", "https://api.test42"},
		{"no placeholders", "no placeholders"},
		{"{{missing}} stays", "{{missing}} stays"},
		{"", ""},
	}
	for _, c := range cases {
		if got := Apply(c.in, vars); got != c.want {
			t.Errorf("Apply(%q) = %q, want %q", c.in, got, c.want)
		}
	}

	if got := Apply("{{host}}", nil); got != "{{host}}" {
		t.Errorf("Apply with nil vars should be identity, got %q", got)
	}
}

func TestUnresolved(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"{{host}}/api/today", []string{"host"}},
		{"{{host}}/{{ id }}/x", []string{"host", "id"}},
		{"https://api.test/users", nil},
		{"", nil},
	}
	for _, c := range cases {
		got := Unresolved(c.in)
		if len(got) != len(c.want) {
			t.Errorf("Unresolved(%q) = %v, want %v", c.in, got, c.want)
			continue
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Errorf("Unresolved(%q) = %v, want %v", c.in, got, c.want)
			}
		}
	}
}

func TestRequest(t *testing.T) {
	req := collection.Request{
		Method: "POST",
		URL:    "{{host}}/posts",
		Query: []collection.Param{
			{Name: "tag", Value: "{{tag}}"},
		},
		Headers: []collection.Header{
			{Name: "X-Token", Value: "{{token}}"},
			{Name: "Static", Value: "yes"},
		},
		Auth: &collection.Auth{Type: "bearer", Token: "{{token}}"},
		Body: `{"id": {{id}}}`,
	}
	vars := map[string]string{
		"host":  "https://api.test",
		"token": "abc123",
		"id":    "7",
		"tag":   "news",
	}

	out := Request(req, vars)

	if out.URL != "https://api.test/posts" {
		t.Errorf("URL = %q", out.URL)
	}
	if out.Query[0].Value != "news" {
		t.Errorf("Query[0].Value = %q", out.Query[0].Value)
	}
	if out.Headers[0].Value != "abc123" || out.Headers[1].Value != "yes" {
		t.Errorf("Headers = %+v", out.Headers)
	}
	if out.Auth.Token != "abc123" {
		t.Errorf("Auth.Token = %q", out.Auth.Token)
	}
	if out.Body != `{"id": 7}` {
		t.Errorf("Body = %q", out.Body)
	}

	if req.URL != "{{host}}/posts" {
		t.Error("original request was mutated")
	}
	if req.Auth.Token != "{{token}}" {
		t.Error("original auth was mutated")
	}
	if req.Query[0].Value != "{{tag}}" {
		t.Error("original query was mutated")
	}
}

func TestFormatJSON(t *testing.T) {
	// valid JSON pretty-prints
	if got := FormatJSON(`{"a":1,"b":[true,null]}`); got != "{\n  \"a\": 1,\n  \"b\": [\n    true,\n    null\n  ]\n}" {
		t.Errorf("valid JSON not formatted: %q", got)
	}
	// already-formatted stays put (idempotent)
	formatted := "{\n  \"a\": 1\n}"
	if got := FormatJSON(formatted); got != formatted {
		t.Errorf("idempotence broken: %q", got)
	}
	// placeholder in a value position: formats around the placeholder
	got := FormatJSON(`{"title": "{{title}}", "userId": {{user_id}}}`)
	want := "{\n  \"title\": \"{{title}}\",\n  \"userId\": {{user_id}}\n}"
	if got != want {
		t.Errorf("placeholder body:\n got %q\nwant %q", got, want)
	}
	// multiple placeholders survive in order
	got = FormatJSON(`{"a": {{x}}, "b": {{y}}}`)
	want = "{\n  \"a\": {{x}},\n  \"b\": {{y}}\n}"
	if got != want {
		t.Errorf("multi-placeholder body:\n got %q\nwant %q", got, want)
	}
	// genuinely invalid JSON passes through untouched
	for _, body := range []string{`{"a": 1`, "plain text", `{"userId": 1,`} {
		if got := FormatJSON(body); got != body {
			t.Errorf("invalid body %q changed to %q", body, got)
		}
	}
	// empty input is untouched
	if got := FormatJSON(""); got != "" {
		t.Errorf("empty input changed to %q", got)
	}
}
