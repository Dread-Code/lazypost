package app

import (
	"strings"
	"testing"

	"lazypost/internal/collection"
)

func TestCurlLineInterpolates(t *testing.T) {
	req := collection.Request{
		Method: "GET",
		URL:    "{{host}}/api/random",
		Headers: []collection.Header{
			{Name: "Accept", Value: "application/json"},
		},
	}
	got := CurlLine(req, map[string]string{"host": "https://zenquotes.io"})
	for _, want := range []string{"https://zenquotes.io/api/random", "-H 'Accept: application/json'"} {
		if !strings.Contains(got, want) {
			t.Errorf("CurlLine = %q, want contains %q", got, want)
		}
	}
	if strings.Contains(got, "{{host}}") {
		t.Errorf("CurlLine should interpolate host, got %q", got)
	}

	// unknown placeholders still pass through
	got = CurlLine(req, nil)
	if !strings.Contains(got, "{{host}}") {
		t.Errorf("no vars should keep {{host}}, got %q", got)
	}
}
