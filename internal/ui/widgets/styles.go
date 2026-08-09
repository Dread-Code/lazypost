package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Colors are package vars set by Theme.Apply() ([[Design - themes]]).
// Read them freely; assign only through a theme.
var (
	ColorPrimary lipgloss.AdaptiveColor
	ColorDim     lipgloss.AdaptiveColor
	ColorSuccess lipgloss.AdaptiveColor
	ColorWarn    lipgloss.AdaptiveColor
	ColorError   lipgloss.AdaptiveColor
	ColorInfo    lipgloss.AdaptiveColor
	ColorAccent  lipgloss.AdaptiveColor
	ColorMuted   lipgloss.AdaptiveColor
	ColorBorder  lipgloss.AdaptiveColor
	// InputColor is text color in inputs/textareas (was hardcoded #FFFFFF).
	InputColor lipgloss.AdaptiveColor
)

// methodPillFg sits on the method badge: the badge background is the
// method's color, which is dark on light terminals and light on dark
// ones, so the text inverts to match.
var methodPillFg = adaptive("#FFFFFF", "#000000")

var methodColors map[string]lipgloss.AdaptiveColor

func MethodStyle(method string) lipgloss.Style {
	c, ok := methodColors[method]
	if !ok {
		c = ColorPrimary
	}
	return lipgloss.NewStyle().Foreground(c).Bold(true)
}

// MethodBadge renders the HTTP method as a colored pill: the strongest
// visual anchor of the request top bar.
func MethodBadge(method string) string {
	c, ok := methodColors[method]
	if !ok {
		c = ColorPrimary
	}
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(methodPillFg).
		Background(c).
		Padding(0, 1).
		Render(method)
}

// EnvBadge renders the active environment ("env: dev") as a pink pill on
// the title bar, mirroring the method badge. The pink hue is the theme's
// accent for status — active envs pop, "env: none" stays muted text.
func EnvBadge(name string) string {
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(methodPillFg).
		Background(ColorAccent).
		Padding(0, 1).
		Render(name)
}

func StatusColor(code int) lipgloss.AdaptiveColor {
	switch {
	case code >= 200 && code < 300:
		return ColorSuccess
	case code >= 300 && code < 400:
		return ColorInfo
	case code >= 400:
		return ColorError
	default:
		return ColorWarn
	}
}

var (
	PaneStyle              lipgloss.Style
	ActivePaneStyle        lipgloss.Style
	ModalStyle             lipgloss.Style
	TitleStyle             lipgloss.Style
	HintStyle              lipgloss.Style
	ErrorStyle             lipgloss.Style
	NoticeStyle            lipgloss.Style
	KeyStyle               lipgloss.Style
	SectionStyle           lipgloss.Style
	LegendTitleStyle       lipgloss.Style
	ActiveLegendTitleStyle lipgloss.Style
	SelectedRowStyle       lipgloss.Style
	TabStyle               lipgloss.Style
	ActiveTabStyle         lipgloss.Style
)

// PaneAccent is the focused look of one section: its border, legend
// title, and active tab all share a single hue, so each pane reads as a
// distinct section while focus stays recognizable (the accent replaces
// the muted border). Modals stay primary; panes wear their own hue.
type PaneAccent struct {
	Active    lipgloss.Style // focused border
	Legend    lipgloss.Style // focused legend title
	ActiveTab lipgloss.Style // active tab in the pane's tab bar
}

// Per-section accents, rebuilt by Theme.Apply(). The hues map onto the
// theme semantics: the collection is the app identity (primary), the
// request editor is information (info), the response is the result
// (success).
var (
	SidebarAccent  PaneAccent
	EditorAccent   PaneAccent
	ResponseAccent PaneAccent
)

// init applies the default theme so styles are valid before any other
// package uses them; main.go may switch themes later via Theme.Apply().
func init() {
	DefaultTheme.Apply()
}

// TabBar renders a tab strip that fits within maxW columns; active is the
// highlighted index and accent colors it (the pane's accent, or nil for
// the default primary — modals). Tabs carry two columns of side padding,
// shrinking to one when the full strip doesn't fit; on the narrowest
// panes the tabs farthest from the active one are dropped (in order) so
// the active tab and its neighbours always stay visible.
func TabBar(tabs []string, active, maxW int, accent *PaneAccent) string {
	activeStyle := ActiveTabStyle
	if accent != nil {
		activeStyle = accent.ActiveTab
	}
	for _, pad := range []int{2, 1} {
		strip := tabRender(tabs, active, pad, activeStyle)
		if lipgloss.Width(strip) <= maxW {
			return strip
		}
	}

	// Keep the active tab and grow the window outward, accepting only
	// candidates that still fit.
	pad := 1
	indices := []int{active}
	for d := 1; ; d++ {
		var cand []int
		cand = append(cand, indices...)
		changed := false
		if l := active - d; l >= 0 {
			cand = append([]int{l}, cand...)
			changed = true
		}
		if r := active + d; r < len(tabs) {
			cand = append(cand, r)
			changed = true
		}
		if !changed {
			break
		}
		if tabWidthIdx(tabs, cand, pad) <= maxW {
			indices = cand
		}
	}
	var s string
	for _, i := range indices {
		style := TabStyle
		if i == active {
			style = activeStyle
		}
		s += style.Padding(0, pad).Render(tabs[i])
	}
	return s
}

// tabRender builds the strip with the given side padding; the active tab
// wears activeStyle.
func tabRender(tabs []string, active, pad int, activeStyle lipgloss.Style) string {
	var s string
	for i, t := range tabs {
		style := TabStyle
		if i == active {
			style = activeStyle
		}
		s += style.Padding(0, pad).Render(t)
	}
	return s
}

// tabWidthIdx returns the width of the given subset of tabs (in index
// order) with the given side padding.
func tabWidthIdx(tabs []string, idx []int, pad int) int {
	w := 0
	for _, i := range idx {
		w += lipgloss.Width(tabs[i]) + pad*2
	}
	return w
}

// Rule renders a horizontal rule spanning width columns in the border
// color. It anchors tab rows and separates content zones.
func Rule(width int) string {
	if width <= 0 {
		return ""
	}
	return lipgloss.NewStyle().Foreground(ColorBorder).Render(strings.Repeat("─", width))
}

// SectionLine renders a section header in the legend language: a dash
// run with the title sitting on it, "── Title ─────────". width caps the
// total line length; a title wider than width is truncated.
func SectionLine(title string, width int) string {
	if width < 1 {
		width = 1
	}
	title = TruncateRunes(title, width-4)
	dash := lipgloss.NewStyle().Foreground(ColorBorder)
	fill := width - 2 - lipgloss.Width(" "+title+" ")
	if fill < 1 {
		fill = 1
	}
	return dash.Render("──") +
		SectionStyle.Render(" "+title+" ") +
		dash.Render(strings.Repeat("─", fill))
}

// TruncateRunes shortens s to at most n runes, appending an ellipsis
// when truncated. Operates on plain text (no ANSI).
func TruncateRunes(s string, n int) string {
	if n <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	if n == 1 {
		return "…"
	}
	return string(r[:n-1]) + "…"
}
