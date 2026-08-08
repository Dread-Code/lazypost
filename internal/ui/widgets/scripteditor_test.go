package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

func typed(s *scriptEditor, keys ...tea.KeyMsg) {
	for _, k := range keys {
		s.Update(k)
	}
}

func runeMsg(s string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func TestScriptEditorTyping(t *testing.T) {
	e := newScriptEditor(60, 10, "-- runs before")
	typed(e, runeMsg("return true"))
	if e.Value() != "return true" {
		t.Errorf("typed value = %q", e.Value())
	}
	if e.cursor != len([]rune("return true")) {
		t.Errorf("cursor = %d", e.cursor)
	}
}

func TestScriptEditorBackspaceDeleteEnter(t *testing.T) {
	e := newScriptEditor(60, 10, "")
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
	e := newScriptEditor(60, 10, "")
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
	e := newScriptEditor(60, 10, "")
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

	e := newScriptEditor(60, 10, "")
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
	e := newScriptEditor(60, 5, "-- runs before")
	if !strings.Contains(e.View(), "-- runs before") {
		t.Error("placeholder not rendered")
	}
	e.SetValue("x")
	if strings.Contains(e.View(), "-- runs before") {
		t.Error("placeholder shown while non-empty")
	}
}

func TestScriptEditorScrollFollowsCursor(t *testing.T) {
	e := newScriptEditor(40, 3, "")
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
	e := newScriptEditor(20, 5, "")
	e.SetValue(`return "` + strings.Repeat("x", 200) + `"`)
	out := e.View()
	if !strings.Contains(out, "…") {
		t.Error("wide line should be truncated with an ellipsis")
	}
}
