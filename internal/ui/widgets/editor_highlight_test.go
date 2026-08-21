package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Dread-Code/lazypost/lib/codeeditor"
	"github.com/charmbracelet/lipgloss"

	"github.com/Dread-Code/lazypost/internal/ui/themes"
	"github.com/muesli/termenv"
)

func forceTrueColor(t *testing.T) {
	t.Helper()
	prev := lipgloss.DefaultRenderer().ColorProfile()
	lipgloss.DefaultRenderer().SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.DefaultRenderer().SetColorProfile(prev) })
}

// The request Body tab is a codeeditor.Editor wired to the JSON
// highlighter; a valid JSON body must render with token colors.
func TestCodeEditorJSONBodyHighlighted(t *testing.T) {
	forceTrueColor(t)

	e := codeeditor.New(60, 10, "", jsonHighlighter())
	e.SetValue(`{"title": "hi", "count": 1, "ok": true}`)
	e.SetCursor(len([]rune(e.Value())))
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

// Regression ([[Gotcha - request body renders uncolored while the
// response highlights]]): keys and value strings must render in
// different colors. A line-scoped lexer reclassifies every key as a
// string and the whole body renders in one color.
func TestCodeEditorJSONBodyKeysAndStringsDistinct(t *testing.T) {
	forceTrueColor(t)

	e := codeeditor.New(60, 10, "", jsonHighlighter())
	e.SetValue("{\n  \"title\": \"hi\",\n  \"count\": 2\n}")
	e.SetCursor(len([]rune(e.Value())))
	e.Focus()
	out := e.View()
	keyColor := fgColorAt(out, `"title"`)
	strColor := fgColorAt(out, `"hi"`)
	if keyColor == "" || strColor == "" {
		t.Fatalf("no fg colors found:\n%s", out)
	}
	if keyColor == strColor {
		t.Errorf("key and string render in the same color %s — body looks one-colored", keyColor)
	}
}

// Regression: visual selection over real chroma-colored JSON must be
// one uniform reverse-video block — no token color or reset may break
// the selection after the first token ([[Gotcha - vim visual selection
// breaks after the first token]]).
func TestCodeEditorJSONVisualSelectionUniform(t *testing.T) {
	forceTrueColor(t)

	e := codeeditor.New(60, 10, "", jsonHighlighter())
	e.SetValue("{\n  \"title\": \"hi\",\n  \"count\": 2\n}")
	e.SetCursor(3) // start of line 2 (the key line)
	e.Focus()
	e.SetMode(codeeditor.ModeVisualChar)
	// extend across the key token and into the string value
	e.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("lllllllllll")})
	out := e.View()
	start := strings.Index(out, "\x1b[7m")
	if start < 0 {
		t.Fatalf("no reversed selection in view:\n%s", out)
	}
	end := strings.Index(out[start:], "\x1b[0m")
	sel := out[start : start+end]
	inner := sel[4:] // strip the \x1b[7m (sel ends at the reset)
	if strings.Contains(inner, "\x1b[") {
		t.Errorf("token colors or resets leaked into the selection span %q", sel)
	}
	// the selected text is intact inside the block
	if !strings.Contains(inner, `"title"`) {
		t.Errorf("selection lost the key text: %q", sel)
	}
}

// Regression: with the cursor on the line after a Lua comment, the code
// must keep its colors — the cursor-line prefix used to omit the
// trailing newline, so chroma's `--.*$` comment token swallowed the
// next line ([[Gotcha - cursor line after a comment renders as a
// comment]]).
func TestLuaCursorLineAfterCommentNotCommented(t *testing.T) {
	forceTrueColor(t)

	e := codeeditor.New(60, 10, "", luaHighlighter())
	e.SetValue("-- a comment\nreq.headers[\"X\"] = \"y\"\nlocal z = 2")
	e.SetCursor(len([]rune("-- a comment\n"))) // line 2, col 0
	e.Focus()
	out := e.View()
	// "eq.headers" is an identifier: it renders uncolored. If the comment
	// swallowed the cursor line, it would be wrapped in the muted style.
	commentStyled := lipgloss.NewStyle().Foreground(themes.NewStyles(themes.DefaultTheme).ColorMuted).Render(`eq.headers`)
	if strings.Contains(out, commentStyled) {
		t.Errorf("cursor line after a comment is comment-colored:\n%s", out)
	}
	if !strings.Contains(out, `eq.headers`) {
		t.Errorf("cursor line lost its code:\n%s", out)
	}
}

// fgColorAt returns the last SGR 38;2;r;g;b sequence before lit in out.
func fgColorAt(out, lit string) string {
	i := strings.Index(out, lit)
	if i < 0 {
		return ""
	}
	for j := i - 1; j >= 0; j-- {
		if out[j] == 'm' {
			start := strings.LastIndex(out[:j+1], "\x1b[")
			if start < 0 {
				return ""
			}
			return out[start : j+1]
		}
	}
	return ""
}

// Regression: the cursor line is re-painted with the preceding buffer as
// lexer context, so a key on a later line keeps its key color while the
// cursor sits on that line (line-scoped lexing would make it a string).
func TestCodeEditorJSONBodyCursorLineKeepsContext(t *testing.T) {
	forceTrueColor(t)

	e := codeeditor.New(60, 10, "", jsonHighlighter())
	e.SetValue("{\n  \"title\": \"hi\",\n  \"count\": 2\n}")
	e.SetCursor(len([]rune(e.Value()))) // end of the last line
	e.Focus()
	out := e.View()
	keyColor := fgColorAt(out, `"count"`)
	strColor := fgColorAt(out, `"hi"`)
	if keyColor == "" || strColor == "" {
		t.Fatalf("no fg colors found:\n%s", out)
	}
	if keyColor == strColor {
		t.Errorf("cursor line lost context: key and string share %s", keyColor)
	}
}

// Regression: while editing, the buffer is usually not valid JSON yet —
// the highlighter must still color it (no identity gate).
func TestCodeEditorJSONBodyFragmentWhileEditing(t *testing.T) {
	forceTrueColor(t)

	e := codeeditor.New(60, 10, "", jsonHighlighter())
	e.SetValue(`{"title": "`) // unterminated string: invalid JSON
	e.SetCursor(len([]rune(e.Value())))
	e.Focus()
	out := e.View()
	if !strings.Contains(out, "\x1b[") {
		t.Errorf("fragment mode should color invalid-but-typed JSON, got:\n%s", out)
	}
	// typing still works on the highlighted editor (insert mode first:
	// focus lands in NORMAL)
	e.SetMode(codeeditor.ModeInsert)
	e.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("hi")})
	if !strings.Contains(e.Value(), `"hi`) {
		t.Errorf("typing into fragment body failed, value = %q", e.Value())
	}
}
