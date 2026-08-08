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

func TestTruncateRunesAnsi(t *testing.T) {
	cases := []struct {
		in   string
		n    int
		want string
	}{
		{"plain text", 5, "plai…"},
		{"plain", 10, "plain"},
		{"", 5, ""},
		{"x", 1, "…"},
		{"\x1b[31mred\x1b[0m", 3, "\x1b[31mred\x1b[0m"},
		{"\x1b[31mred\x1b[0m", 9, "\x1b[31mred\x1b[0m"},
		{"\x1b[31mred\x1b[0m", 2, "\x1b[31mr…\x1b[0m"},
		{"\x1b[38;2;1;2;3mA\x1b[0m", 1, "…"},
		{"\x1b[38;2;1;2;3mAB\x1b[0m", 2, "\x1b[38;2;1;2;3mAB\x1b[0m"},
		{"\x1b[38;2;1;2;3mABC\x1b[0m", 2, "\x1b[38;2;1;2;3mA…\x1b[0m"},
	}
	for _, c := range cases {
		if got := truncateRunesAnsi(c.in, c.n); got != c.want {
			t.Errorf("truncateRunesAnsi(%q, %d) = %q, want %q", c.in, c.n, got, c.want)
		}
	}
}
