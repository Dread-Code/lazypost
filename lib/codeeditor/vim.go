package codeeditor

import "unicode"

// Mode is the editor's editing mode.
type Mode int

const (
	// ModeInsert is the default: plain text editing (typing, backspace,
	// arrows, home/end, enter). Esc switches to ModeNormal.
	ModeInsert Mode = iota
	// ModeNormal is vim's normal mode: h j k l motions, w b e, 0 $ ^,
	// gg G, %, operators d/y/x, paste p/P, numeric counts, v/V enter
	// visual mode, i/a/A/I/o/O enter insert mode.
	ModeNormal
	// ModeVisualChar is vim's characterwise visual mode: motions extend
	// the selection between the anchor and the cursor; y yanks it
	// (firing the yank hook), d/x delete it, esc/v leave.
	ModeVisualChar
	// ModeVisualLine is vim's linewise visual mode; the selection snaps
	// to whole lines.
	ModeVisualLine
)

func (m Mode) String() string {
	switch m {
	case ModeInsert:
		return "INSERT"
	case ModeNormal:
		return "NORMAL"
	case ModeVisualChar:
		return "VISUAL"
	case ModeVisualLine:
		return "VISUAL"
	}
	return "UNKNOWN"
}

// wordRunes is the "word" definition used by w/b/e motions: a maximal
// run of non-space runes (punctuation counts as part of a word — a
// simplified first cut).
func isWordRune(r rune) bool { return !unicode.IsSpace(r) }

// lineStartRune returns the rune offset of the start of the line
// containing pos.
func (e *Editor) lineStartRune(r []rune, pos int) int {
	if pos > len(r) {
		pos = len(r)
	}
	for pos > 0 && r[pos-1] != '\n' {
		pos--
	}
	return pos
}

// lineEndRune returns the rune offset just past the last non-newline
// rune of the line containing pos (i.e. at the \n, or len(r)).
func (e *Editor) lineEndRune(r []rune, pos int) int {
	if pos > len(r) {
		pos = len(r)
	}
	for pos < len(r) && r[pos] != '\n' {
		pos++
	}
	return pos
}

// lineEndInclNewline is lineEndRune extended across a trailing newline.
func (e *Editor) lineEndInclNewline(r []rune, pos int) int {
	end := e.lineEndRune(r, pos)
	if end < len(r) {
		return end + 1
	}
	return end
}

// firstNonBlank returns the rune offset of the first non-space rune on
// the line containing pos.
func (e *Editor) firstNonBlank(r []rune, pos int) int {
	start := e.lineStartRune(r, pos)
	for start < len(r) && r[start] != '\n' && isWordRune(r[start]) == false && r[start] != ' ' {
		start++
	}
	for start < len(r) && r[start] == ' ' {
		start++
	}
	return start
}

// nextWord returns the rune offset of the start of the next word.
func nextWord(r []rune, pos int) int {
	n := len(r)
	if pos >= n {
		return n
	}
	i := pos
	if r[i] == ' ' {
		for i < n && r[i] == ' ' {
			i++
		}
		return i
	}
	for i < n && r[i] != ' ' {
		i++
	}
	for i < n && r[i] == ' ' {
		i++
	}
	return i
}

// prevWord returns the rune offset of the start of the current or
// previous word.
func prevWord(r []rune, pos int) int {
	if pos <= 0 {
		return 0
	}
	i := pos
	for i > 0 && r[i-1] == ' ' {
		i--
	}
	for i > 0 && r[i-1] != ' ' {
		i--
	}
	return i
}

// wordEnd returns the rune offset just past the end of the current or
// next word.
func wordEnd(r []rune, pos int) int {
	n := len(r)
	if pos >= n {
		return n
	}
	i := pos
	if r[i] == ' ' {
		for i < n && r[i] == ' ' {
			i++
		}
		for i < n && r[i] != ' ' {
			i++
		}
		return i
	}
	for i < n && r[i] != ' ' {
		i++
	}
	return i
}

// matchBracket returns the rune offset of the partner of the bracket at
// pos, scanning forward for the next bracket if pos is not one itself;
// -1 if unmatched.
func matchBracket(r []rune, pos int) int {
	closer := map[rune]rune{'(': ')', '[': ']', '{': '}'}
	opener := map[rune]rune{')': '(', ']': '[', '}': '{'}
	if pos > len(r) {
		pos = len(r)
	}
	for ; pos < len(r); pos++ {
		if c, ok := closer[r[pos]]; ok {
			// opener at pos: scan forward for the matching closer
			depth := 1
			for i := pos + 1; i < len(r); i++ {
				switch {
				case r[i] == c:
					depth--
					if depth == 0 {
						return i
					}
				case r[i] == r[pos]:
					depth++
				}
			}
			return -1
		}
		if o, ok := opener[r[pos]]; ok {
			// closer at pos: scan backward for the matching opener
			depth := 1
			for i := pos - 1; i >= 0; i-- {
				switch {
				case r[i] == o:
					depth--
					if depth == 0 {
						return i
					}
				case r[i] == r[pos]:
					depth++
				}
			}
			return -1
		}
	}
	return -1
}

// selection returns the normalized rune range of the visual selection
// [start, end).
func (e *Editor) selection(r []rune) (int, int) {
	start, end := e.anchor, e.cursor
	if e.mode == ModeVisualLine {
		start = e.lineStartRune(r, start)
		end = e.lineEndInclNewline(r, end)
	}
	if start > end {
		start, end = end, start
	}
	return start, end
}

// yankSelection copies the visual selection into the register (firing
// the yank hook) and returns to normal mode with the cursor at the end
// of the selection.
func (e *Editor) yankSelection(r []rune) {
	start, end := e.selection(r)
	e.reg = string(r[start:end])
	if e.yank != nil {
		e.yank(e.reg)
	}
	e.cursor = end
	e.mode = ModeNormal
}

// deleteSelection removes the visual selection and returns to normal
// mode with the cursor at the selection start.
func (e *Editor) deleteSelection(r []rune) {
	start, end := e.selection(r)
	e.reg = string(r[start:end])
	e.value = string(concat(r[:start], r[end:]))
	e.cursor = start
	e.mode = ModeNormal
}

// deleteRange removes the rune range [start, end), storing it in the
// register, and returns the new cursor offset.
func (e *Editor) deleteRange(r []rune, start, end int) int {
	e.reg = string(r[start:end])
	e.value = string(concat(r[:start], r[end:]))
	return start
}
