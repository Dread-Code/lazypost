package ui

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"lazypost/internal/render"

	"lazypost/internal/ui/themes"
)

// cursorBlock marks the edit position; styled after the textarea cursor.
var cursorBlock = lipgloss.NewStyle().Reverse(true)

// highlightLine paints one line of source. Each language wires its own
// (Lua: render.HighlightLua, JSON: render.HighlightJSONFragment) so a
// single editor core serves every code-ish field.
type highlightLine func(line string) string

// codeEditor is a minimal single-buffer editor with per-line syntax
// highlighting, replacing bubbles textarea for the script hooks and the
// request body ([[Design - script editor highlighting]] · [[ADR-0015
// Request body editor gets JSON highlighting]]). The stock textarea
// applies one style to the whole value, so per-token colors need a
// custom widget.
//
// First-cut scope: typing, backspace/delete, enter, arrows, home/end,
// cursor-follow vertical scroll. No selection, no undo, no horizontal
// scroll; long strings spanning lines lose color on continuation lines.
type codeEditor struct {
	value       string
	cursor      int // rune offset into value
	top         int // first visible line
	width       int
	height      int
	focused     bool
	placeholder string
	highlight   highlightLine
}

func newCodeEditor(width, height int, placeholder string, highlight highlightLine) *codeEditor {
	if highlight == nil {
		highlight = func(line string) string { return line }
	}
	return &codeEditor{width: width, height: height, placeholder: placeholder, highlight: highlight}
}

func (s *codeEditor) Value() string { return s.value }

func (s *codeEditor) SetValue(v string) {
	s.value = v
	if s.cursor > utf8.RuneCountInString(v) {
		s.cursor = utf8.RuneCountInString(v)
	}
	s.scrollToCursor()
}

func (s *codeEditor) Focus() tea.Cmd { s.focused = true; return nil }
func (s *codeEditor) Blur()          { s.focused = false }

func (s *codeEditor) SetWidth(w int) { s.width = w }
func (s *codeEditor) SetHeight(h int) {
	s.height = h
	s.scrollToCursor()
}

// Update handles editing keys; everything else (section navigation etc.)
// is intercepted by the parent Editor before it reaches the widget.
func (s *codeEditor) Update(msg tea.Msg) (*codeEditor, tea.Cmd) {
	km, ok := msg.(tea.KeyMsg)
	if !ok {
		return s, nil
	}
	r := []rune(s.value)
	switch km.Type {
	case tea.KeyRunes, tea.KeySpace:
		// bubbletea reports a lone space as KeySpace (Runes still carry
		// the ' '); both insert
		s.insertAt(r, string(km.Runes))
	case tea.KeyEnter:
		s.insertAt(r, "\n")
	case tea.KeyBackspace:
		if s.cursor > 0 {
			s.value = string(concat(r[:s.cursor-1], r[s.cursor:]))
			s.cursor--
		}
	case tea.KeyDelete:
		if s.cursor < len(r) {
			s.value = string(concat(r[:s.cursor], r[s.cursor+1:]))
		}
	case tea.KeyLeft:
		if s.cursor > 0 {
			s.cursor--
		}
	case tea.KeyRight:
		if s.cursor < len(r) {
			s.cursor++
		}
	case tea.KeyHome:
		start, _ := s.lineBounds(r, s.cursor)
		s.cursor = start
	case tea.KeyEnd:
		_, end := s.lineBounds(r, s.cursor)
		s.cursor = end
	case tea.KeyUp:
		s.moveLine(r, -1)
	case tea.KeyDown:
		s.moveLine(r, +1)
	}
	s.scrollToCursor()
	return s, nil
}

func (s *codeEditor) insertAt(text []rune, ins string) {
	in := []rune(ins)
	s.value = string(concat(text[:s.cursor], in, text[s.cursor:]))
	s.cursor += len(in)
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
func (s *codeEditor) lineBounds(r []rune, pos int) (int, int) {
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

// moveLine moves the cursor one line up/down, clamping the column to the
// target line's length.
func (s *codeEditor) moveLine(r []rune, d int) {
	start, end := s.lineBounds(r, s.cursor)
	col := s.cursor - start
	if d < 0 {
		if start == 0 {
			return
		}
		targetStart := start - 1
		for targetStart > 0 && r[targetStart-1] != '\n' {
			targetStart--
		}
		s.cursor = minRune(targetStart+col, start-1)
		return
	}
	if end >= len(r) {
		return
	}
	targetEnd := end + 1
	for targetEnd < len(r) && r[targetEnd] != '\n' {
		targetEnd++
	}
	s.cursor = minRune(end+1+col, targetEnd)
}

func minRune(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (s *codeEditor) lineOf(r []rune, pos int) int {
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

func (s *codeEditor) scrollToCursor() {
	if s.height < 1 {
		return
	}
	line := s.lineOf([]rune(s.value), s.cursor)
	if line < s.top {
		s.top = line
	} else if line >= s.top+s.height {
		s.top = line - s.height + 1
	}
}

// luaPaintColors maps Lua token kinds onto the active theme; identifiers
// stay at the terminal default. Styles are built from the package color
// vars, so a runtime theme switch applies on the next render.
func luaPaintColors(kind render.LuaKind, lit string) string {
	var color lipgloss.AdaptiveColor
	switch kind {
	case render.LuaKeyword:
		color = themes.ColorPrimary
	case render.LuaString:
		color = themes.ColorSuccess
	case render.LuaComment:
		color = themes.ColorMuted
	case render.LuaNumber:
		color = themes.ColorInfo
	case render.LuaOperator:
		color = themes.ColorDim
	default: // LuaIdentifier
		return lit
	}
	return lipgloss.NewStyle().Foreground(color).Render(lit)
}

// highlightLuaLine paints a single Lua source line (the Scripts tab).
func highlightLuaLine(line string) string {
	return render.HighlightLua(line, luaPaintColors)
}

// highlightJSONLine paints a single request-body line. Fragment mode
// (no validity gate) keeps colors while the body is still being edited.
func highlightJSONLine(line string) string {
	return render.HighlightJSONFragment(line, highlightJSONColors)
}

func (s *codeEditor) View() string {
	if s.value == "" {
		return s.placeholderView()
	}
	lines := strings.Split(s.value, "\n")
	total := len(lines)
	gutterW := len(strconv.Itoa(total))
	visibleW := s.width - gutterW - 2
	if visibleW < 4 {
		visibleW = 4
	}
	cursorLine, cursorCol := -1, 0
	if s.focused {
		r := []rune(s.value)
		cursorLine = s.lineOf(r, s.cursor)
		start, _ := s.lineBounds(r, s.cursor)
		cursorCol = s.cursor - start
	}
	end := minRune(s.top+s.height, total)
	var b strings.Builder
	for i := s.top; i < end; i++ {
		var rendered string
		if i == cursorLine {
			rendered = s.renderCursorLine(lines[i], cursorCol)
		} else {
			rendered = s.highlight(lines[i])
		}
		b.WriteString(themes.HintStyle.Render(fmt.Sprintf("%*d ", gutterW, i+1)))
		b.WriteString(truncateRunesAnsi(rendered, visibleW))
		if i < end-1 {
			b.WriteString("\n")
		}
	}
	return b.String()
}

// renderCursorLine paints the line with the cursor block inserted at the
// given rune column. The line is split at the cursor byte offset and each
// half is highlighted separately; a split inside a token keeps the same
// color on both sides, so the seam is invisible.
func (s *codeEditor) renderCursorLine(line string, col int) string {
	rb := []rune(line)
	if col > len(rb) {
		col = len(rb)
	}
	byteOff := len(string(rb[:col]))
	pre := s.highlight(line[:byteOff])
	rest := line[byteOff:]
	ch := " "
	remainder := ""
	if rest != "" {
		first, size := utf8.DecodeRuneInString(rest)
		ch = string(first)
		remainder = rest[size:]
	}
	return pre + cursorBlock.Render(ch) + s.highlight(remainder)
}

func (s *codeEditor) placeholderView() string {
	p := themes.HintStyle.Render(s.placeholder)
	if s.focused {
		p = cursorBlock.Render(" ") + p
	}
	return p
}
