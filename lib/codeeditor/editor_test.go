package codeeditor

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// markerHighlighter wraps every line in angle brackets so tests can
// assert that painted output flows through the editor untouched. Cuts
// produce pieces whose concatenation is the marked line.
type markerHighlighter struct{}

func (markerHighlighter) Lines(src string) []string {
	lines := strings.Split(src, "\n")
	out := make([]string, len(lines))
	for i, l := range lines {
		out[i] = "<" + l + ">"
	}
	return out
}

func (markerHighlighter) Split(prefix, line string, cuts ...int) []string {
	if len(cuts) == 0 {
		return []string{"<" + line + ">"}
	}
	pieces := make([]string, 0, len(cuts)+1)
	prev := 0
	for _, c := range cuts {
		if c < 0 {
			c = 0
		}
		if c > len(line) {
			c = len(line)
		}
		pieces = append(pieces, line[prev:c])
		prev = c
	}
	pieces = append(pieces, line[prev:])
	pieces[0] = "<" + pieces[0]
	pieces[len(pieces)-1] += ">"
	return pieces
}

func typed(e *Editor, keys ...tea.KeyMsg) {
	for _, k := range keys {
		e.Update(k)
	}
}

func runeMsg(s string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func TestEditorTyping(t *testing.T) {
	e := New(60, 10, "-- runs before", markerHighlighter{})
	typed(e, runeMsg("return true"))
	if e.Value() != "return true" {
		t.Errorf("typed value = %q", e.Value())
	}
	if e.Cursor() != len([]rune("return true")) {
		t.Errorf("cursor = %d", e.Cursor())
	}
}

// Regression: bubbletea reports a lone space as tea.KeySpace (not
// KeyRunes), and the editor must insert it.
func TestEditorSpaceKey(t *testing.T) {
	e := New(60, 10, "", markerHighlighter{})
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

func TestEditorBackspaceDeleteEnter(t *testing.T) {
	e := New(60, 10, "", markerHighlighter{})
	typed(e, runeMsg("abc"))
	e.SetCursor(1)
	typed(e, tea.KeyMsg{Type: tea.KeyBackspace})
	if e.Value() != "bc" {
		t.Errorf("after backspace = %q", e.Value())
	}
	if e.Cursor() != 0 {
		t.Errorf("cursor after backspace = %d", e.Cursor())
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

func TestEditorCursorLines(t *testing.T) {
	e := New(60, 10, "", markerHighlighter{})
	typed(e, runeMsg("ab\ncd\ne"))
	// cursor at end: line 2 ("e"), col 1
	start, end := e.lineBounds([]rune(e.Value()), e.Cursor())
	if e.Value()[start:end] != "e" {
		t.Errorf("bounds line = %q", e.Value()[start:end])
	}
	// up: line 1 ("cd"), col clamped to 1
	typed(e, tea.KeyMsg{Type: tea.KeyUp})
	st, en := e.lineBounds([]rune(e.Value()), e.Cursor())
	if e.Value()[st:en] != "cd" {
		t.Errorf("after up, line = %q", e.Value()[st:en])
	}
	if e.Cursor()-st != 1 {
		t.Errorf("after up, col = %d", e.Cursor()-st)
	}
	// down: back to "e"
	typed(e, tea.KeyMsg{Type: tea.KeyDown})
	st, en = e.lineBounds([]rune(e.Value()), e.Cursor())
	if e.Value()[st:en] != "e" {
		t.Errorf("after down, line = %q", e.Value()[st:en])
	}
	// home/end
	e.SetCursor(len([]rune(e.Value())))
	typed(e, tea.KeyMsg{Type: tea.KeyHome})
	st, _ = e.lineBounds([]rune(e.Value()), e.Cursor())
	if e.Cursor() != st {
		t.Errorf("home cursor = %d, line start = %d", e.Cursor(), st)
	}
	typed(e, tea.KeyMsg{Type: tea.KeyEnd})
	_, en = e.lineBounds([]rune(e.Value()), e.Cursor())
	if e.Cursor() != en {
		t.Errorf("end cursor = %d, line end = %d", e.Cursor(), en)
	}
}

func TestEditorSetValueClamps(t *testing.T) {
	e := New(60, 10, "", markerHighlighter{})
	e.cursor = 500 // internal invariant: an out-of-range cursor is legal state
	e.SetValue("short")
	if e.Cursor() != len([]rune("short")) {
		t.Errorf("cursor not clamped: %d", e.Cursor())
	}
	if e.Value() != "short" {
		t.Errorf("value = %q", e.Value())
	}
}

func TestEditorView(t *testing.T) {
	prev := lipgloss.DefaultRenderer().ColorProfile()
	lipgloss.DefaultRenderer().SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.DefaultRenderer().SetColorProfile(prev) })

	e := New(60, 10, "", markerHighlighter{})
	e.SetValue("local x = 1")
	e.SetCursor(len([]rune(e.Value())))
	e.Focus()
	out := e.View()
	// the highlighter's paint flows through; the cursor block splits the
	// cursor line, so assert the marker'd halves
	for _, want := range []string{"<local x = 1", ">", "1 "} {
		if !strings.Contains(out, want) {
			t.Errorf("view lost %q:\n%q", want, out)
		}
	}
	// cursor block is a reverse-video cell
	if !strings.Contains(out, "\x1b[7m") {
		t.Errorf("cursor block missing:\n%q", out)
	}
}

// A StyleProvider is evaluated per render, so a style change (theme
// switch) lands on the next frame.
func TestEditorStyleProviderReapplies(t *testing.T) {
	prev := lipgloss.DefaultRenderer().ColorProfile()
	lipgloss.DefaultRenderer().SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.DefaultRenderer().SetColorProfile(prev) })

	e := New(60, 5, "", markerHighlighter{})
	e.SetValue("x")
	red := lipgloss.NewStyle().Foreground(lipgloss.Color("#ff0000"))
	e.SetStyleProvider(func() Style {
		return Style{Gutter: red}
	})
	if !strings.Contains(e.View(), "\x1b[38;2;255;0;0m") {
		t.Error("style provider not applied to the gutter")
	}
	// clearing the provider falls back to defaults
	e.SetStyleProvider(nil)
	if strings.Contains(e.View(), "\x1b[38;2;255;0;0m") {
		t.Error("default style still colored after clearing the provider")
	}
}

func TestEditorPlaceholder(t *testing.T) {
	e := New(60, 5, "-- runs before", markerHighlighter{})
	if !strings.Contains(e.View(), "-- runs before") {
		t.Error("placeholder not rendered")
	}
	e.SetValue("x")
	if strings.Contains(e.View(), "-- runs before") {
		t.Error("placeholder shown while non-empty")
	}
}

func TestEditorScrollFollowsCursor(t *testing.T) {
	e := New(40, 3, "", markerHighlighter{})
	e.SetValue("a\nb\nc\nd\ne")
	if e.Top() != 0 {
		t.Errorf("initial top = %d", e.Top())
	}
	// move cursor to the last line; the window should slide
	for i := 0; i < 4; i++ {
		e.Update(tea.KeyMsg{Type: tea.KeyDown})
	}
	if e.Top() != 2 {
		t.Errorf("top after scrolling = %d", e.Top())
	}
	out := e.View()
	if !strings.Contains(out, "<e>") || strings.Contains(out, "<a>") {
		t.Errorf("window should show the last lines, got:\n%s", out)
	}
}

func TestEditorWideLineTruncated(t *testing.T) {
	e := New(20, 5, "", markerHighlighter{})
	e.SetValue(`return "` + strings.Repeat("x", 200) + `"`)
	out := e.View()
	if !strings.Contains(out, "…") {
		t.Error("wide line should be truncated with an ellipsis")
	}
}

// prefixRecorder captures the prefix the editor hands to the
// highlighter for the cursor line.
type prefixRecorder struct {
	got string
	hl  Highlighter
}

func (p *prefixRecorder) Lines(src string) []string { return p.hl.Lines(src) }

func (p *prefixRecorder) Split(prefix, line string, cuts ...int) []string {
	p.got = prefix
	return p.hl.Split(prefix, line, cuts...)
}

// The cursor-line prefix must be the exact preceding text including the
// trailing newline — tokens that terminate at a line boundary (a Lua
// `--` comment, a single-line string) would otherwise swallow the
// cursor line and color it as a comment.
func TestEditorCursorPrefixIncludesNewline(t *testing.T) {
	rec := &prefixRecorder{hl: markerHighlighter{}}
	e := New(60, 10, "", rec)
	e.SetValue("-- a comment\nreq.headers[\"X\"] = \"y\"")
	e.SetCursor(len([]rune("-- a comment\n"))) // cursor on line 2, col 0
	e.Focus()
	e.View()
	if rec.got != "-- a comment\n" {
		t.Errorf("cursor-line prefix = %q, want %q (trailing newline)", rec.got, "-- a comment\n")
	}
	// the first line gets no prefix at all
	rec.got = ""
	e.SetCursor(0)
	e.View()
	if rec.got != "" {
		t.Errorf("first-line prefix = %q, want empty", rec.got)
	}
}

// The window always renders exactly e.height rows: rows beyond the
// buffer are blank, so consumers never need to pad (the mode footer in
// lazypost relies on it).
func TestEditorViewFillsHeight(t *testing.T) {
	e := New(60, 5, "", markerHighlighter{})
	e.SetValue("a\nb")
	out := e.View()
	if rows := strings.Count(out, "\n") + 1; rows != 5 {
		t.Errorf("view has %d rows, want 5 (height):\n%q", rows, out)
	}
	// the placeholder path fills too
	e.SetValue("")
	out = e.View()
	if rows := strings.Count(out, "\n") + 1; rows != 5 {
		t.Errorf("placeholder view has %d rows, want 5:\n%q", rows, out)
	}
	// a multi-line placeholder must not overflow the window: it fills
	// exactly height rows (a 2-line placeholder in height 5)
	e = New(60, 5, "line one\nline two", markerHighlighter{})
	out = e.View()
	if rows := strings.Count(out, "\n") + 1; rows != 5 {
		t.Errorf("multi-line placeholder view has %d rows, want 5:\n%q", rows, out)
	}
	// a buffer longer than the window still renders exactly height rows
	e.SetValue(strings.Repeat("x\n", 20))
	out = e.View()
	if rows := strings.Count(out, "\n") + 1; rows != 5 {
		t.Errorf("long buffer view has %d rows, want 5", rows)
	}
}

// The identity highlighter is the default for plain editing and must
// never alter the buffer.
func TestEditorNilHighlighterIdentity(t *testing.T) {
	e := New(60, 5, "", nil)
	e.SetValue("plain\nsecond")
	colored := e.hl.Lines(e.Value())
	if len(colored) != 2 || colored[0] != "plain" || colored[1] != "second" {
		t.Errorf("identity lines = %q", colored)
	}
	pieces := e.hl.Split("plain\n", "second", 3, 5)
	if len(pieces) != 3 || pieces[0] != "sec" || pieces[1] != "on" || pieces[2] != "d" {
		t.Errorf("identity split = %q", pieces)
	}
	// cuts are clamped and deduplicated
	pieces = e.hl.Split("plain\n", "second", -1, 99, 3, 3)
	if len(pieces) != 4 || pieces[0] != "" || pieces[1] != "sec" || pieces[2] != "ond" || pieces[3] != "" {
		t.Errorf("identity split clamp/dedupe = %q", pieces)
	}
}
