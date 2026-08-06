package httpclient

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"postgo/internal/collection"
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
