package ui

import (
	"github.com/Dread-Code/codeeditor"
	"github.com/charmbracelet/lipgloss"

	"lazypost/internal/render"
	"lazypost/internal/ui/themes"
)

// jsonHighlighter colors request bodies. The whole buffer is lexed at
// once so the object context survives across lines and keys keep their
// key color; a line-scoped lex reclassifies every key as a string and
// the body renders in one color ([[Gotcha - request body renders
// uncolored while the response highlights]]).
func jsonHighlighter() codeeditor.Highlighter {
	return codeeditorHighlighter{
		lines: func(src string) []string { return render.HighlightJSONLines(src, highlightJSONColors) },
		split: func(prefix, line string, cut int) (string, string) {
			return render.HighlightJSONSplit(prefix, line, cut, highlightJSONColors)
		},
	}
}

// luaHighlighter colors the script hooks (the Scripts tab).
func luaHighlighter() codeeditor.Highlighter {
	return codeeditorHighlighter{
		lines: func(src string) []string { return render.HighlightLuaLines(src, luaPaintColors) },
		split: func(prefix, line string, cut int) (string, string) {
			return render.HighlightLuaSplit(prefix, line, cut, luaPaintColors)
		},
	}
}

// codeeditorHighlighter adapts lazypost's chroma+theme wiring to the
// codeeditor.Highlighter contract.
type codeeditorHighlighter struct {
	lines func(string) []string
	split func(string, string, int) (string, string)
}

func (h codeeditorHighlighter) Lines(src string) []string { return h.lines(src) }

func (h codeeditorHighlighter) Split(prefix, line string, cut int) (string, string) {
	return h.split(prefix, line, cut)
}

// luaPaintColors maps Lua token kinds onto the active theme; identifiers
// stay at the terminal default. Styles are built from the package color
// vars, so a runtime theme switch applies on the next render.
func luaPaintColors(kind render.LuaKind, lit string) string {
	var color lipgloss.AdaptiveColor
	switch kind {
	case render.LuaKeyword:
		color = themes.ColorPrimary
	case render.LuaString:
		color = themes.ColorSuccess
	case render.LuaComment:
		color = themes.ColorMuted
	case render.LuaNumber:
		color = themes.ColorInfo
	case render.LuaOperator:
		color = themes.ColorDim
	default: // LuaIdentifier
		return lit
	}
	return lipgloss.NewStyle().Foreground(color).Render(lit)
}

// editorStyles keeps the code editors themed: the gutter and placeholder
// follow the active theme. The provider is re-evaluated on every render,
// so a runtime theme switch lands on the next frame.
func editorStyles() codeeditor.Style {
	return codeeditor.Style{
		Gutter:      themes.HintStyle,
		Placeholder: themes.HintStyle,
	}
}
