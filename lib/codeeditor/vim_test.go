package codeeditor

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

func key(s string) tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)} }
func esc() tea.KeyMsg         { return tea.KeyMsg{Type: tea.KeyEsc} }

// load sets a value, focuses, and switches to normal mode with the
// cursor at the given rune offset.
func load(e *Editor, v string, cursor int) {
	e.SetValue(v)
	e.SetCursor(cursor)
	e.SetMode(ModeNormal)
}

func TestVimEscEntersAndLeavesNormal(t *testing.T) {
	e := New(60, 10, "", markerHighlighter{})
	e.Focus()
	if e.Mode() != ModeNormal {
		t.Fatalf("mode after focus = %v, want normal (entering the field is NORMAL)", e.Mode())
	}
	e.SetMode(ModeInsert)
	typed(e, runeMsg("ab"))
	if e.Mode() != ModeInsert {
		t.Fatalf("mode = %v, want insert", e.Mode())
	}
	e.Update(esc())
	if e.Mode() != ModeNormal {
		t.Fatalf("mode after esc = %v, want normal", e.Mode())
	}
	// insert mode entries
	for _, k := range []string{"i", "a", "A", "I", "o", "O"} {
		load(e, "ab", 1)
		e.Update(key(k))
		if e.Mode() != ModeInsert {
			t.Errorf("key %q should enter insert mode, got %v", k, e.Mode())
		}
	}
	// a/A/I/o/O land the cursor where vim would
	load(e, "ab", 1)
	e.Update(key("a"))
	if e.cursor != 2 {
		t.Errorf("a: cursor = %d, want 2", e.cursor)
	}
	load(e, "ab", 1)
	e.Update(key("A"))
	if e.cursor != 2 {
		t.Errorf("A: cursor = %d, want 2", e.cursor)
	}
	load(e, "  ab", 3)
	e.Update(key("I"))
	if e.cursor != 2 {
		t.Errorf("I: cursor = %d, want 2 (first non-blank)", e.cursor)
	}
	load(e, "ab", 1)
	e.Update(key("o"))
	if e.Value() != "ab\n" || e.cursor != 3 {
		t.Errorf("o: value = %q cursor = %d", e.Value(), e.cursor)
	}
	load(e, "ab", 1)
	e.Update(key("O"))
	if e.Value() != "\nab" || e.cursor != 0 {
		t.Errorf("O: value = %q cursor = %d", e.Value(), e.cursor)
	}
}

func TestVimMotions(t *testing.T) {
	text := "foo bar baz"
	// h/l
	for _, c := range []struct {
		keys string
		want int
	}{
		{"h", 0}, {"l", 2}, {"lll", 4},
	} {
		e := New(60, 10, "", markerHighlighter{})
		load(e, text, 1)
		e.Update(key(c.keys))
		if e.cursor != c.want {
			t.Errorf("%q: cursor = %d, want %d", c.keys, e.cursor, c.want)
		}
	}
	// w/b/e
	for _, c := range []struct {
		keys string
		from int
		want int
	}{
		{"w", 0, 4}, {"w", 4, 8}, {"b", 8, 4}, {"b", 4, 0}, {"e", 0, 3}, {"e", 5, 7},
	} {
		e := New(60, 10, "", markerHighlighter{})
		load(e, text, c.from)
		e.Update(key(c.keys))
		if e.cursor != c.want {
			t.Errorf("%q from %d: cursor = %d, want %d", c.keys, c.from, e.cursor, c.want)
		}
	}
	// $ ^ 0 gg G
	for _, c := range []struct {
		keys string
		from int
		want int
	}{
		{"$", 1, 11}, {"^", 5, 0}, {"0", 5, 0},
		{"G", 0, 23}, {"gg", 23, 0},
	} {
		e := New(60, 10, "", markerHighlighter{})
		load(e, text+"\nsecond line", c.from)
		e.Update(key(c.keys))
		if e.cursor != c.want {
			t.Errorf("%q from %d: cursor = %d, want %d", c.keys, c.from, e.cursor, c.want)
		}
	}
	// j/k
	e := New(60, 10, "", markerHighlighter{})
	load(e, "ab\ncd\ne", 2) // end of line 1
	e.Update(key("j"))
	if e.cursor != 5 {
		t.Errorf("j: cursor = %d, want 5", e.cursor)
	}
	e.Update(key("k"))
	if e.cursor != 2 {
		t.Errorf("k: cursor = %d, want 2", e.cursor)
	}
	// % bracket match
	e = New(60, 10, "", markerHighlighter{})
	load(e, `{"a": [1, 2]}`, 0)
	e.Update(key("%"))
	if e.cursor != 12 {
		t.Errorf("%%: cursor = %d, want 12", e.cursor)
	}
}

func TestVimOperators(t *testing.T) {
	// x deletes a char, count applied
	e := New(60, 10, "", markerHighlighter{})
	load(e, "abcde", 1)
	e.Update(key("3x"))
	if e.Value() != "ae" {
		t.Errorf("3x: value = %q", e.Value())
	}
	// dd deletes a line; count applied
	e = New(60, 10, "", markerHighlighter{})
	load(e, "a\nb\nc\nd", 2)
	e.Update(key("2dd"))
	if e.Value() != "a\nd" {
		t.Errorf("2dd: value = %q", e.Value())
	}
	// dw, d$ , d0
	for _, c := range []struct {
		keys string
		from int
		want string
	}{
		{"dw", 4, "foo baz"}, // cursor on "bar" start: word + space
		{"d$", 4, "foo "},    // to end of line
		{"d0", 4, "bar baz"}, // line start to cursor
	} {
		e := New(60, 10, "", markerHighlighter{})
		load(e, "foo bar baz", c.from)
		e.Update(key(c.keys))
		if e.Value() != c.want {
			t.Errorf("%q: value = %q, want %q", c.keys, e.Value(), c.want)
		}
	}
	// d2w counts through the operator
	e = New(60, 10, "", markerHighlighter{})
	load(e, "one two three four", 0)
	e.Update(key("d2w"))
	if e.Value() != "three four" {
		t.Errorf("d2w: value = %q", e.Value())
	}
	// deletions fill the register: dd then p pastes the line back
	e = New(60, 10, "", markerHighlighter{})
	load(e, "a\nb", 2)
	e.Update(key("dd"))
	e.Update(key("p"))
	if e.Value() != "a\nb" {
		t.Errorf("dd+p: value = %q", e.Value())
	}
	// yy + P pastes before the cursor line
	e = New(60, 10, "", markerHighlighter{})
	load(e, "a\nb\nc", 2)
	e.Update(key("yy"))
	e.Update(key("P"))
	if e.Value() != "a\nb\nb\nc" {
		t.Errorf("yy+P: value = %q", e.Value())
	}
	// yw + p
	e = New(60, 10, "", markerHighlighter{})
	load(e, "one two", 0)
	e.Update(key("yw"))
	e.Update(key("p"))
	if e.Value() != "oonene two" {
		t.Errorf("yw+p: value = %q", e.Value())
	}
}

func TestVimPasteBeforeSemantics(t *testing.T) {
	e := New(60, 10, "", markerHighlighter{})
	load(e, "abcd", 2)
	e.reg = "X"
	e.Update(key("P"))
	if e.Value() != "abXcd" || e.Cursor() != 2 {
		t.Errorf("characterwise P = %q cursor=%d, want abXcd at 2", e.Value(), e.Cursor())
	}

	e = New(60, 10, "", markerHighlighter{})
	load(e, "a\nb\nc", 2)
	e.reg = "X\n"
	e.Update(key("P"))
	if e.Value() != "a\nX\nb\nc" {
		t.Errorf("linewise P = %q, want line before cursor", e.Value())
	}

	e = New(60, 10, "", markerHighlighter{})
	load(e, "a\nb\nc", 2)
	e.reg = "X\n"
	e.Update(key("p"))
	if e.Value() != "a\nb\nX\nc" {
		t.Errorf("linewise p = %q, want line after cursor", e.Value())
	}
}

func TestVimReverseLineSelection(t *testing.T) {
	e := New(60, 10, "", markerHighlighter{})
	load(e, "a\nb\nc", 4)
	e.Update(key("V"))
	e.Update(key("kk"))
	e.Update(key("y"))
	if e.reg != "a\nb\nc" {
		t.Errorf("reverse linewise yank = %q, want all lines", e.reg)
	}
}

func TestVimReverseBracketOperatorIncludesClosingBracket(t *testing.T) {
	e := New(60, 10, "", markerHighlighter{})
	load(e, "(abc)", 4)
	e.Update(key("d%"))
	if e.Value() != "" {
		t.Errorf("reverse d%% = %q, want empty buffer", e.Value())
	}
}

func TestVimWhitespaceMotions(t *testing.T) {
	e := New(60, 10, "", markerHighlighter{})
	load(e, "one\t two\nthree", 0)
	e.Update(key("w"))
	if e.Cursor() != 5 {
		t.Errorf("w across tab/newline = %d, want 5", e.Cursor())
	}
	e.Update(key("b"))
	if e.Cursor() != 0 {
		t.Errorf("b across tab/newline = %d, want 0", e.Cursor())
	}

	e = New(60, 10, "", markerHighlighter{})
	load(e, " \tfoo", 4)
	e.Update(key("I"))
	if e.Cursor() != 2 {
		t.Errorf("I with tab indentation = %d, want 2", e.Cursor())
	}
}

func TestVimTransientStateClearsAcrossFocus(t *testing.T) {
	e := New(60, 10, "", markerHighlighter{})
	load(e, "one two", 0)
	e.Update(key("d"))
	e.Blur()
	e.Focus()
	e.Update(key("w"))
	if e.Value() != "one two" {
		t.Errorf("pending operator survived focus change: %q", e.Value())
	}
}

func TestVimYankHook(t *testing.T) {
	var yanked []string
	e := New(60, 10, "", markerHighlighter{})
	e.SetYank(func(s string) { yanked = append(yanked, s) })
	load(e, "one two", 0)
	e.Update(key("yw"))
	e.Update(key("yy"))
	if len(yanked) != 2 || yanked[0] != "one" || yanked[1] != "one two" {
		t.Errorf("yanked = %q", yanked)
	}
	// deletions fill the register but do not fire the hook
	load(e, "a\nb", 0)
	e.Update(key("dd"))
	if len(yanked) != 2 {
		t.Errorf("dd fired the yank hook: %q", yanked)
	}
}

func TestVimVisual(t *testing.T) {
	// charwise: v + l + y yanks exactly the selection
	var yanked []string
	e := New(60, 10, "", markerHighlighter{})
	e.SetYank(func(s string) { yanked = append(yanked, s) })
	load(e, "one two", 0)
	e.Update(key("v"))
	if e.Mode() != ModeVisualChar {
		t.Fatalf("mode = %v, want visual", e.Mode())
	}
	e.Update(key("lll"))
	e.Update(key("y"))
	if e.Mode() != ModeNormal {
		t.Errorf("mode after y = %v, want normal", e.Mode())
	}
	if len(yanked) != 1 || yanked[0] != "one" {
		t.Errorf("visual yank = %q, want [one]", yanked)
	}
	// visual d deletes the selection
	e = New(60, 10, "", markerHighlighter{})
	load(e, "one two", 0)
	e.Update(key("v"))
	e.Update(key("lll"))
	e.Update(key("d"))
	if e.Value() != " two" {
		t.Errorf("visual d: value = %q", e.Value())
	}
	// linewise V yanks whole lines
	yanked = nil
	e = New(60, 10, "", markerHighlighter{})
	e.SetYank(func(s string) { yanked = append(yanked, s) })
	load(e, "a\nb\nc", 2)
	e.Update(key("V"))
	if e.Mode() != ModeVisualLine {
		t.Fatalf("mode = %v, want visual line", e.Mode())
	}
	e.Update(key("j"))
	e.Update(key("y"))
	if len(yanked) != 1 || yanked[0] != "b\nc" {
		t.Errorf("linewise yank = %q, want [b\\nc]", yanked)
	}
	// esc leaves visual mode
	e = New(60, 10, "", markerHighlighter{})
	load(e, "a\nb\nc", 2)
	e.Update(key("v"))
	e.Update(esc())
	if e.Mode() != ModeNormal {
		t.Errorf("mode after esc = %v, want normal", e.Mode())
	}
}

// tokenHighlighter paints each word with its own color, mimicking a
// syntax highlighter whose tokens each carry their own SGR + reset —
// the case that used to break the selection after the first token.
type tokenHighlighter struct{}

var tokenColors = []string{"\x1b[31m", "\x1b[32m", "\x1b[34m"}

func (tokenHighlighter) paint(line string) string {
	var b strings.Builder
	i := 0
	for _, w := range strings.Fields(line) {
		b.WriteString(tokenColors[i%len(tokenColors)])
		b.WriteString(w)
		b.WriteString("\x1b[0m ")
		i++
	}
	return strings.TrimSuffix(b.String(), " ")
}

func (tokenHighlighter) Lines(src string) []string {
	out := []string{}
	for _, l := range strings.Split(src, "\n") {
		out = append(out, tokenHighlighter{}.paint(l))
	}
	return out
}

func (tokenHighlighter) Split(prefix, line string, cuts ...int) []string {
	// paint each piece independently so cut points are plain-line byte
	// offsets (slicing the painted string directly would cut mid-ANSI)
	pieces := []string{}
	prev := 0
	for _, c := range append(append([]int{}, cuts...), len(line)) {
		pieces = append(pieces, tokenHighlighter{}.paint(line[prev:c]))
		prev = c
	}
	return pieces
}

// Selection rendering must be uniform: token colors are stripped and
// the whole selection renders in one reverse-video style, without a
// break where the first token's reset lands.
func TestVimVisualSelectionUniform(t *testing.T) {
	prev := lipgloss.DefaultRenderer().ColorProfile()
	lipgloss.DefaultRenderer().SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.DefaultRenderer().SetColorProfile(prev) })

	e := New(60, 10, "", tokenHighlighter{})
	e.SetValue("aa bb cc")
	e.SetCursor(0)
	e.Focus()
	e.SetMode(ModeVisualChar)
	e.Update(key("llll")) // select "aa b" (4 runes)
	out := e.View()
	// the selected span is one reversed block over plain text
	if !strings.Contains(out, "\x1b[7maa b\x1b[0m") {
		t.Errorf("selection not uniform over %q:\n%q", "aa b", out)
	}
	// no token color leaks inside the selection (the first token's
	// reset used to break the reverse video there)
	for _, tok := range []string{"\x1b[31maa", "\x1b[32mbb"} {
		if strings.Contains(out, tok) {
			t.Errorf("token color %q leaked into the selection:\n%q", tok, out)
		}
	}
	// the unselected remainder keeps its colors (its piece paints "cc"
	// as the first word of that piece, hence red)
	if !strings.Contains(out, "\x1b[31mcc\x1b[0m") {
		t.Errorf("unselected text lost its colors:\n%q", out)
	}
}

// Selection must render as reverse video over exactly the selected
// runes (piece boundaries carry the selection into the painted output).
func TestVimVisualSelectionRendering(t *testing.T) {
	prev := lipgloss.DefaultRenderer().ColorProfile()
	lipgloss.DefaultRenderer().SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.DefaultRenderer().SetColorProfile(prev) })

	e := New(60, 10, "", markerHighlighter{})
	e.SetValue("one two")
	e.SetCursor(0)
	e.SetMode(ModeVisualChar)
	e.Update(key("lll")) // select "one"
	out := e.View()
	// the selected runes render inside reverse-video SGR
	if !strings.Contains(out, "\x1b[7m<one\x1b[0m") {
		t.Errorf("selection not reversed over %q:\n%q", "one", out)
	}
}

func TestVimUnknownKeysResetPending(t *testing.T) {
	e := New(60, 10, "", markerHighlighter{})
	load(e, "one two", 0)
	e.Update(key("d"))
	e.Update(key("z")) // unknown target: pending must clear
	e.Update(key("d"))
	if e.Value() != "one two" {
		t.Errorf("value changed after d z d: %q", e.Value())
	}
}
