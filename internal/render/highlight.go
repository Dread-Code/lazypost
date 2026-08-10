package render

import (
	"encoding/json"
	"sort"
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
	pieces := HighlightJSONSplitN(prefix, line, []int{cut}, paint)
	return pieces[0], pieces[1]
}

// HighlightJSONSplitN paints line in the context of prefix and returns
// the colored line cut at the given byte offsets into line (clamped,
// sorted, deduplicated): n cuts yield n+1 pieces whose concatenation is
// the full painted line. The cursor seam and visual-selection
// boundaries are both expressed as cuts.
func HighlightJSONSplitN(prefix, line string, cuts []int, paint func(Kind, string) string) []string {
	return paintPieces(prefix, line, cuts, getJSONLexer(), jsonKind, paint)
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
// inside the prefix are context only and dropped; tokens fully after
// the cut land in the second piece; a token spanning the prefix boundary
// or the cut keeps its color on both sides.
func paintSplit[K ~int](prefix, line string, cut int, lexer chroma.Lexer, kindOf func(chroma.TokenType) K, paint func(K, string) string) (string, string) {
	pieces := paintPieces(prefix, line, []int{cut}, lexer, kindOf, paint)
	return pieces[0], pieces[1]
}

// paintPieces lexes prefix+line as one stream and returns the colored
// line cut at the given byte offsets into line (clamped, sorted,
// deduplicated): n cuts yield n+1 pieces whose concatenation is the
// full painted line. Tokens entirely inside the prefix are context only
// and dropped; a token spanning any cut keeps its color on both sides.
func paintPieces[K ~int](prefix, line string, cuts []int, lexer chroma.Lexer, kindOf func(chroma.TokenType) K, paint func(K, string) string) []string {
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
	pieces := make([]string, len(uniq)+1)
	if toks := tokenise(prefix+line, lexer); toks != nil {
		bounds := make([]int, 0, len(uniq)+2)
		bounds = append(bounds, len(prefix))
		for _, c := range uniq {
			bounds = append(bounds, len(prefix)+c)
		}
		bounds = append(bounds, len(prefix)+lineLen)
		consumed := 0
		for _, t := range toks {
			v := t.Value
			tokStart, tokEnd := consumed, consumed+len(v)
			consumed = tokEnd
			kind := kindOf(t.Type)
			for p := 0; p < len(bounds)-1; p++ {
				s := max(tokStart, bounds[p])
				e := min(tokEnd, bounds[p+1])
				if s < e {
					pieces[p] += paintOr(kind, v[s-tokStart:e-tokStart], paint)
				}
			}
		}
		return pieces
	}
	prev := 0
	for i, c := range uniq {
		pieces[i] = line[prev:c]
		prev = c
	}
	pieces[len(uniq)] = line[prev:]
	return pieces
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
