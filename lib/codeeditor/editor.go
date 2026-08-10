// Package codeeditor provides a minimal single-buffer terminal editor
// with per-line syntax highlighting and vim editing modes, built on
// Bubble Tea.
//
// The editor is lexer- and theme-agnostic: consumers supply a
// Highlighter that paints whole buffers one line at a time (and can
// re-paint a single line around arbitrary cut points with the preceding
// buffer as lexer context), and optionally a StyleProvider for the
// gutter, placeholder, and cursor block. Anything beyond plain text
// editing — selection in insert mode, undo, horizontal scroll — is
// intentionally out of scope.
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
// The editor is vim-modal: focusing it lands in NORMAL mode (motions,
// operators), and insert mode is an explicit i/a/A/I/o/O away. First-
// cut scope: typing, backspace/delete, enter, arrows, home/end,
// cursor-follow vertical scroll. No selection in insert mode, no undo,
// no horizontal scroll.
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

	mode     Mode         // current editing mode
	anchor   int          // visual-mode selection anchor (rune offset)
	reg      string       // unnamed register (last yank/delete)
	pending  rune         // operator awaiting its target ("d"|"y")
	gPending bool         // 'g' awaiting 'g' (gg → buffer start)
	count    int          // numeric prefix (d2w, 3yy)
	yank     func(string) // clipboard hook, fired on yanks; nil = register only
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

// Mode returns the current editing mode.
func (e *Editor) Mode() Mode { return e.mode }

// SetMode switches the editing mode. Leaving visual mode clears the
// selection; entering visual modes anchors it at the cursor.
func (e *Editor) SetMode(m Mode) {
	switch m {
	case ModeVisualChar:
		e.anchor = e.cursor
	case ModeVisualLine:
		e.anchor = e.lineStartRune([]rune(e.value), e.cursor)
	default:
		e.anchor = -1
	}
	e.mode = m
}

// SetYank sets the hook fired with every yanked text (visual y, yy, yw,
// y$, y0). nil disables it; the internal register still records yanks
// and deletions, so p/P work without a hook.
func (e *Editor) SetYank(fn func(string)) { e.yank = fn }

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
	if e.anchor > utf8.RuneCountInString(v) {
		e.anchor = utf8.RuneCountInString(v)
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

func (e *Editor) Focus() tea.Cmd {
	e.focused = true
	// entering the field always lands in NORMAL mode: navigation and
	// operators first, editing is an explicit i/a/A/I/o/O away
	e.SetMode(ModeNormal)
	return nil
}
func (e *Editor) Blur() { e.focused = false }

// Resize sets the editor's cell dimensions.
func (e *Editor) Resize(width, height int) {
	e.width, e.height = width, height
	e.scrollToCursor()
}

// Update routes keys by mode; anything else (section navigation etc.)
// is the parent widget's business and never reaches the editor.
func (e *Editor) Update(msg tea.Msg) (*Editor, tea.Cmd) {
	km, ok := msg.(tea.KeyMsg)
	if !ok {
		return e, nil
	}
	switch e.mode {
	case ModeNormal:
		e.updateNormal(km)
	case ModeVisualChar, ModeVisualLine:
		e.updateVisual(km)
	default:
		e.updateInsert(km)
	}
	e.scrollToCursor()
	return e, nil
}

func (e *Editor) updateInsert(km tea.KeyMsg) {
	r := []rune(e.value)
	switch km.Type {
	case tea.KeyEsc:
		e.mode = ModeNormal
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
}

func (e *Editor) updateNormal(km tea.KeyMsg) {
	switch km.Type {
	case tea.KeyEsc:
		// already normal
	case tea.KeyLeft:
		if e.cursor > 0 {
			e.cursor--
		}
	case tea.KeyRight:
		e.cursor = minRune(e.cursor+1, utf8.RuneCountInString(e.value))
	case tea.KeyHome:
		e.cursor = e.lineStartRune([]rune(e.value), e.cursor)
	case tea.KeyEnd:
		e.cursor = e.lineEndRune([]rune(e.value), e.cursor)
	case tea.KeyUp:
		e.moveLine([]rune(e.value), -1)
	case tea.KeyDown:
		e.moveLine([]rune(e.value), +1)
	case tea.KeyRunes, tea.KeySpace:
		// a fast "dd" can arrive as one batch of runes; process in order
		for _, r := range km.Runes {
			e.normalRune(r)
		}
	}
}

func (e *Editor) normalRune(r rune) {
	rs := []rune(e.value)
	// an operator (d/y) awaiting its target: the next key is the target,
	// except digits 1-9 which extend the count (d2w). 0 after an
	// operator is the d0/y0 target, never a count.
	if e.pending != 0 && !(r >= '1' && r <= '9') {
		e.targetOp(r)
		return
	}
	switch r {
	case '0':
		if e.pending == 0 && e.count == 0 {
			e.cursor = e.lineStartRune(rs, e.cursor)
			return
		}
		e.count = e.count * 10
	case '1', '2', '3', '4', '5', '6', '7', '8', '9':
		e.count = e.count*10 + int(r-'0')
	case 'h':
		n := e.n()
		for i := 0; i < n; i++ {
			if e.cursor > 0 {
				e.cursor--
			}
		}
	case 'l':
		n := e.n()
		for i := 0; i < n; i++ {
			if e.cursor < len(rs) {
				e.cursor++
			}
		}
	case 'j':
		n := e.n()
		for i := 0; i < n; i++ {
			e.moveLine(rs, +1)
		}
	case 'k':
		n := e.n()
		for i := 0; i < n; i++ {
			e.moveLine(rs, -1)
		}
	case 'w':
		n := e.n()
		for i := 0; i < n; i++ {
			e.cursor = nextWord(rs, e.cursor)
		}
	case 'b':
		n := e.n()
		for i := 0; i < n; i++ {
			e.cursor = prevWord(rs, e.cursor)
		}
	case 'e':
		n := e.n()
		for i := 0; i < n; i++ {
			e.cursor = wordEnd(rs, e.cursor)
		}
	case '$':
		e.cursor = e.lineEndRune(rs, e.cursor)
	case '^':
		e.cursor = e.firstNonBlank(rs, e.cursor)
	case 'g':
		if e.gPending {
			e.cursor = 0
			e.gPending = false
		} else {
			e.gPending = true
		}
	case 'G':
		e.cursor = len(rs)
	case '%':
		if i := matchBracket(rs, e.cursor); i >= 0 {
			e.cursor = i
		}
	case 'i':
		e.mode = ModeInsert
	case 'a':
		e.cursor = minRune(e.cursor+1, len(rs))
		e.mode = ModeInsert
	case 'A':
		e.cursor = e.lineEndRune(rs, e.cursor)
		e.mode = ModeInsert
	case 'I':
		e.cursor = e.firstNonBlank(rs, e.cursor)
		e.mode = ModeInsert
	case 'o':
		pos := e.lineEndInclNewline(rs, e.cursor)
		e.value = string(concat(rs[:pos], []rune("\n"), rs[pos:]))
		e.cursor = pos + 1
		e.mode = ModeInsert
	case 'O':
		pos := e.lineStartRune(rs, e.cursor)
		e.value = string(concat(rs[:pos], []rune("\n"), rs[pos:]))
		e.cursor = pos
		e.mode = ModeInsert
	case 'x':
		e.deleteChar(e.n())
	case 'd', 'y':
		e.pending = r
	case 'p':
		e.paste(false)
	case 'P':
		e.paste(true)
	case 'v':
		e.SetMode(ModeVisualChar)
	case 'V':
		e.SetMode(ModeVisualLine)
	case 'u', 'q', ':':
		// unbound first cut: undo, macros, ex-commands
	default:
		e.resetPending()
	}
}

// n returns the pending count (1 if none) and resets it.
func (e *Editor) n() int {
	n := e.count
	if n == 0 {
		n = 1
	}
	e.count = 0
	return n
}

// resetPending clears transient normal-mode state.
func (e *Editor) resetPending() {
	e.pending = 0
	e.gPending = false
	e.count = 0
}

// lineOp performs dd (delete line) or yy (yank line) n times.
func (e *Editor) lineOp(op rune, n int) {
	rs := []rune(e.value)
	start := e.lineStartRune(rs, e.cursor)
	end := e.lineEndInclNewline(rs, e.cursor)
	for i := 1; i < n; i++ {
		if end >= len(rs) {
			break
		}
		end = e.lineEndInclNewline(rs, end)
	}
	e.reg = string(rs[start:end])
	if op == 'd' {
		e.value = string(concat(rs[:start], rs[end:]))
		e.cursor = start
	} else if e.yank != nil {
		e.yank(e.reg)
	}
	e.resetPending()
}

// targetOp performs d<target> or y<target> with the current count.
func (e *Editor) targetOp(target rune) {
	op := e.pending
	rs := []rune(e.value)
	var start, end int
	switch target {
	case 'd', 'y':
		// dd / yy (linewise, count lines)
		e.lineOp(target, e.n())
		return
	case 'w':
		start = e.cursor
		end = e.cursor
		n := e.n()
		for i := 0; i < n; i++ {
			if op == 'y' {
				// yw yanks to the end of the current word (vim semantics)
				end = wordEnd(rs, end)
			} else {
				end = nextWord(rs, end)
			}
		}
	case '$':
		start = e.cursor
		end = e.lineEndRune(rs, e.cursor)
	case '0':
		start = e.lineStartRune(rs, e.cursor)
		end = e.cursor
	case '%':
		if i := matchBracket(rs, e.cursor); i >= 0 {
			if i >= e.cursor {
				start, end = e.cursor, i+1
			} else {
				start, end = i, e.cursor
			}
		} else {
			e.resetPending()
			return
		}
	default:
		e.resetPending()
		return
	}
	e.reg = string(rs[start:end])
	if op == 'd' {
		e.value = string(concat(rs[:start], rs[end:]))
		e.cursor = start
	} else if e.yank != nil {
		e.yank(e.reg)
	}
	e.resetPending()
}

// deleteChar deletes n runes at the cursor (x).
func (e *Editor) deleteChar(n int) {
	rs := []rune(e.value)
	if e.cursor >= len(rs) {
		e.resetPending()
		return
	}
	end := minRune(e.cursor+n, len(rs))
	e.reg = string(rs[e.cursor:end])
	e.value = string(concat(rs[:e.cursor], rs[end:]))
	e.resetPending()
}

// paste inserts the register at the cursor (P) or just after it (p);
// register content ending in a newline pastes as a whole line.
func (e *Editor) paste(before bool) {
	if e.reg == "" {
		return
	}
	rs := []rune(e.value)
	ins := []rune(e.reg)
	if len(ins) > 0 && ins[len(ins)-1] == '\n' {
		pos := e.lineEndInclNewline(rs, e.cursor)
		e.value = string(concat(rs[:pos], ins, rs[pos:]))
		e.cursor = pos
	} else {
		pos := minRune(e.cursor+1, len(rs))
		e.value = string(concat(rs[:pos], ins, rs[pos:]))
		e.cursor = pos
	}
}

func (e *Editor) updateVisual(km tea.KeyMsg) {
	switch km.Type {
	case tea.KeyEsc:
		e.mode = ModeNormal
	case tea.KeyLeft:
		e.cursor = maxRune(e.cursor-1, 0)
	case tea.KeyRight:
		e.cursor = minRune(e.cursor+1, utf8.RuneCountInString(e.value))
	case tea.KeyUp:
		e.moveLine([]rune(e.value), -1)
	case tea.KeyDown:
		e.moveLine([]rune(e.value), +1)
	case tea.KeyHome:
		e.cursor = e.lineStartRune([]rune(e.value), e.cursor)
	case tea.KeyEnd:
		e.cursor = e.lineEndRune([]rune(e.value), e.cursor)
	case tea.KeyRunes, tea.KeySpace:
		for _, r := range km.Runes {
			e.visualRune(r)
		}
	}
}

func (e *Editor) visualRune(r rune) {
	rs := []rune(e.value)
	switch r {
	case 'v':
		if e.mode == ModeVisualLine {
			e.mode = ModeVisualChar
		} else {
			e.mode = ModeNormal
		}
	case 'V':
		if e.mode == ModeVisualChar {
			e.mode = ModeVisualLine
		} else {
			e.mode = ModeNormal
		}
	case 'y':
		e.yankSelection(rs)
	case 'd', 'x':
		e.deleteSelection(rs)
	case 'h':
		e.cursor = maxRune(e.cursor-1, 0)
	case 'l':
		e.cursor = minRune(e.cursor+1, len(rs))
	case 'j':
		e.moveLine(rs, +1)
	case 'k':
		e.moveLine(rs, -1)
	case 'w':
		e.cursor = nextWord(rs, e.cursor)
	case 'b':
		e.cursor = prevWord(rs, e.cursor)
	case 'e':
		e.cursor = wordEnd(rs, e.cursor)
	case '$':
		e.cursor = e.lineEndRune(rs, e.cursor)
	case '^':
		e.cursor = e.firstNonBlank(rs, e.cursor)
	case '0':
		e.cursor = e.lineStartRune(rs, e.cursor)
	case 'g':
		if e.gPending {
			e.cursor = 0
			e.gPending = false
		} else {
			e.gPending = true
		}
	case 'G':
		e.cursor = len(rs)
	case '%':
		if i := matchBracket(rs, e.cursor); i >= 0 {
			e.cursor = i
		}
	}
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

func maxRune(a, b int) int {
	if a > b {
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
// source line painted by the Highlighter, with the cursor block and the
// visual selection applied as styled pieces, all truncated to the
// editor width. The window always renders exactly e.height rows —
// buffer lines beyond the window scroll, rows beyond the buffer render
// blank, so consumers never need to pad.
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
	var selStart, selEnd = -1, -1
	if e.mode == ModeVisualChar || e.mode == ModeVisualLine {
		selStart, selEnd = e.selection([]rune(e.value))
	}
	end := e.top + e.height
	var b strings.Builder
	for i := e.top; i < end; i++ {
		var rendered string
		switch {
		case i < total && i < len(colored):
			rendered = renderLine(e, lines, i, st, selStart, selEnd, cursorLine, cursorCol)
			b.WriteString(st.Gutter.Render(fmt.Sprintf("%*d ", gutterW, i+1)))
		case i < total:
			rendered = lines[i]
			b.WriteString(st.Gutter.Render(fmt.Sprintf("%*d ", gutterW, i+1)))
		default:
			// filler row beyond the buffer: blank gutter, no number
			b.WriteString(strings.Repeat(" ", gutterW+1))
		}
		b.WriteString(TruncateRunesAnsi(rendered, visibleW))
		if i < end-1 {
			b.WriteString("\n")
		}
	}
	return b.String()
}

// renderLine paints one source line as styled pieces: cuts split the
// line at the visual-selection boundaries and around the cursor block,
// and each piece gets plain/selected/cursor styling. The cuts reuse the
// highlighter's context-aware token painting, so a cut inside a token
// keeps that token's color on both sides.
func renderLine(e *Editor, lines []string, i int, st Style, selStart, selEnd, cursorLine, cursorCol int) string {
	line := lines[i]
	rb := []rune(line)
	prefix := strings.Join(lines[:i], "\n")

	// cursor char cut positions (byte offsets into the line). A cursor
	// at the end of the line has no char piece; the block renders after.
	cur1, cur2 := -1, -1
	if i == cursorLine && e.focused {
		cur1 = len(string(rb[:cursorCol]))
		cur2 = cur1
		if cur1 < len(line) {
			_, size := utf8.DecodeRuneInString(line[cur1:])
			cur2 = cur1 + size
		}
	}

	// selection cut positions
	sel1, sel2 := -1, -1
	if selStart >= 0 {
		ls := lineRuneStart(lines, i)
		le := ls + len(rb)
		if le > selStart && ls < selEnd {
			if selStart > ls {
				sel1 = len(string(rb[:selStart-ls]))
			}
			if selEnd < le {
				sel2 = len(string(rb[:selEnd-ls]))
			}
		}
	}

	cuts := []int{}
	addCut := func(c int) {
		for j, x := range cuts {
			if x >= c {
				if x == c {
					return
				}
				cuts = append(cuts, 0)
				copy(cuts[j+1:], cuts[j:])
				cuts[j] = c
				return
			}
		}
		cuts = append(cuts, c)
	}
	if sel1 >= 0 {
		addCut(sel1)
	}
	if sel2 >= 0 {
		addCut(sel2)
	}
	if cur1 >= 0 {
		addCut(cur1)
	}
	if cur2 >= 0 && cur2 != cur1 {
		addCut(cur2)
	}
	// the cursor char piece is the one ending at the cur2 cut
	cursorPiece := -1
	for j, c := range cuts {
		if c == cur2 {
			cursorPiece = j
			break
		}
	}

	pieces := e.hl.Split(prefix, line, cuts...)
	var b strings.Builder
	for pi, p := range pieces {
		if pi == cursorPiece && cur1 < cur2 {
			// the cursor char: block over the char (reverse video — the
			// same visual as a selected char inside the selection)
			b.WriteString(st.CursorBlock.Render(line[cur1:cur2]))
			continue
		}
		pieceStart := 0
		if pi > 0 {
			pieceStart = cuts[pi-1]
		}
		pieceEnd := len(rb)
		if pi < len(cuts) {
			pieceEnd = cuts[pi]
		}
		// byte-rune mismatch: piece boundaries above are byte offsets;
		// convert to rune offsets for the selection test
		rs := len([]rune(line[:pieceStart]))
		re := len([]rune(line[:pieceEnd]))
		if selectedIn(lines, i, rs, re, selStart, selEnd) {
			b.WriteString(st.CursorBlock.Render(p))
			continue
		}
		b.WriteString(p)
	}
	if i == cursorLine && e.focused && cur1 >= 0 && cur2 == cur1 {
		// cursor at end of line: a synthetic block cell after the text
		b.WriteString(st.CursorBlock.Render(" "))
	}
	return b.String()
}

// selectedIn reports whether the rune range [pieceStart, pieceEnd) of
// line i falls inside the selection [selStart, selEnd).
func selectedIn(lines []string, i, pieceStart, pieceEnd, selStart, selEnd int) bool {
	if selStart < 0 || pieceStart >= pieceEnd {
		return false
	}
	ls := lineRuneStart(lines, i)
	ps, pe := ls+pieceStart, ls+pieceEnd
	return ps < selEnd && pe > selStart
}

// lineRuneStart returns the rune offset of the start of line i.
func lineRuneStart(lines []string, i int) int {
	total := 0
	for j := 0; j < i; j++ {
		total += len([]rune(lines[j])) + 1
	}
	return total
}

func (e *Editor) placeholderView() string {
	st := e.style()
	p := st.Placeholder.Render(e.placeholder)
	if e.focused {
		p = st.CursorBlock.Render(" ") + p
	}
	if e.height > 1 {
		p += strings.Repeat("\n", e.height-1)
	}
	return p
}
