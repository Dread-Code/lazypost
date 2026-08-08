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
	jsonLexerOnce.Do(func() { jsonLexer = chroma.Coalesce(lexers.Get("json")) })
	return paintTokens(src, jsonLexer, paint, jsonKind)
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

// paintTokens rebuilds src by painting every token that maps to a known
// kind; all other tokens (whitespace, comments, errors) pass through
// verbatim. A lexer panic degrades to the uncolored input: highlighting
// must never break live editing.
func paintTokens[K ~int](src string, lexer chroma.Lexer, paint func(K, string) string, kindOf func(chroma.TokenType) K) string {
	var toks []chroma.Token
	func() {
		defer func() { recover() }()
		it, err := lexer.Tokenise(nil, src)
		if err == nil && it != nil {
			toks = it.Tokens()
		}
	}()
	if toks == nil {
		return src
	}
	var b strings.Builder
	b.Grow(len(src) + len(toks)*8)
	for _, t := range toks {
		if k := kindOf(t.Type); k >= 0 {
			b.WriteString(paint(k, t.Value))
		} else {
			b.WriteString(t.Value)
		}
	}
	return b.String()
}
