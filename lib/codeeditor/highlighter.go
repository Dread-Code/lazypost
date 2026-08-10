package codeeditor

import "strings"

// Highlighter paints an entire buffer one line at a time and can
// re-paint a single line around the cursor. Both methods receive the
// whole buffer, so lexer state survives across lines: JSON keys stay
// keys, Lua block comments and long strings keep their color on
// continuation lines.
//
// Implementations own the tokenizer and the color mapping (a theme);
// the editor itself is lexer-agnostic.
type Highlighter interface {
	// Lines paints src and returns one colored string per source line.
	Lines(src string) []string
	// Split paints line in the context of prefix and returns the colored
	// line split at cut (a byte offset into line, clamped): the cursor
	// seam. The prefix restores the token state, so a cut inside a token
	// keeps the same color on both sides.
	Split(prefix, line string, cut int) (string, string)
}

// IdentityHighlighter passes every line through unchanged — the default
// for editors without syntax coloring.
func IdentityHighlighter() Highlighter {
	return identityHighlighter{}
}

type identityHighlighter struct{}

func (identityHighlighter) Lines(src string) []string { return strings.Split(src, "\n") }

func (identityHighlighter) Split(prefix, line string, cut int) (string, string) {
	if cut < 0 {
		cut = 0
	}
	if cut > len(line) {
		cut = len(line)
	}
	return line[:cut], line[cut:]
}
