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

// View must not mutate the widget's width — resize happens at the state
// change points (Resize/Focus/Blur/SetRequest), never while rendering.
func TestURLBarViewIsPure(t *testing.T) {
	u := NewURLBar(80)
	u.SetRequest("GET", "https://api.test/things")

	u.Resize(80)
	u.Focus()
	before := u.url.Width
	u.View()
	u.View()
	if u.url.Width != before {
		t.Errorf("View changed width: before %d, after %d", before, u.url.Width)
	}

	u.Blur()
	blurW := u.url.Width
	if blurW == before {
		t.Errorf("Blur should widen the input (hint room released), got %d", blurW)
	}
	u.View()
	if u.url.Width != blurW {
		t.Errorf("View changed width after blur: %d != %d", u.url.Width, blurW)
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
