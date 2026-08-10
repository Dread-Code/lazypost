package codeeditor

import (
	"strings"
	"unicode/utf8"
)

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
	width, cut := 0, 0
	need := n - 1
	for i := 0; i < len(s); {
		if s[i] == 0x1b {
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
			i = j
			continue
		}
		_, size := utf8.DecodeRuneInString(s[i:])
		width++
		if width == need {
			cut = i + size
		}
		i += size
	}
	if width <= n {
		return s
	}
	truncated := s[:cut]
	if strings.Contains(truncated, "\x1b[") {
		// the cut dropped the token's reset; append one so the next
		// line never inherits a stale color
		return truncated + "…\x1b[0m"
	}
	return truncated + "…"
}
