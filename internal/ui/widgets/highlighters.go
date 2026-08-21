package ui

import (
	"github.com/Dread-Code/lazypost/lib/codeeditor"
	"github.com/charmbracelet/lipgloss"

	"github.com/Dread-Code/lazypost/internal/render"
	"github.com/Dread-Code/lazypost/internal/ui/themes"
)

// jsonHighlighter colors request bodies. The whole buffer is lexed at
// once so the object context survives across lines and keys keep their
// key color; a line-scoped lex reclassifies every key as a string and
// the body renders in one color ([[Gotcha - request body renders
// uncolored while the response highlights]]).
func jsonHighlighter() codeeditor.Highlighter {
	styles := themes.NewStyles(themes.DefaultTheme)
	return jsonHighlighterWithStyles(&styles)
}

func jsonHighlighterWithStyles(styles *themes.Styles) codeeditor.Highlighter {
	return codeeditorHighlighter{
		lines: func(src string) []string {
			return render.HighlightJSONLines(src, func(kind render.Kind, lit string) string {
				return jsonPaintColors(*styles, kind, lit)
			})
		},
		split: func(prefix, line string, cuts []int) []string {
			return render.HighlightJSONSplitN(prefix, line, cuts, func(kind render.Kind, lit string) string {
				return jsonPaintColors(*styles, kind, lit)
			})
		},
	}
}

// luaHighlighter colors the script hooks (the Scripts tab).
func luaHighlighter() codeeditor.Highlighter {
	styles := themes.NewStyles(themes.DefaultTheme)
	return luaHighlighterWithStyles(&styles)
}

func luaHighlighterWithStyles(styles *themes.Styles) codeeditor.Highlighter {
	return codeeditorHighlighter{
		lines: func(src string) []string {
			return render.HighlightLuaLines(src, func(kind render.LuaKind, lit string) string {
				return luaPaintColors(*styles, kind, lit)
			})
		},
		split: func(prefix, line string, cuts []int) []string {
			return render.HighlightLuaSplitN(prefix, line, cuts, func(kind render.LuaKind, lit string) string {
				return luaPaintColors(*styles, kind, lit)
			})
		},
	}
}

// codeeditorHighlighter adapts lazypost's chroma+theme wiring to the
// codeeditor.Highlighter contract.
type codeeditorHighlighter struct {
	lines func(string) []string
	split func(string, string, []int) []string
}

func (h codeeditorHighlighter) Lines(src string) []string { return h.lines(src) }

func (h codeeditorHighlighter) Split(prefix, line string, cuts ...int) []string {
	return h.split(prefix, line, cuts)
}

// luaPaintColors maps Lua token kinds onto the active theme; identifiers
// stay at the terminal default. Styles are built from the package color
// vars, so a runtime theme switch applies on the next render.
func jsonPaintColors(styles themes.Styles, kind render.Kind, lit string) string {
	var color lipgloss.AdaptiveColor
	switch kind {
	case render.KindKey:
		color = styles.ColorPrimary
	case render.KindString:
		color = styles.ColorSuccess
	case render.KindNumber:
		color = styles.ColorInfo
	case render.KindLiteral:
		color = styles.ColorWarn
	default:
		color = styles.ColorMuted
	}
	return lipgloss.NewStyle().Foreground(color).Render(lit)
}

func luaPaintColors(styles themes.Styles, kind render.LuaKind, lit string) string {
	var color lipgloss.AdaptiveColor
	switch kind {
	case render.LuaKeyword:
		color = styles.ColorPrimary
	case render.LuaString:
		color = styles.ColorSuccess
	case render.LuaComment:
		color = styles.ColorMuted
	case render.LuaNumber:
		color = styles.ColorInfo
	case render.LuaOperator:
		color = styles.ColorDim
	default: // LuaIdentifier
		return lit
	}
	return lipgloss.NewStyle().Foreground(color).Render(lit)
}

// editorStyles keeps the code editors themed: the gutter and placeholder
// follow the active theme. The provider is re-evaluated on every render,
// so a runtime theme switch lands on the next frame.
func editorStyles(styles themes.Styles) codeeditor.Style {
	return codeeditor.Style{
		Gutter:      styles.HintStyle,
		Placeholder: styles.HintStyle,
	}
}
