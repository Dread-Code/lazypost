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

// Selection must render as reverse video over exactly the selected runes
// (piece boundaries carry the selection into the painted output).
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
