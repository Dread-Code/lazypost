package script

import (
	"testing"

	"postgo/internal/collection"
)

func TestPreReturnsVars(t *testing.T) {
	req := &collection.Request{Method: "GET", URL: "https://api.test/things"}
	vars, err := Pre(`
		return { token = "abc123" }
	`, req, map[string]string{"host": "https://api.test"})
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
	`, req, nil)
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
	`, req, map[string]string{"host": "api.test"})
	if err != nil {
		t.Fatal(err)
	}
	if vars["resolved"] != "api.test" {
		t.Errorf("env not exposed: %v", vars)
	}
}

func TestPreBlocksDangerousLibs(t *testing.T) {
	req := &collection.Request{URL: "https://api.test"}
	_, err := Pre(`os.execute("rm -rf /")`, req, nil)
	if err == nil {
		t.Fatal("expected error for dangerous os.execute")
	}
}

func TestPostPasses(t *testing.T) {
	msg, err := Post(`return response.status_code == 201`, &collection.Request{}, nil,
		"201 Created", 201, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	if msg != "" {
		t.Errorf("expected pass, got %q", msg)
	}
}

func TestPostFails(t *testing.T) {
	msg, err := Post(`return response.status_code == 200`, &collection.Request{}, nil,
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
	`, &collection.Request{}, nil, "200 OK", 200, nil, `{"error": "boom"}`)
	if err != nil {
		t.Fatal(err)
	}
	if msg == "" {
		t.Error("expected failure for error body")
	}
}
