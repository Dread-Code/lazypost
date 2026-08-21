package script

import (
	"context"
	"strings"
	"testing"

	"lazypost/internal/collection"
)

func TestPreReturnsVars(t *testing.T) {
	req := &collection.Request{Method: "GET", URL: "https://api.test/things"}
	vars, err := Pre(`
		return { token = "abc123" }
	`, req, map[string]string{"host": "https://api.test"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if vars["token"] != "abc123" {
		t.Errorf("expected token var, got %v", vars)
	}
	// original request untouched
	if req.URL != "https://api.test/things" {
		t.Errorf("request mutated: %v", req)
	}
}

func TestPreMutatesRequest(t *testing.T) {
	req := &collection.Request{Method: "GET", URL: "https://api.test/things"}
	_, err := Pre(`
		req.headers["X-Timestamp"] = os.time()
		req.body = "hello"
	`, req, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(req.Headers) != 1 || req.Headers[0].Name != "X-Timestamp" {
		t.Errorf("header not applied: %v", req.Headers)
	}
	if req.Body != "hello" {
		t.Errorf("body not applied: %q", req.Body)
	}
}

func TestPreEnvsAvailable(t *testing.T) {
	req := &collection.Request{URL: "https://{{host}}/things"}
	vars, err := Pre(`
		return { resolved = env["host"] }
	`, req, map[string]string{"host": "api.test"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if vars["resolved"] != "api.test" {
		t.Errorf("env not exposed: %v", vars)
	}
}

func TestPreBlocksDangerousLibs(t *testing.T) {
	req := &collection.Request{URL: "https://api.test"}
	_, err := Pre(`os.execute("rm -rf /")`, req, nil, nil)
	if err == nil {
		t.Fatal("expected error for dangerous os.execute")
	}
}

func TestPreRejectsFilesystemAndOutputBaseFunctions(t *testing.T) {
	for _, source := range []string{
		`dofile("/etc/hosts")`,
		`loadfile("/etc/hosts")`,
		`load("return true")`,
		`loadstring("return true")`,
		`require("anything")`,
		`print("not allowed")`,
	} {
		req := &collection.Request{URL: "https://api.test"}
		_, err := Pre(source, req, nil, nil)
		if err == nil {
			t.Errorf("script %q succeeded; dangerous base function should be unavailable", source)
		}
	}
}

func TestHookContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := PreContext(ctx, `while true do end`, &collection.Request{URL: "https://api.test"}, nil, nil)
	if err == nil || !strings.Contains(err.Error(), "context canceled") {
		t.Fatalf("PreContext error = %v, want context cancellation", err)
	}

	ctx, cancel = context.WithCancel(context.Background())
	cancel()
	_, err = PostContext(ctx, `while true do end`, &collection.Request{}, nil, nil, "200 OK", 200, nil, "")
	if err == nil || !strings.Contains(err.Error(), "context canceled") {
		t.Fatalf("PostContext error = %v, want context cancellation", err)
	}
}

func TestNilStoreSetDoesNotPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("nil store panicked: %v", r)
		}
	}()
	if _, err := Pre(`store.set("key", "value")`, &collection.Request{URL: "https://api.test"}, nil, nil); err != nil {
		t.Fatalf("Pre with nil store: %v", err)
	}
}

func TestStoreGetSet(t *testing.T) {
	req := &collection.Request{URL: "https://api.test"}
	store := map[string]string{"token": "seed"}
	if _, err := Pre(`
		store.set("other", "xyz")
		if store.get("token") ~= "seed" then
			error("expected seed token")
		end
	`, req, nil, store); err != nil {
		t.Fatal(err)
	}
	if store["other"] != "xyz" {
		t.Errorf("store.set did not persist: %v", store)
	}

	msg, err := Post(`
		store.set("session", "abc")
		return true
	`, req, nil, store, "200 OK", 200, nil, "")
	if err != nil || msg != "" {
		t.Fatalf("post: %q %v", msg, err)
	}
	if store["session"] != "abc" {
		t.Errorf("post store.set did not persist: %v", store)
	}
}

func TestPostPasses(t *testing.T) {
	msg, err := Post(`return response.status_code == 201`, &collection.Request{}, nil, nil,
		"201 Created", 201, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	if msg != "" {
		t.Errorf("expected pass, got %q", msg)
	}
}

func TestPostFails(t *testing.T) {
	msg, err := Post(`return response.status_code == 200`, &collection.Request{}, nil, nil,
		"500 Internal", 500, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	if msg == "" {
		t.Error("expected post failure message")
	}
}

func TestPostSeesBody(t *testing.T) {
	msg, err := Post(`
		if string.find(response.body, "error", 1, true) then
			return "body contains error"
		end
		return true
	`, &collection.Request{}, nil, nil, "200 OK", 200, nil, `{"error": "boom"}`)
	if err != nil {
		t.Fatal(err)
	}
	if msg == "" {
		t.Error("expected failure for error body")
	}
}

func TestPostWritesStoreEvenOnFailure(t *testing.T) {
	store := map[string]string{}
	msg, err := Post(`
		store.set("extracted", "yes")
		return false
	`, &collection.Request{}, nil, store, "500 Internal", 500, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	if msg == "" {
		t.Error("expected failure message")
	}
	if store["extracted"] != "yes" {
		t.Errorf("expected extraction to persist despite failure: %v", store)
	}
}
