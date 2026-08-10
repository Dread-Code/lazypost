package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"lazypost/internal/httpclient"
)

// forceColorProfile pins the renderer to TrueColor so tests can assert on
// ANSI presence deterministically, regardless of TTY detection.
func forceColorProfile(t *testing.T) {
	t.Helper()
	prev := lipgloss.DefaultRenderer().ColorProfile()
	lipgloss.DefaultRenderer().SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.DefaultRenderer().SetColorProfile(prev) })
}

func TestHighlightedBodyColorsValidJSON(t *testing.T) {
	forceColorProfile(t)

	body := highlightedBody(&httpclient.Response{
		Body: []byte(`{"quote":"stay curious","count":42,"ok":true}`),
	}, 60)

	if !strings.Contains(body, "\x1b[") {
		t.Error("valid JSON body should be ANSI-colored, got plain output")
	}
	for _, want := range []string{`"quote"`, "stay curious", "42", "true"} {
		if !strings.Contains(body, want) {
			t.Errorf("colored output lost %q", want)
		}
	}
}

func TestHighlightedBodySkipsColor(t *testing.T) {
	forceColorProfile(t)

	big := &httpclient.Response{
		Body: []byte(`{"big":"` + strings.Repeat("x", maxHighlightBody) + `"}`),
	}
	if got := highlightedBody(big, 60); strings.Contains(got, "\x1b[") {
		t.Error("body above the highlight cap must stay uncolored")
	}

	notJSON := &httpclient.Response{Body: []byte("not json at all")}
	if got := highlightedBody(notJSON, 60); strings.Contains(got, "\x1b[") {
		t.Error("invalid JSON must stay uncolored")
	}
}

// Regression: truncation used to run before highlighting, and a line cut
// mid-string made the whole document fail json.Valid → plain output.
func TestHighlightedBodyLongLinesStayColored(t *testing.T) {
	forceColorProfile(t)

	long := `{"key":"` + strings.Repeat("x", 200) + `"}`
	got := highlightedBody(&httpclient.Response{Body: []byte(long)}, 40)
	if !strings.Contains(got, "\x1b[") {
		t.Fatal("valid JSON with a line wider than the pane must still be colored")
	}
	if !strings.Contains(got, "…") {
		t.Error("long line should be truncated with an ellipsis")
	}
}
