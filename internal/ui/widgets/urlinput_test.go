package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// stripStyles keeps visible text so the tests assert structure, not ANSI.
func stripStyles(s string) string {
	var b strings.Builder
	in := false
	for _, r := range s {
		if r == 0x1b {
			in = true
			continue
		}
		if in {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				in = false
			}
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// tokenTexts returns the visible text of each token.
func tokenTexts(s string) []string {
	var out []string
	for _, t := range tokenizeURL(s) {
		if t.text != "" {
			out = append(out, t.text)
		}
	}
	return out
}

func TestTokenizeURLStructure(t *testing.T) {
	tests := []struct {
		in   string
		want []string
	}{
		{
			"https://api.test.com/v1/things?page=2&limit=10#top",
			[]string{"https://", "api.test.com", "/v1/things", "?", "page", "=", "2", "&", "limit", "=", "10", "#top"},
		},
		{
			"{{host}}/api/posts",
			[]string{"{{host}}", "/api/posts"},
		},
		{
			"http://user:pass@example.com:8080/",
			[]string{"http://", "user:pass@", "example.com", ":8080", "/"},
		},
		{
			"https://example.com",
			[]string{"https://", "example.com"},
		},
		{
			"example.com/x",
			[]string{"example.com", "/x"},
		},
	}
	for _, tc := range tests {
		got := tokenTexts(tc.in)
		if strings.Join(got, "") != strings.Join(tc.want, "") {
			t.Errorf("tokens for %q = %q, want %q", tc.in, got, tc.want)
		}
		if len(got) != len(tc.want) {
			t.Errorf("token count for %q = %d, want %d", tc.in, len(got), len(tc.want))
		}
	}
}

func TestTokenizeURLStylesParts(t *testing.T) {
	prev := lipgloss.DefaultRenderer().ColorProfile()
	lipgloss.DefaultRenderer().SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.DefaultRenderer().SetColorProfile(prev) })

	toks := tokenizeURL("https://user:pass@api.test.com:8080/v1?key={{token}}&page=2#top")
	styled := func(text string) bool {
		for _, tok := range toks {
			if tok.text == text {
				return strings.Contains(tok.style.Render(tok.text), "\x1b[")
			}
		}
		t.Fatalf("token %q not found", text)
		return false
	}
	if !styled("https://") {
		t.Error("scheme should be styled")
	}
	if !styled("api.test.com") {
		t.Error("host should be styled")
	}
	if !styled("{{token}}") {
		t.Error("placeholder should be styled")
	}
	// default-foreground parts (path, query values) stay unstyled
	if styled("/v1") {
		t.Error("path should stay at the default foreground")
	}
	if styled("2") {
		t.Error("query value should stay at the default foreground")
	}
}

func TestURLEditUpdatesTokensLive(t *testing.T) {
	f := newURLField("placeholder")
	f.width = 80
	f.focus()
	f.insert("https://api.test/x")
	if got := stripStyles(f.View()); !strings.Contains(got, "https://api.test/x") {
		t.Errorf("typed URL lost: %q", got)
	}
	// extend the URL with a query; the view must follow and color the new part
	f.insert("?page=1")
	got := stripStyles(f.View())
	for _, want := range []string{"https://api.test/x?page=1"} {
		if !strings.Contains(got, want) {
			t.Errorf("live edit lost %q: %q", want, got)
		}
	}
}

func TestURLFieldEditingKeys(t *testing.T) {
	f := newURLField("p")
	f.width = 40
	f.focus()

	f.insert("abc")
	if f.Value() != "abc" || f.CursorPos() != 3 {
		t.Fatalf("insert: value %q cursor %d", f.Value(), f.CursorPos())
	}

	f.moveCursor(-1) // now at index 2
	f.insert("X")
	if f.Value() != "abXc" {
		t.Errorf("insert mid-string = %q", f.Value())
	}

	f.deleteBefore() // removes X
	if f.Value() != "abc" {
		t.Errorf("deleteBefore = %q", f.Value())
	}

	f.deleteAt() // removes c
	if f.Value() != "ab" {
		t.Errorf("deleteAt = %q", f.Value())
	}

	f.SetCursor(0)
	f.insert("zz")
	if f.Value() != "zzab" {
		t.Errorf("insert at start = %q", f.Value())
	}

	f.SetCursor(len(f.Value()))
	f.deleteToStart()
	if f.Value() != "" {
		t.Errorf("deleteToStart = %q", f.Value())
	}

	f.insert("hello world")
	f.deleteWordBackward()
	if f.Value() != "hello" {
		t.Errorf("deleteWordBackward = %q", f.Value())
	}

	f.SetCursor(0)
	f.deleteToEnd()
	if f.Value() != "" {
		t.Errorf("deleteToEnd = %q", f.Value())
	}
}

func (f *urlField) focus() {
	f.focused = true
	f.cursor = len(f.runes())
	f.ensureVisible()
}
