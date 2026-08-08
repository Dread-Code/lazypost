package render

import (
	"sync"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
)

// LuaKind is a Lua token category for syntax highlighting.
type LuaKind int

const (
	LuaKeyword LuaKind = iota
	LuaString
	LuaComment
	LuaNumber
	LuaIdentifier
	LuaOperator
)

var (
	luaLexerOnce sync.Once
	luaLexer     chroma.Lexer
)

// HighlightLua returns src with paint applied to each Lua token; paint
// receives the token kind and its literal and returns the colored form.
// Unlike JSON there is no validity gate: scripts are always mid-edit, so
// unterminated strings and comments are tolerated and any residual input
// passes through uncolored rather than failing.
func HighlightLua(src string, paint func(LuaKind, string) string) string {
	luaLexerOnce.Do(func() { luaLexer = chroma.Coalesce(lexers.Get("lua")) })
	return paintTokens(src, luaLexer, paint, luaKind)
}

// luaKind maps chroma token types onto LuaKind; -1 means pass through.
// Family ranges (types.go): Keyword [1000,2000), Name [2000,3100),
// LiteralString [3100,3200), LiteralNumber [3200,4000), Operator 4000+,
// Punctuation 5000+, Comment [6000,7000). Note that the Name range must
// end at LiteralString, not Operator, or the literal groups fall into it.
func luaKind(t chroma.TokenType) LuaKind {
	switch {
	case t >= chroma.Keyword && t < chroma.Name:
		return LuaKeyword
	case t >= chroma.Name && t < chroma.LiteralString:
		return LuaIdentifier
	case t >= chroma.LiteralString && t < chroma.LiteralNumber:
		return LuaString
	case t >= chroma.LiteralNumber && t < chroma.Operator:
		return LuaNumber
	case t >= chroma.Operator && t < chroma.Comment:
		return LuaOperator
	case t >= chroma.Comment && t < chroma.Generic:
		return LuaComment
	}
	return LuaKind(-1)
}
