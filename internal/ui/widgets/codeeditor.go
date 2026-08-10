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

// highlighter paints an entire buffer one line at a time and can re-paint
// a single line around the cursor. Both paint whole buffers, so lexer
// state survives across lines: JSON keys stay keys ([[ADR-0015]]), Lua
// block comments and long strings keep their color on continuation lines.
type highlighter struct {
	// lines paints src and returns one colored string per source line.
	lines func(src string) []string
	// split paints line in the context of prefix and returns the colored
	// line split at cut (a byte offset into line): the cursor seam. The
	// prefix restores the token state, so a cut inside a token keeps the
	// same color on both sides.
	split func(prefix, line string, cut int) (string, string)
}

// codeEditor is a minimal single-buffer editor with per-line syntax
// highlighting, replacing bubbles textarea for the script hooks and the
// request body ([[Design - script editor highlighting]] · [[ADR-0015
// Request body editor gets JSON highlighting]]). The stock textarea
// applies one style to the whole value, so per-token colors need a
// custom widget.
//
// First-cut scope: typing, backspace/delete, enter, arrows, home/end,
// cursor-follow vertical scroll. No selection, no undo, no horizontal
// scroll.
type codeEditor struct {
	value       string
	cursor      int // rune offset into value
	top         int // first visible line
	width       int
	height      int
	focused     bool
	placeholder string
	highlight   highlighter
}

func newCodeEditor(width, height int, placeholder string, hl highlighter) *codeEditor {
	if hl.lines == nil {
		hl.lines = func(src string) []string { return strings.Split(src, "\n") }
	}
	if hl.split == nil {
		hl.split = func(prefix, line string, cut int) (string, string) {
			if cut < 0 {
				cut = 0
			}
			if cut > len(line) {
				cut = len(line)
			}
			return line[:cut], line[cut:]
		}
	}
	return &codeEditor{width: width, height: height, placeholder: placeholder, highlight: hl}
}

// jsonHighlighter colors request bodies. The whole buffer is lexed at
// once so the object context survives across lines and keys keep their
// key color; a line-scoped lex reclassifies every key as a string and
// the body renders in one color ([[Gotcha - request body renders
// uncolored while the response highlights]]).
func jsonHighlighter() highlighter {
	return highlighter{
		lines: func(src string) []string { return render.HighlightJSONLines(src, highlightJSONColors) },
		split: func(prefix, line string, cut int) (string, string) {
			return render.HighlightJSONSplit(prefix, line, cut, highlightJSONColors)
		},
	}
}

// luaHighlighter colors the script hooks (the Scripts tab).
func luaHighlighter() highlighter {
	return highlighter{
		lines: func(src string) []string { return render.HighlightLuaLines(src, luaPaintColors) },
		split: func(prefix, line string, cut int) (string, string) {
			return render.HighlightLuaSplit(prefix, line, cut, luaPaintColors)
		},
	}
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

func (s *codeEditor) View() string {
	if s.value == "" {
		return s.placeholderView()
	}
	lines := strings.Split(s.value, "\n")
	colored := s.highlight.lines(s.value)
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
			rendered = s.renderCursorLine(lines, i, cursorCol)
		} else if i < len(colored) {
			rendered = colored[i]
		} else {
			rendered = lines[i]
		}
		b.WriteString(themes.HintStyle.Render(fmt.Sprintf("%*d ", gutterW, i+1)))
		b.WriteString(truncateRunesAnsi(rendered, visibleW))
		if i < end-1 {
			b.WriteString("\n")
		}
	}
	return b.String()
}

// renderCursorLine paints the cursor's line with the cursor block
// inserted at the given rune column. The preceding lines are fed back as
// lexer context, so a line continuing a token started earlier (a string,
// a Lua block comment) still colors correctly. Each half is cut from the
// same token stream, so a seam inside a token keeps the same color on
// both sides and is invisible.
func (s *codeEditor) renderCursorLine(lines []string, idx, col int) string {
	line := lines[idx]
	rb := []rune(line)
	if col > len(rb) {
		col = len(rb)
	}
	byteOff := len(string(rb[:col]))
	prefix := strings.Join(lines[:idx], "\n")
	pre, _ := s.highlight.split(prefix, line, byteOff)
	charSize := 0
	if byteOff < len(line) {
		_, charSize = utf8.DecodeRuneInString(line[byteOff:])
	}
	_, post := s.highlight.split(prefix, line, byteOff+charSize)
	ch := " "
	if charSize > 0 {
		ch = line[byteOff : byteOff+charSize]
	}
	return pre + cursorBlock.Render(ch) + post
}

func (s *codeEditor) placeholderView() string {
	p := themes.HintStyle.Render(s.placeholder)
	if s.focused {
		p = cursorBlock.Render(" ") + p
	}
	return p
}
