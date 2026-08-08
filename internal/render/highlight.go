package render

import (
	"encoding/json"
	"strings"
)

// Kind is a JSON token category for syntax highlighting.
type Kind int

const (
	KindString Kind = iota
	KindNumber
	KindLiteral // true, false, null
	KindKey
	KindPunctuation
)

type token struct {
	kind  Kind
	start int
	end   int
}

// HighlightJSON returns src with paint applied to each JSON token; paint
// receives the token kind and its literal and returns the colored form.
// Invalid JSON is returned untouched, so malformed input can never inject
// ANSI into the output. Callers decide the color mapping (the theme).
func HighlightJSON(src string, paint func(Kind, string) string) string {
	if !json.Valid([]byte(src)) {
		return src
	}
	toks := lexJSON(src)
	if len(toks) == 0 {
		return src
	}
	var b strings.Builder
	b.Grow(len(src) + len(toks)*8)
	pos := 0
	for _, t := range toks {
		b.WriteString(src[pos:t.start])
		b.WriteString(paint(t.kind, src[t.start:t.end]))
		pos = t.end
	}
	b.WriteString(src[pos:])
	return b.String()
}

// lexJSON scans valid JSON into tokens. A string followed by ':' becomes a
// KindKey; strings in arrays or value position stay KindString.
func lexJSON(src string) []token {
	var toks []token
	last := -1 // index of the most recent token, for key promotion
	n := len(src)
	i := 0
	for i < n {
		c := src[i]
		switch {
		case c == ' ' || c == '\t' || c == '\n' || c == '\r':
			i++
		case c == '"':
			start := i
			i++
			for i < n {
				if src[i] == '\\' {
					i += 2
					continue
				}
				if src[i] == '"' {
					i++
					break
				}
				i++
			}
			last = len(toks)
			toks = append(toks, token{KindString, start, i})
		case c == '{' || c == '}' || c == '[' || c == ']' || c == ',' || c == ':':
			start := i
			i++
			if c == ':' && last >= 0 && toks[last].kind == KindString {
				toks[last].kind = KindKey
			}
			last = len(toks)
			toks = append(toks, token{KindPunctuation, start, i})
		case c == 't' && strings.HasPrefix(src[i:], "true"):
			last = len(toks)
			toks = append(toks, token{KindLiteral, i, i + 4})
			i += 4
		case c == 'f' && strings.HasPrefix(src[i:], "false"):
			last = len(toks)
			toks = append(toks, token{KindLiteral, i, i + 5})
			i += 5
		case c == 'n' && strings.HasPrefix(src[i:], "null"):
			last = len(toks)
			toks = append(toks, token{KindLiteral, i, i + 4})
			i += 4
		case c == '-' || (c >= '0' && c <= '9'):
			start := i
			i++
			for i < n && (src[i] == '.' || src[i] == 'e' || src[i] == 'E' ||
				src[i] == '+' || src[i] == '-' || (src[i] >= '0' && src[i] <= '9')) {
				i++
			}
			last = len(toks)
			toks = append(toks, token{KindNumber, start, i})
		default:
			i++ // unreachable on valid JSON; cannot loop
		}
	}
	return toks
}
