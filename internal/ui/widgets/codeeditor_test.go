package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

func typed(s *codeEditor, keys ...tea.KeyMsg) {
	for _, k := range keys {
		s.Update(k)
	}
}

func runeMsg(s string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func TestScriptEditorTyping(t *testing.T) {
	e := newCodeEditor(60, 10, "-- runs before", highlightLuaLine)
	typed(e, runeMsg("return true"))
	if e.Value() != "return true" {
		t.Errorf("typed value = %q", e.Value())
	}
	if e.cursor != len([]rune("return true")) {
		t.Errorf("cursor = %d", e.cursor)
	}
}

// Regression: bubbletea reports a lone space as tea.KeySpace (not
// KeyRunes), and the widget must insert it.
func TestScriptEditorSpaceKey(t *testing.T) {
	e := newCodeEditor(60, 10, "", highlightLuaLine)
	e.Update(tea.KeyMsg{Type: tea.KeySpace, Runes: []rune(" ")})
	if e.Value() != " " {
		t.Errorf("space key inserted %q", e.Value())
	}
	typed(e, runeMsg("x"))
	e.Update(tea.KeyMsg{Type: tea.KeySpace, Runes: []rune(" ")})
	if e.Value() != " x " {
		t.Errorf("space mid-text inserted %q", e.Value())
	}
}

func TestScriptEditorBackspaceDeleteEnter(t *testing.T) {
	e := newCodeEditor(60, 10, "", highlightLuaLine)
	typed(e, runeMsg("abc"))
	e.cursor = 1
	typed(e, tea.KeyMsg{Type: tea.KeyBackspace})
	if e.Value() != "bc" {
		t.Errorf("after backspace = %q", e.Value())
	}
	if e.cursor != 0 {
		t.Errorf("cursor after backspace = %d", e.cursor)
	}
	typed(e, tea.KeyMsg{Type: tea.KeyDelete})
	if e.Value() != "c" {
		t.Errorf("after delete = %q", e.Value())
	}
	typed(e, tea.KeyMsg{Type: tea.KeyEnter})
	if e.Value() != "\nc" {
		t.Errorf("after enter = %q", e.Value())
	}
}

func TestScriptEditorCursorLines(t *testing.T) {
	e := newCodeEditor(60, 10, "", highlightLuaLine)
	typed(e, runeMsg("ab\ncd\ne"))
	// cursor at end: line 2 ("e"), col 1
	start, end := e.lineBounds([]rune(e.Value()), e.cursor)
	if e.Value()[start:end] != "e" {
		t.Errorf("bounds line = %q", e.Value()[start:end])
	}
	// up: line 1 ("cd"), col clamped to 1
	typed(e, tea.KeyMsg{Type: tea.KeyUp})
	st, en := e.lineBounds([]rune(e.Value()), e.cursor)
	if e.Value()[st:en] != "cd" {
		t.Errorf("after up, line = %q", e.Value()[st:en])
	}
	if e.cursor-st != 1 {
		t.Errorf("after up, col = %d", e.cursor-st)
	}
	// down: back to "e"
	typed(e, tea.KeyMsg{Type: tea.KeyDown})
	st, en = e.lineBounds([]rune(e.Value()), e.cursor)
	if e.Value()[st:en] != "e" {
		t.Errorf("after down, line = %q", e.Value()[st:en])
	}
	// home/end
	e.cursor = len([]rune(e.Value()))
	typed(e, tea.KeyMsg{Type: tea.KeyHome})
	st, _ = e.lineBounds([]rune(e.Value()), e.cursor)
	if e.cursor != st {
		t.Errorf("home cursor = %d, line start = %d", e.cursor, st)
	}
	typed(e, tea.KeyMsg{Type: tea.KeyEnd})
	_, en = e.lineBounds([]rune(e.Value()), e.cursor)
	if e.cursor != en {
		t.Errorf("end cursor = %d, line end = %d", e.cursor, en)
	}
}

func TestScriptEditorSetValueClamps(t *testing.T) {
	e := newCodeEditor(60, 10, "", highlightLuaLine)
	e.cursor = 500
	e.SetValue("short")
	if e.cursor != len([]rune("short")) {
		t.Errorf("cursor not clamped: %d", e.cursor)
	}
	if e.Value() != "short" {
		t.Errorf("value = %q", e.Value())
	}
}

func TestScriptEditorView(t *testing.T) {
	prev := lipgloss.DefaultRenderer().ColorProfile()
	lipgloss.DefaultRenderer().SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.DefaultRenderer().SetColorProfile(prev) })

	e := newCodeEditor(60, 10, "", highlightLuaLine)
	e.SetValue("local x = 1")
	e.cursor = len([]rune(e.Value()))
	e.Focus()
	out := e.View()
	if !strings.Contains(out, "\x1b[") {
		t.Error("highlighted script should contain ANSI")
	}
	for _, want := range []string{"local", "x", "1"} {
		if !strings.Contains(out, want) {
			t.Errorf("view lost %q:\n%s", want, out)
		}
	}
	// cursor block is a reverse-video cell
	if !strings.Contains(out, "\x1b[7m") {
		t.Errorf("cursor block missing:\n%q", out)
	}
	// line number gutter
	if !strings.Contains(out, "1 ") {
		t.Errorf("gutter missing:\n%q", out)
	}
}

func TestScriptEditorPlaceholder(t *testing.T) {
	e := newCodeEditor(60, 5, "-- runs before", highlightLuaLine)
	if !strings.Contains(e.View(), "-- runs before") {
		t.Error("placeholder not rendered")
	}
	e.SetValue("x")
	if strings.Contains(e.View(), "-- runs before") {
		t.Error("placeholder shown while non-empty")
	}
}

func TestScriptEditorScrollFollowsCursor(t *testing.T) {
	e := newCodeEditor(40, 3, "", highlightLuaLine)
	e.SetValue("a\nb\nc\nd\ne")
	if e.top != 0 {
		t.Errorf("initial top = %d", e.top)
	}
	// move cursor to the last line; the window should slide
	for i := 0; i < 4; i++ {
		e.Update(tea.KeyMsg{Type: tea.KeyDown})
	}
	if e.top != 2 {
		t.Errorf("top after scrolling = %d", e.top)
	}
	out := e.View()
	if !strings.Contains(out, "e") || strings.Contains(out, "a") {
		t.Errorf("window should show the last lines, got:\n%s", out)
	}
}

func TestScriptEditorWideLineTruncated(t *testing.T) {
	e := newCodeEditor(20, 5, "", highlightLuaLine)
	e.SetValue(`return "` + strings.Repeat("x", 200) + `"`)
	out := e.View()
	if !strings.Contains(out, "…") {
		t.Error("wide line should be truncated with an ellipsis")
	}
}

// The request Body tab is a codeEditor wired to the JSON fragment
// highlighter; a valid JSON body must render with token colors.
func TestCodeEditorJSONBodyHighlighted(t *testing.T) {
	prev := lipgloss.DefaultRenderer().ColorProfile()
	lipgloss.DefaultRenderer().SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.DefaultRenderer().SetColorProfile(prev) })

	e := newCodeEditor(60, 10, "", highlightJSONLine)
	e.SetValue(`{"title": "hi", "count": 1, "ok": true}`)
	e.cursor = len([]rune(e.Value()))
	e.Focus()
	out := e.View()
	if !strings.Contains(out, "\x1b[") {
		t.Errorf("JSON body should be highlighted, got:\n%s", out)
	}
	for _, want := range []string{`"title"`, `"hi"`, "1", "true"} {
		if !strings.Contains(out, want) {
			t.Errorf("view lost %q:\n%s", want, out)
		}
	}
}

// Regression: while editing, the buffer is usually not valid JSON yet —
// the fragment highlighter must still color it (no identity gate).
func TestCodeEditorJSONBodyFragmentWhileEditing(t *testing.T) {
	prev := lipgloss.DefaultRenderer().ColorProfile()
	lipgloss.DefaultRenderer().SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.DefaultRenderer().SetColorProfile(prev) })

	e := newCodeEditor(60, 10, "", highlightJSONLine)
	e.SetValue(`{"title": "`) // unterminated string: invalid JSON
	e.cursor = len([]rune(e.Value()))
	e.Focus()
	out := e.View()
	if !strings.Contains(out, "\x1b[") {
		t.Errorf("fragment mode should color invalid-but-typed JSON, got:\n%s", out)
	}
	// typing still works on the highlighted editor
	e.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("hi")})
	if !strings.Contains(e.Value(), `"hi`) {
		t.Errorf("typing into fragment body failed, value = %q", e.Value())
	}
}
