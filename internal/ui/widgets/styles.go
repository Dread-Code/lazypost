package ui

import "github.com/charmbracelet/lipgloss"

// Colors are package vars set by Theme.Apply() ([[Design - themes]]).
// Read them freely; assign only through a theme.
var (
	ColorPrimary lipgloss.AdaptiveColor
	ColorDim     lipgloss.AdaptiveColor
	ColorSuccess lipgloss.AdaptiveColor
	ColorWarn    lipgloss.AdaptiveColor
	ColorError   lipgloss.AdaptiveColor
	ColorInfo    lipgloss.AdaptiveColor
	ColorMuted   lipgloss.AdaptiveColor
	// InputColor is text color in inputs/textareas (was hardcoded #FFFFFF).
	InputColor lipgloss.AdaptiveColor
)

var methodColors map[string]lipgloss.AdaptiveColor

func MethodStyle(method string) lipgloss.Style {
	c, ok := methodColors[method]
	if !ok {
		c = ColorPrimary
	}
	return lipgloss.NewStyle().Foreground(c).Bold(true)
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
	PaneStyle       lipgloss.Style
	ActivePaneStyle lipgloss.Style
	TitleStyle      lipgloss.Style
	HintStyle       lipgloss.Style
	ErrorStyle      lipgloss.Style
	NoticeStyle     lipgloss.Style
	TabStyle        lipgloss.Style
	ActiveTabStyle  lipgloss.Style
)

// init applies the default theme so styles are valid before any other
// package uses them; main.go may switch themes later via Theme.Apply().
func init() {
	DefaultTheme.Apply()
}

// TabBar renders a simple tab strip; active is the highlighted index.
func TabBar(tabs []string, active int) string {
	var s string
	for i, t := range tabs {
		if i == active {
			s += ActiveTabStyle.Render(t)
		} else {
			s += TabStyle.Render(t)
		}
	}
	return s
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
