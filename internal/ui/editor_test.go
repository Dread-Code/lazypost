package ui

import (
	"testing"

	"postgo/internal/collection"
)

func TestEditorCarriesHooks(t *testing.T) {
	e := NewEditor(60, 15)
	req := &collection.Request{
		Name:   "thing",
		Method: "GET",
		URL:    "https://api.test/things",
		Pre:    "req.headers['X-Ts'] = os.time()",
		Post:   "return response.status_code == 200",
	}
	e.SetRequest(req, "/col/thing.yaml")
	got := e.Request()
	if got.Pre != req.Pre || got.Post != req.Post {
		t.Errorf("hooks not carried: pre=%q post=%q", got.Pre, got.Post)
	}
	if got.Name != "thing" {
		t.Errorf("name lost: %q", got.Name)
	}

	e.New()
	if e.Request().Pre != "" || e.Request().Post != "" {
		t.Error("New should clear hooks")
	}
}
