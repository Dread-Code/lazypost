package app

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"lazypost/internal/collection"
	"lazypost/internal/httpclient"
)

func fakeResponse() *httpclient.Response {
	return &httpclient.Response{
		StatusCode:  http.StatusOK,
		Status:      "200 OK",
		Headers:     http.Header{},
		Body:        []byte(`{"ok":true}`),
		ContentType: "application/json",
	}
}

// captureClient returns a Client that records the request and vars it
// was called with, and returns a canned response.
func captureClient(captured *collection.Request, vars *map[string]string) Client {
	return func(req collection.Request, v map[string]string) (*httpclient.Response, error) {
		*captured = req
		*vars = v
		return fakeResponse(), nil
	}
}

func TestSendNoHooks(t *testing.T) {
	var gotReq collection.Request
	var gotVars map[string]string
	req := collection.Request{Method: "GET", URL: "https://api.test/x"}

	res, err := Send(captureClient(&gotReq, &gotVars), req,
		map[string]string{"host": "https://api.test"}, nil)
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if res.Response == nil {
		t.Fatal("expected response")
	}
	if gotVars["host"] != "https://api.test" {
		t.Errorf("client vars = %v, want env passed through", gotVars)
	}
	if gotReq.URL != req.URL {
		t.Errorf("client req URL = %q, want %q", gotReq.URL, req.URL)
	}
}

func TestSendPreHookVarsInterpolate(t *testing.T) {
	var gotVars map[string]string
	req := collection.Request{
		Method: "GET",
		URL:    "https://api.test/x",
		Pre:    `return {token = "abc"}`,
	}

	_, err := Send(captureClient(&collection.Request{}, &gotVars), req, nil, nil)
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if gotVars["token"] != "abc" {
		t.Errorf("client vars = %v, want pre-hook token", gotVars)
	}
}

func TestSendStoreOverridesEnv(t *testing.T) {
	var gotVars map[string]string
	req := collection.Request{Method: "GET", URL: "https://api.test/x"}

	_, err := Send(captureClient(&collection.Request{}, &gotVars), req,
		map[string]string{"host": "env-host"}, map[string]string{"host": "store-host"})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if gotVars["host"] != "store-host" {
		t.Errorf("client vars host = %q, want store to win", gotVars["host"])
	}
}

func TestSendPreHookErrorDiscardsStore(t *testing.T) {
	req := collection.Request{
		Method: "GET",
		URL:    "https://api.test/x",
		Pre:    `store.set("k", "v") error("boom")`,
	}

	res, err := Send(captureClient(&collection.Request{}, &map[string]string{}), req, nil, map[string]string{})
	if err == nil {
		t.Fatal("expected pre-hook error")
	}
	if !strings.Contains(err.Error(), "pre script") {
		t.Errorf("error = %q, want pre script wrapper", err)
	}
	if res.Store != nil {
		t.Errorf("pre-hook failure should discard store writes, got %v", res.Store)
	}
}

func TestSendExecErrorKeepsStore(t *testing.T) {
	req := collection.Request{
		Method: "GET",
		URL:    "https://api.test/x",
		Pre:    `store.set("k", "v")`,
	}
	fail := func(collection.Request, map[string]string) (*httpclient.Response, error) {
		return nil, errors.New("dial refused")
	}

	res, err := Send(fail, req, nil, map[string]string{})
	if err == nil {
		t.Fatal("expected exec error")
	}
	if res.Store["k"] != "v" {
		t.Errorf("store = %v, want pre-hook writes kept on exec failure", res.Store)
	}
}

func TestSendPostHookFailure(t *testing.T) {
	req := collection.Request{
		Method: "GET",
		URL:    "https://api.test/x",
		Post:   `return false`,
	}

	_, err := Send(captureClient(&collection.Request{}, &map[string]string{}), req, nil, nil)
	if err == nil || !strings.Contains(err.Error(), "post hook") {
		t.Errorf("expected post hook failure, got %v", err)
	}
}

func TestSendPostHookMergesStore(t *testing.T) {
	req := collection.Request{
		Method: "GET",
		URL:    "https://api.test/x",
		Post:   `store.set("token", "abc") return true`,
	}

	res, err := Send(captureClient(&collection.Request{}, &map[string]string{}), req, nil, map[string]string{})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if res.Store["token"] != "abc" {
		t.Errorf("store = %v, want post-hook write", res.Store)
	}
}

func TestSendPostSeesSentRequest(t *testing.T) {
	req := collection.Request{
		Method: "GET",
		URL:    "https://api.test/{{path}}",
		Pre:    `req.url = "https://api.test/users"`,
		Post:   `if req.url == "https://api.test/users" then store.set("sent_url", "yes") end return true`,
	}

	res, err := Send(captureClient(&collection.Request{}, &map[string]string{}), req, nil, map[string]string{})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if res.Store["sent_url"] != "yes" {
		t.Errorf("post hook must see the request as sent, store = %v", res.Store)
	}
}
