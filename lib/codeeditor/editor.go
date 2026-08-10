// Package codeeditor provides a minimal single-buffer terminal editor
// with per-line syntax highlighting, built on Bubble Tea.
//
// The editor is lexer- and theme-agnostic: consumers supply a
// Highlighter that paints whole buffers one line at a time (and can
// re-paint a single line around the cursor with the preceding buffer as
// lexer context), and optionally a StyleProvider for the gutter,
// placeholder, and cursor block. Anything beyond plain text editing —
// selection, undo, horizontal scroll — is intentionally out of scope.
package codeeditor

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Style holds the styles the editor applies while rendering. A
// StyleProvider returning the zero Style uses the package defaults:
// plain gutter and placeholder, reverse-video cursor block.
type Style struct {
	Gutter      lipgloss.Style
	Placeholder lipgloss.Style
	CursorBlock lipgloss.Style
}

func (s Style) resolved() Style {
	if s.CursorBlock.Render(" ") == " " {
		// no explicit cursor style: fall back to the reverse-video block
		s.CursorBlock = lipgloss.NewStyle().Reverse(true)
	}
	return s
}

// Editor is a minimal single-buffer editor with per-line syntax
// highlighting, replacing bubbles textarea where per-token colors are
// needed (the stock textarea applies a single style to the whole
// value).
//
// First-cut scope: typing, backspace/delete, enter, arrows, home/end,
// cursor-follow vertical scroll. No selection, no undo, no horizontal
// scroll.
type Editor struct {
	value       string
	cursor      int // rune offset into value
	top         int // first visible line
	width       int
	height      int
	focused     bool
	placeholder string
	hl          Highlighter
	styles      func() Style
}

// New builds an editor. hl may be nil (plain editing). The style
// provider defaults to the package defaults.
func New(width, height int, placeholder string, hl Highlighter) *Editor {
	if hl == nil {
		hl = IdentityHighlighter()
	}
	return &Editor{width: width, height: height, placeholder: placeholder, hl: hl}
}

// SetStyleProvider sets the function the editor calls on every render
// to obtain its styles, so consumers can re-theme live (a theme switch
// takes effect on the next frame). nil restores the defaults.
func (e *Editor) SetStyleProvider(fn func() Style) {
	e.styles = fn
}

func (e *Editor) style() Style {
	if e.styles != nil {
		return e.styles().resolved()
	}
	return Style{}.resolved()
}

func (e *Editor) Value() string { return e.value }

func (e *Editor) SetValue(v string) {
	e.value = v
	if e.cursor > utf8.RuneCountInString(v) {
		e.cursor = utf8.RuneCountInString(v)
	}
	e.scrollToCursor()
}

// Cursor returns the cursor position as a rune offset into the value.
func (e *Editor) Cursor() int { return e.cursor }

// SetCursor moves the cursor to a rune offset into the value, clamped.
func (e *Editor) SetCursor(pos int) {
	if pos < 0 {
		pos = 0
	}
	if pos > utf8.RuneCountInString(e.value) {
		pos = utf8.RuneCountInString(e.value)
	}
	e.cursor = pos
	e.scrollToCursor()
}

// Top returns the first visible line.
func (e *Editor) Top() int { return e.top }

func (e *Editor) Focus() tea.Cmd { e.focused = true; return nil }
func (e *Editor) Blur()          { e.focused = false }

// Resize sets the editor's cell dimensions.
func (e *Editor) Resize(width, height int) {
	e.width, e.height = width, height
	e.scrollToCursor()
}

// Update handles editing keys; anything else (section navigation etc.)
// is the parent widget's business and never reaches the editor.
func (e *Editor) Update(msg tea.Msg) (*Editor, tea.Cmd) {
	km, ok := msg.(tea.KeyMsg)
	if !ok {
		return e, nil
	}
	r := []rune(e.value)
	switch km.Type {
	case tea.KeyRunes, tea.KeySpace:
		// bubbletea reports a lone space as KeySpace (Runes still carry
		// the ' '); both insert
		e.insertAt(r, string(km.Runes))
	case tea.KeyEnter:
		e.insertAt(r, "\n")
	case tea.KeyBackspace:
		if e.cursor > 0 {
			e.value = string(concat(r[:e.cursor-1], r[e.cursor:]))
			e.cursor--
		}
	case tea.KeyDelete:
		if e.cursor < len(r) {
			e.value = string(concat(r[:e.cursor], r[e.cursor+1:]))
		}
	case tea.KeyLeft:
		if e.cursor > 0 {
			e.cursor--
		}
	case tea.KeyRight:
		if e.cursor < len(r) {
			e.cursor++
		}
	case tea.KeyHome:
		start, _ := e.lineBounds(r, e.cursor)
		e.cursor = start
	case tea.KeyEnd:
		_, end := e.lineBounds(r, e.cursor)
		e.cursor = end
	case tea.KeyUp:
		e.moveLine(r, -1)
	case tea.KeyDown:
		e.moveLine(r, +1)
	}
	e.scrollToCursor()
	return e, nil
}

func (e *Editor) insertAt(text []rune, ins string) {
	in := []rune(ins)
	e.value = string(concat(text[:e.cursor], in, text[e.cursor:]))
	e.cursor += len(in)
}

func concat(parts ...[]rune) []rune {
	var total int
	for _, p := range parts {
		total += len(p)
	}
	out := make([]rune, 0, total)
	for _, p := range parts {
		out = append(out, p...)
	}
	return out
}

// lineBounds returns the rune offsets of the line containing pos.
func (e *Editor) lineBounds(r []rune, pos int) (int, int) {
	if pos > len(r) {
		pos = len(r)
	}
	start := pos
	for start > 0 && r[start-1] != '\n' {
		start--
	}
	end := pos
	for end < len(r) && r[end] != '\n' {
		end++
	}
	return start, end
}

// moveLine moves the cursor one line up/down, clamping the column to
// the target line's length.
func (e *Editor) moveLine(r []rune, d int) {
	start, end := e.lineBounds(r, e.cursor)
	col := e.cursor - start
	if d < 0 {
		if start == 0 {
			return
		}
		targetStart := start - 1
		for targetStart > 0 && r[targetStart-1] != '\n' {
			targetStart--
		}
		e.cursor = minRune(targetStart+col, start-1)
		return
	}
	if end >= len(r) {
		return
	}
	targetEnd := end + 1
	for targetEnd < len(r) && r[targetEnd] != '\n' {
		targetEnd++
	}
	e.cursor = minRune(end+1+col, targetEnd)
}

func minRune(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (e *Editor) lineOf(r []rune, pos int) int {
	if pos > len(r) {
		pos = len(r)
	}
	n := 0
	for i := 0; i < pos; i++ {
		if r[i] == '\n' {
			n++
		}
	}
	return n
}

func (e *Editor) scrollToCursor() {
	if e.height < 1 {
		return
	}
	line := e.lineOf([]rune(e.value), e.cursor)
	if line < e.top {
		e.top = line
	} else if line >= e.top+e.height {
		e.top = line - e.height + 1
	}
}

// View renders the visible window: a line-number gutter, then each
// source line painted by the Highlighter (the cursor line re-painted
// around the cursor block), all truncated to the editor width.
func (e *Editor) View() string {
	if e.value == "" {
		return e.placeholderView()
	}
	st := e.style()
	lines := strings.Split(e.value, "\n")
	colored := e.hl.Lines(e.value)
	total := len(lines)
	gutterW := len(strconv.Itoa(total))
	visibleW := e.width - gutterW - 2
	if visibleW < 4 {
		visibleW = 4
	}
	cursorLine, cursorCol := -1, 0
	if e.focused {
		r := []rune(e.value)
		cursorLine = e.lineOf(r, e.cursor)
		start, _ := e.lineBounds(r, e.cursor)
		cursorCol = e.cursor - start
	}
	end := minRune(e.top+e.height, total)
	var b strings.Builder
	for i := e.top; i < end; i++ {
		var rendered string
		if i == cursorLine {
			rendered = e.renderCursorLine(lines, i, cursorCol, st)
		} else if i < len(colored) {
			rendered = colored[i]
		} else {
			rendered = lines[i]
		}
		b.WriteString(st.Gutter.Render(fmt.Sprintf("%*d ", gutterW, i+1)))
		b.WriteString(TruncateRunesAnsi(rendered, visibleW))
		if i < end-1 {
			b.WriteString("\n")
		}
	}
	return b.String()
}

// renderCursorLine paints the cursor's line with the cursor block
// inserted at the given rune column. The preceding lines are fed back
// as lexer context, so a line continuing a token started earlier (a
// string, a block comment) still colors correctly. Each half is cut
// from the same token stream, so a seam inside a token keeps the same
// color on both sides and is invisible.
func (e *Editor) renderCursorLine(lines []string, idx, col int, st Style) string {
	line := lines[idx]
	rb := []rune(line)
	if col > len(rb) {
		col = len(rb)
	}
	byteOff := len(string(rb[:col]))
	prefix := strings.Join(lines[:idx], "\n")
	pre, _ := e.hl.Split(prefix, line, byteOff)
	charSize := 0
	if byteOff < len(line) {
		_, charSize = utf8.DecodeRuneInString(line[byteOff:])
	}
	_, post := e.hl.Split(prefix, line, byteOff+charSize)
	ch := " "
	if charSize > 0 {
		ch = line[byteOff : byteOff+charSize]
	}
	return pre + st.CursorBlock.Render(ch) + post
}

func (e *Editor) placeholderView() string {
	st := e.style()
	p := st.Placeholder.Render(e.placeholder)
	if e.focused {
		p = st.CursorBlock.Render(" ") + p
	}
	return p
}
