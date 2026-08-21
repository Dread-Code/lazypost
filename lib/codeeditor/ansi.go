package codeeditor

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// ansiSeqLen returns the byte length of the escape sequence starting at
// s[i] (assumes s[i] == 0x1b), covering CSI sequences and bare ESC +
// final byte; 1 if malformed.
func ansiSeqLen(s string, i int) int {
	j := i + 1
	if j < len(s) && s[j] == '[' {
		j++ // CSI intro
		for j < len(s) && s[j] >= 0x20 && s[j] <= 0x3f {
			j++ // parameters + intermediates
		}
	}
	if j < len(s) && s[j] >= 0x40 && s[j] <= 0x7e {
		j++ // final byte
	}
	return j - i
}

// StripAnsi removes every ANSI escape sequence from s, keeping the
// visible text — used to repaint selected pieces with a uniform style
// (a wrapped token keeps its own colors and resets, which would break
// the selection after the first token).
func StripAnsi(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); {
		if s[i] == 0x1b {
			i += ansiSeqLen(s, i)
			continue
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}

// TruncateRunesAnsi shortens s to at most n visible runes, appending an
// ellipsis when truncated. ANSI escape sequences count as zero width and
// are copied whole, so colored content is never clipped mid-sequence.
func TruncateRunesAnsi(s string, n int) string {
	if n <= 0 {
		return ""
	}
	if n == 1 {
		return "…"
	}
	return ansi.Truncate(s, n, "…")
}
