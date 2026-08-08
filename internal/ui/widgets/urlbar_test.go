package ui

import (
	"testing"
)

func TestURLBarFocusMovesCursorToEnd(t *testing.T) {
	u := NewURLBar(80)
	u.SetRequest("GET", "https://api.test/things/42")

	// leave the cursor in the middle, as a previous editing session would
	u.url.SetCursor(5)
	if u.url.Position() == len(u.url.Value()) {
		t.Fatal("test setup: cursor should be mid-URL")
	}

	u.Focus()
	if got := u.url.Position(); got != len(u.url.Value()) {
		t.Errorf("cursor after Focus = %d, want %d (end of URL)", got, len(u.url.Value()))
	}
}

func TestURLBarBlank(t *testing.T) {
	u := NewURLBar(80)
	if u.URL() != "" {
		t.Errorf("fresh bar URL = %q", u.URL())
	}
	u.SetRequest("", "https://api.test/x")
	if u.Method() != "GET" {
		t.Errorf("empty method should fall back to GET, got %q", u.Method())
	}
	u.New()
	if u.URL() != "" || u.Method() != "GET" {
		t.Errorf("New should blank the bar, got %q %q", u.Method(), u.URL())
	}
}
