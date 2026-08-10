package render

import (
	"encoding/json"
	"strings"
	"sync"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
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

var (
	jsonLexerOnce sync.Once
	jsonLexer     chroma.Lexer
)

// HighlightJSON returns src with paint applied to each JSON token; paint
// receives the token kind and its literal and returns the colored form.
// Invalid JSON is returned untouched, so malformed input can never inject
// ANSI into the output. Callers decide the color mapping (the theme).
func HighlightJSON(src string, paint func(Kind, string) string) string {
	if !json.Valid([]byte(src)) {
		return src
	}
	return highlightJSONTokens(src, paint)
}

// HighlightJSONLines paints src line by line, lexing the whole buffer at
// once. Object context survives across lines, so keys keep their key
// color even on a line that starts after the opening brace. There is no
// validity gate — editors need colors while the buffer is half-typed.
func HighlightJSONLines(src string, paint func(Kind, string) string) []string {
	return paintLines(src, getJSONLexer(), jsonKind, paint)
}

// HighlightJSONSplit paints line in the context of prefix (the text
// before it) and returns the colored line split at cut (a byte offset
// into line, clamped). The prefix restores the lexer state, so a cut
// inside a token keeps that token's color on both sides — used to re-
// paint the cursor line around the cursor.
func HighlightJSONSplit(prefix, line string, cut int, paint func(Kind, string) string) (string, string) {
	return paintSplit(prefix, line, cut, getJSONLexer(), jsonKind, paint)
}

func getJSONLexer() chroma.Lexer {
	jsonLexerOnce.Do(func() { jsonLexer = chroma.Coalesce(lexers.Get("json")) })
	return jsonLexer
}

func highlightJSONTokens(src string, paint func(Kind, string) string) string {
	return paintTokens(src, getJSONLexer(), paint, jsonKind)
}

// jsonKind maps chroma token types onto Kind; -1 means pass through.
// chroma emits NameTag for object keys, which is how keys stay distinct
// from value strings.
func jsonKind(t chroma.TokenType) Kind {
	switch t {
	case chroma.NameTag:
		return KindKey
	case chroma.LiteralStringDouble:
		return KindString
	case chroma.KeywordConstant:
		return KindLiteral
	case chroma.LiteralNumberInteger, chroma.LiteralNumberFloat:
		return KindNumber
	case chroma.Punctuation:
		return KindPunctuation
	}
	return Kind(-1)
}

// tokenise runs lexer over src with panic recovery; nil on failure.
// Highlighting must never break live editing, so a lexer panic degrades
// to the uncolored input.
func tokenise(src string, lexer chroma.Lexer) []chroma.Token {
	var toks []chroma.Token
	func() {
		defer func() { recover() }()
		it, err := lexer.Tokenise(nil, src)
		if err == nil && it != nil {
			toks = it.Tokens()
		}
	}()
	return toks
}

func paintOr[K ~int](kind K, lit string, paint func(K, string) string) string {
	if kind < 0 {
		return lit
	}
	return paint(kind, lit)
}

// paintLines lexes src once and returns one painted string per source
// line. A token spanning lines (Lua block comments, long strings) is
// painted per line segment, so every returned line carries balanced ANSI.
func paintLines[K ~int](src string, lexer chroma.Lexer, kindOf func(chroma.TokenType) K, paint func(K, string) string) []string {
	if toks := tokenise(src, lexer); toks != nil {
		lines := []string{""}
		for _, t := range toks {
			segs := strings.Split(t.Value, "\n")
			lines[len(lines)-1] += paintOr(kindOf(t.Type), segs[0], paint)
			for _, seg := range segs[1:] {
				lines = append(lines, paintOr(kindOf(t.Type), seg, paint))
			}
		}
		return lines
	}
	return strings.Split(src, "\n")
}

// paintSplit lexes prefix+line as one stream and returns the colored
// line split at cut (a byte offset into line, clamped). Tokens entirely
// inside the prefix are context only and dropped; tokens fully after the
// cut land in the second piece; a token spanning the prefix boundary or
// the cut keeps its color on both sides.
func paintSplit[K ~int](prefix, line string, cut int, lexer chroma.Lexer, kindOf func(chroma.TokenType) K, paint func(K, string) string) (string, string) {
	if cut < 0 {
		cut = 0
	}
	if cut > len(line) {
		cut = len(line)
	}
	toks := tokenise(prefix+line, lexer)
	if toks == nil {
		return line[:cut], line[cut:]
	}
	start, bound := len(prefix), len(prefix)+cut
	var pre, post strings.Builder
	consumed := 0
	for _, t := range toks {
		v := t.Value
		tokStart, tokEnd := consumed, consumed+len(v)
		consumed = tokEnd
		kind := kindOf(t.Type)
		switch {
		case tokEnd <= start:
			// entirely inside the prefix: lexer context only
		case tokStart < start:
			// spans the prefix|line boundary: keep only the line side
			piece := v[start-tokStart:]
			if tokEnd > bound {
				pre.WriteString(paintOr(kind, piece[:bound-start], paint))
				post.WriteString(paintOr(kind, piece[bound-start:], paint))
			} else {
				pre.WriteString(paintOr(kind, piece, paint))
			}
		case tokEnd <= bound:
			pre.WriteString(paintOr(kind, v, paint))
		case tokStart >= bound:
			post.WriteString(paintOr(kind, v, paint))
		default:
			// spans the cut: keep the color on both sides
			pre.WriteString(paintOr(kind, v[:bound-tokStart], paint))
			post.WriteString(paintOr(kind, v[bound-tokStart:], paint))
		}
	}
	return pre.String(), post.String()
}

// paintTokens rebuilds src by painting every token that maps to a known
// kind; all other tokens (whitespace, comments, errors) pass through
// verbatim.
func paintTokens[K ~int](src string, lexer chroma.Lexer, paint func(K, string) string, kindOf func(chroma.TokenType) K) string {
	toks := tokenise(src, lexer)
	if toks == nil {
		return src
	}
	var b strings.Builder
	b.Grow(len(src) + len(toks)*8)
	for _, t := range toks {
		b.WriteString(paintOr(kindOf(t.Type), t.Value, paint))
	}
	return b.String()
}
