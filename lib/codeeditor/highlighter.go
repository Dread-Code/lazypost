package codeeditor

import (
	"sort"
	"strings"
)

// Highlighter paints an entire buffer one line at a time and can
// re-paint a single line around arbitrary cut points. Both methods
// receive the whole buffer, so lexer state survives across lines: JSON
// keys stay keys, Lua block comments and long strings keep their color
// on continuation lines.
//
// Implementations own the tokenizer and the color mapping (a theme);
// the editor itself is lexer-agnostic.
type Highlighter interface {
	// Lines paints src and returns one colored string per source line.
	Lines(src string) []string
	// Split paints line in the context of prefix and returns the colored
	// line cut at the given byte offsets into line (clamped, sorted,
	// deduplicated): n cuts yield n+1 pieces whose concatenation is the
	// full painted line. prefix must be the exact preceding text
	// INCLUDING the trailing newline — tokens that terminate at a line
	// boundary (a `--` Lua comment, a single-line string) would swallow
	// the line otherwise. The prefix restores the token state, so a cut
	// inside a token keeps that token's color on both sides — the cursor
	// seam and visual-selection boundaries are both expressed as cuts.
	Split(prefix, line string, cuts ...int) []string
}

// IdentityHighlighter passes every line through unchanged — the default
// for editors without syntax coloring.
func IdentityHighlighter() Highlighter {
	return identityHighlighter{}
}

type identityHighlighter struct{}

func (identityHighlighter) Lines(src string) []string { return strings.Split(src, "\n") }

func (identityHighlighter) Split(prefix, line string, cuts ...int) []string {
	lineLen := len(line)
	sorted := make([]int, 0, len(cuts))
	for _, c := range cuts {
		if c < 0 {
			c = 0
		}
		if c > lineLen {
			c = lineLen
		}
		sorted = append(sorted, c)
	}
	sort.Ints(sorted)
	uniq := sorted[:0]
	for _, c := range sorted {
		if len(uniq) == 0 || c > uniq[len(uniq)-1] {
			uniq = append(uniq, c)
		}
	}
	pieces := make([]string, 0, len(uniq)+1)
	prev := 0
	for _, c := range uniq {
		pieces = append(pieces, line[prev:c])
		prev = c
	}
	return append(pieces, line[prev:])
}
