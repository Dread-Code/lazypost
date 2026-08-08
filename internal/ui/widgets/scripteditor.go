package ui

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"lazypost/internal/render"
)

// cursorBlock marks the edit position; styled after the textarea cursor.
var cursorBlock = lipgloss.NewStyle().Reverse(true)

// scriptEditor is a minimal single-buffer editor with Lua syntax
// highlighting, replacing bubbles textarea for the pre/post hooks
// ([[Design - script editor highlighting]]). The stock textarea applies
// one style to the whole value, so per-token colors need a custom widget.
//
// First-cut scope: typing, backspace/delete, enter, arrows, home/end,
// cursor-follow vertical scroll. No selection, no undo, no horizontal
// scroll; long strings spanning lines lose color on continuation lines.
type scriptEditor struct {
	value       string
	cursor      int // rune offset into value
	top         int // first visible line
	width       int
	height      int
	focused     bool
	placeholder string
}

func newScriptEditor(width, height int, placeholder string) *scriptEditor {
	return &scriptEditor{width: width, height: height, placeholder: placeholder}
}

func (s *scriptEditor) Value() string { return s.value }

func (s *scriptEditor) SetValue(v string) {
	s.value = v
	if s.cursor > utf8.RuneCountInString(v) {
		s.cursor = utf8.RuneCountInString(v)
	}
	s.scrollToCursor()
}

func (s *scriptEditor) Focus() tea.Cmd { s.focused = true; return nil }
func (s *scriptEditor) Blur()          { s.focused = false }

func (s *scriptEditor) SetWidth(w int) { s.width = w }
func (s *scriptEditor) SetHeight(h int) {
	s.height = h
	s.scrollToCursor()
}

// Update handles editing keys; everything else (section navigation etc.)
// is intercepted by the parent Editor before it reaches the widget.
func (s *scriptEditor) Update(msg tea.Msg) (*scriptEditor, tea.Cmd) {
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

func (s *scriptEditor) insertAt(text []rune, ins string) {
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
func (s *scriptEditor) lineBounds(r []rune, pos int) (int, int) {
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
func (s *scriptEditor) moveLine(r []rune, d int) {
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

func (s *scriptEditor) lineOf(r []rune, pos int) int {
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

func (s *scriptEditor) scrollToCursor() {
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
		color = ColorPrimary
	case render.LuaString:
		color = ColorSuccess
	case render.LuaComment:
		color = ColorMuted
	case render.LuaNumber:
		color = ColorInfo
	case render.LuaOperator:
		color = ColorDim
	default: // LuaIdentifier
		return lit
	}
	return lipgloss.NewStyle().Foreground(color).Render(lit)
}

func (s *scriptEditor) View() string {
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
			rendered = render.HighlightLua(lines[i], luaPaintColors)
		}
		b.WriteString(HintStyle.Render(fmt.Sprintf("%*d ", gutterW, i+1)))
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
func (s *scriptEditor) renderCursorLine(line string, col int) string {
	rb := []rune(line)
	if col > len(rb) {
		col = len(rb)
	}
	byteOff := len(string(rb[:col]))
	pre := render.HighlightLua(line[:byteOff], luaPaintColors)
	rest := line[byteOff:]
	ch := " "
	remainder := ""
	if rest != "" {
		first, size := utf8.DecodeRuneInString(rest)
		ch = string(first)
		remainder = rest[size:]
	}
	return pre + cursorBlock.Render(ch) + render.HighlightLua(remainder, luaPaintColors)
}

func (s *scriptEditor) placeholderView() string {
	p := HintStyle.Render(s.placeholder)
	if s.focused {
		p = cursorBlock.Render(" ") + p
	}
	return p
}
