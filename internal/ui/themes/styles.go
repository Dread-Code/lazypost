package themes

import "github.com/charmbracelet/lipgloss"

// methodPillFg sits on the method badge: the badge background is the
// method's color, which is dark on light terminals and light on dark ones.
var methodPillFg = adaptive("#FFFFFF", "#000000")

// PaneAccent is the focused look of one section: its border, legend title,
// and active tab all share a single hue.
type PaneAccent struct {
	Active    lipgloss.Style
	Legend    lipgloss.Style
	ActiveTab lipgloss.Style
}

func tabWidthIdx(tabs []string, idx []int, pad int) int {
	w := 0
	for _, i := range idx {
		w += lipgloss.Width(tabs[i]) + pad*2
	}
	return w
}

// TruncateRunes shortens s to at most n runes, appending an ellipsis when
// truncated. It operates on plain text without ANSI sequences.
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
