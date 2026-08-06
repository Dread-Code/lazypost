package ui

import "github.com/charmbracelet/lipgloss"

var (
	ColorPrimary = lipgloss.AdaptiveColor{Light: "#5A56E0", Dark: "#7AA2F7"}
	ColorDim     = lipgloss.AdaptiveColor{Light: "#888888", Dark: "#666666"}
	ColorSuccess = lipgloss.AdaptiveColor{Light: "#008000", Dark: "#9ECE6A"}
	ColorWarn    = lipgloss.AdaptiveColor{Light: "#B58900", Dark: "#E0AF68"}
	ColorError   = lipgloss.AdaptiveColor{Light: "#CC0000", Dark: "#F7768E"}
	ColorInfo    = lipgloss.AdaptiveColor{Light: "#0066CC", Dark: "#7DCFFF"}
	ColorMuted   = lipgloss.AdaptiveColor{Light: "#999999", Dark: "#565F89"}
)

var methodColors = map[string]lipgloss.AdaptiveColor{
	"GET":     ColorSuccess,
	"POST":    ColorWarn,
	"PUT":     ColorInfo,
	"PATCH":   {Light: "#875FD7", Dark: "#BB9AF7"},
	"DELETE":  ColorError,
	"HEAD":    ColorMuted,
	"OPTIONS": ColorMuted,
}

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
	PaneStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.AdaptiveColor{Light: "#DDDDDD", Dark: "#3B3B3B"})

	ActivePaneStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorPrimary)

	TitleStyle  = lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary)
	HintStyle   = lipgloss.NewStyle().Foreground(ColorMuted)
	ErrorStyle  = lipgloss.NewStyle().Foreground(ColorError)
	NoticeStyle = lipgloss.NewStyle().Foreground(ColorSuccess)

	TabStyle = lipgloss.NewStyle().
			Padding(0, 2).
			Foreground(ColorMuted)

	ActiveTabStyle = lipgloss.NewStyle().
			Padding(0, 2).
			Foreground(ColorPrimary).
			Bold(true).
			Underline(true)
)

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
