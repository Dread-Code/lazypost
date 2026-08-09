package ui

import (
	"testing"

	"github.com/charmbracelet/lipgloss"

	"lazypost/internal/ui/themes"
)

func TestURLBarFocusMovesCursorToEnd(t *testing.T) {
	u := NewURLBar(80)
	u.SetRequest("GET", "https://api.test/things/42")

	// leave the cursor in the middle, as a previous editing session would
	u.url.SetCursor(5)
	if u.url.CursorPos() == len(u.url.Value()) {
		t.Fatal("test setup: cursor should be mid-URL")
	}

	u.Focus()
	if got := u.url.CursorPos(); got != len(u.url.Value()) {
		t.Errorf("cursor after Focus = %d, want %d (end of URL)", got, len(u.url.Value()))
	}
}

// View must not mutate the widget's width — resize happens at the state
// change points (Resize/Focus/Blur/SetRequest), never while rendering.
// Key hints live in the status bar, so focus no longer changes the width.
func TestURLBarViewIsPure(t *testing.T) {
	u := NewURLBar(80)
	u.SetRequest("GET", "https://api.test/things")

	u.Resize(80)
	u.Focus()
	before := u.url.width
	u.View()
	u.View()
	if u.url.width != before {
		t.Errorf("View changed width: before %d, after %d", before, u.url.width)
	}

	u.Blur()
	blurW := u.url.width
	if blurW != before {
		t.Errorf("focus must not change the width (hints live in the status bar): got %d, want %d", blurW, before)
	}
	u.View()
	if u.url.width != blurW {
		t.Errorf("View changed width after blur: %d != %d", u.url.width, blurW)
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

// Editing must keep the URL field's window within the bar so the pill,
// field, and env badge always fit the terminal width.
func TestURLBarFitsTerminalWidth(t *testing.T) {
	u := NewURLBar(80)
	u.SetRequest("GET", "https://api.test/things/{{id}}?page=2&limit=10#top")
	u.SetRight(themes.EnvBadge("env: dev"))
	u.Focus()

	for _, s := range []string{
		"https://api.test/things/{{id}}?page=2&limit=10#top",
		"https://",
		"",
		"{{host}}/api/posts?tag=news&page=2",
	} {
		u.SetRequest("GET", s)
		if w := lipgloss.Width(u.View()); w > u.width {
			t.Errorf("bar %d wide with %q, want <= %d", w, s, u.width)
		}
	}
}
