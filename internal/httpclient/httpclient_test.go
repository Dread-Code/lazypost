package httpclient

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"lazypost/internal/collection"
)

func TestExec(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer tok123" {
			t.Errorf("missing bearer auth, got %q", r.Header.Get("Authorization"))
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("content type = %q", r.Header.Get("Content-Type"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{"ok": true, "path": r.URL.Path})
	}))
	defer srv.Close()

	req := collection.Request{
		Method: "POST",
		URL:    srv.URL + "/things",
		Headers: []collection.Header{
			{Name: "Content-Type", Value: "application/json"},
		},
		Auth: &collection.Auth{Type: "bearer", Token: "tok123"},
		Body: `{"a":1}`,
	}

	res, err := Exec(req, nil)
	if err != nil {
		t.Fatalf("Exec: %v", err)
	}
	if res.StatusCode != http.StatusCreated {
		t.Errorf("status = %d", res.StatusCode)
	}
	if res.FormattedBody() == "" {
		t.Error("empty formatted body")
	}
	if res.Summary() == "" {
		t.Error("empty summary")
	}
	if res.URL != srv.URL+"/things" {
		t.Errorf("executed URL = %q", res.URL)
	}
}

func TestExecAPIKeyInQuery(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("api_key") != "secret" {
			t.Errorf("query api_key = %q", r.URL.Query().Get("api_key"))
		}
		w.Write([]byte("ok"))
	}))
	defer srv.Close()

	req := collection.Request{
		Method: "GET",
		URL:    srv.URL,
		Auth:   &collection.Auth{Type: "apikey", KeyName: "api_key", KeyValue: "secret", KeyIn: "query"},
	}
	res, err := Exec(req, nil)
	if err != nil {
		t.Fatalf("Exec: %v", err)
	}
	if string(res.Body) != "ok" {
		t.Errorf("body = %q", res.Body)
	}
}

func TestExecMergesQueryParams(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("from_url") != "1" {
			t.Errorf("from_url = %q", q.Get("from_url"))
		}
		// explicit params are added (duplicates kept)
		if got := q.Get("tag"); got != "news" {
			t.Errorf("tag = %q", got)
		}
		// apikey-in-query overrides the colliding explicit param
		if got := q.Get("api_key"); got != "secret" {
			t.Errorf("api_key = %q, want apikey to override", got)
		}
		w.Write([]byte("ok"))
	}))
	defer srv.Close()

	req := collection.Request{
		Method: "GET",
		URL:    srv.URL + "/x?from_url=1",
		Query: []collection.Param{
			{Name: "tag", Value: "news"},
			{Name: "api_key", Value: "shadowed"},
		},
		Auth: &collection.Auth{Type: "apikey", KeyName: "api_key", KeyValue: "secret", KeyIn: "query"},
	}
	res, err := Exec(req, nil)
	if err != nil {
		t.Fatalf("Exec: %v", err)
	}
	if string(res.Body) != "ok" {
		t.Errorf("body = %q", res.Body)
	}
	// the executed URL reflects the merged query params
	for _, want := range []string{"api_key=secret", "tag=news", "from_url=1"} {
		if !strings.Contains(res.URL, want) {
			t.Errorf("executed URL %q should contain %q", res.URL, want)
		}
	}
}

func TestExecInterpolatesQueryParams(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("id"); got != "42" {
			t.Errorf("id = %q, want interpolated 42", got)
		}
		w.Write([]byte("ok"))
	}))
	defer srv.Close()

	req := collection.Request{
		Method: "GET",
		URL:    srv.URL + "/x",
		Query:  []collection.Param{{Name: "id", Value: "{{user_id}}"}},
	}
	if _, err := Exec(req, map[string]string{"user_id": "42"}); err != nil {
		t.Fatalf("Exec: %v", err)
	}
}

func TestExecUnresolvedURLPlaceholder(t *testing.T) {
	req := collection.Request{Method: "GET", URL: "{{host}}/api/today"}
	_, err := Exec(req, nil)
	if err == nil {
		t.Fatal("expected error for unresolved URL placeholder")
	}
	for _, want := range []string{"{{host}}", "ctrl+e"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q should mention %q", err, want)
		}
	}
}

func TestExecInterpolatesVars(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-User") != "alice" {
			t.Errorf("X-User = %q", r.Header.Get("X-User"))
		}
		w.Write([]byte("ok"))
	}))
	defer srv.Close()

	req := collection.Request{
		Method:  "GET",
		URL:     "{{host}}/x",
		Headers: []collection.Header{{Name: "X-User", Value: "{{user}}"}},
	}
	if _, err := Exec(req, map[string]string{"host": srv.URL, "user": "alice"}); err != nil {
		t.Fatalf("Exec: %v", err)
	}
}

func TestFormattedBodyNonJSON(t *testing.T) {
	r := &Response{Body: []byte("plain text")}
	if r.FormattedBody() != "plain text" {
		t.Errorf("got %q", r.FormattedBody())
	}
}

func TestFormattedHeadersLeadsWithExecutedURL(t *testing.T) {
	r := &Response{
		URL:     "https://api.test/things?tag=news",
		Headers: http.Header{"Content-Type": []string{"application/json"}},
	}
	got := r.FormattedHeaders()
	if !strings.HasPrefix(got, "URL: https://api.test/things?tag=news\n\n") {
		t.Errorf("headers should lead with the executed URL, got:\n%s", got)
	}
	// real headers still follow, sorted
	if !strings.Contains(got, "Content-Type: application/json") {
		t.Errorf("missing header line, got:\n%s", got)
	}
}
