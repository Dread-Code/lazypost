package ui

import "github.com/charmbracelet/lipgloss"

// Theme is a named set of colors. Every style in the app is rebuilt from
// it by Apply(), so no call site touches colors directly ([[Design -
// themes]]).
type Theme struct {
	Name    string
	Primary lipgloss.AdaptiveColor
	Dim     lipgloss.AdaptiveColor
	Success lipgloss.AdaptiveColor
	Warn    lipgloss.AdaptiveColor
	Error   lipgloss.AdaptiveColor
	Info    lipgloss.AdaptiveColor
	Accent  lipgloss.AdaptiveColor
	Muted   lipgloss.AdaptiveColor
	Border  lipgloss.AdaptiveColor
	Input   lipgloss.AdaptiveColor
	// Field is the background of the URL input box (raised, subtle tones).
	Field lipgloss.AdaptiveColor
	// Selection is the background fill of the highlighted row in lists
	// (sidebar, palette, history); OnSelection is the text color on it.
	Selection   lipgloss.AdaptiveColor
	OnSelection lipgloss.AdaptiveColor
	Methods     map[string]lipgloss.AdaptiveColor
}

// adaptive is a tiny helper for the palette literals below.
func adaptive(light, dark string) lipgloss.AdaptiveColor {
	return lipgloss.AdaptiveColor{Light: light, Dark: dark}
}

// DefaultTheme is the fallback when no theme is selected. Dracula.
var DefaultTheme = Themes["dracula"]

// Themes are the embedded presets. User themes (YAML in
// ~/.config/lazypost/themes/) are planned but not yet loaded — presets
// only for now.
var Themes = map[string]Theme{
	"dracula": {
		Name:        "dracula",
		Primary:     adaptive("#5A56E0", "#BD93F9"), // purple
		Dim:         adaptive("#8A8A8A", "#6272A4"),
		Success:     adaptive("#008000", "#50FA7B"),
		Warn:        adaptive("#B58900", "#F1FA8C"),
		Error:       adaptive("#CC0000", "#FF5555"),
		Info:        adaptive("#0066CC", "#8BE9FD"),
		Accent:      adaptive("#F92672", "#FF79C6"),
		Muted:       adaptive("#999999", "#6272A4"),
		Border:      adaptive("#DDDDDD", "#44475A"),
		Input:       adaptive("#44475A", "#F8F8F2"),
		Field:       adaptive("#E9E9F0", "#343746"),
		Selection:   adaptive("#5A56E0", "#BD93F9"),
		OnSelection: adaptive("#FFFFFF", "#282A36"),
		Methods: map[string]lipgloss.AdaptiveColor{
			"GET":     adaptive("#008000", "#50FA7B"),
			"POST":    adaptive("#B58900", "#F1FA8C"),
			"PUT":     adaptive("#0066CC", "#8BE9FD"),
			"PATCH":   adaptive("#875FD7", "#FF79C6"),
			"DELETE":  adaptive("#CC0000", "#FF5555"),
			"HEAD":    adaptive("#999999", "#6272A4"),
			"OPTIONS": adaptive("#999999", "#6272A4"),
		},
	},

	"catppuccin": {
		Name:        "catppuccin",
		Primary:     adaptive("#8839EF", "#CBA6F7"), // mauve
		Dim:         adaptive("#6C7086", "#45475A"),
		Success:     adaptive("#40A02B", "#A6E3A1"),
		Warn:        adaptive("#DF8E1D", "#F9E2AF"),
		Error:       adaptive("#D20F39", "#F38BA8"),
		Info:        adaptive("#1E66F5", "#89B4FA"),
		Accent:      adaptive("#EA76CB", "#F5C2E7"),
		Muted:       adaptive("#9CA0B0", "#6C7086"),
		Border:      adaptive("#CCD0DA", "#313244"),
		Input:       adaptive("#4C4F69", "#CDD6F4"),
		Field:       adaptive("#DCE0E8", "#313244"),
		Selection:   adaptive("#8839EF", "#CBA6F7"),
		OnSelection: adaptive("#FFFFFF", "#1E1E2E"),
		Methods: map[string]lipgloss.AdaptiveColor{
			"GET":     adaptive("#40A02B", "#A6E3A1"),
			"POST":    adaptive("#DF8E1D", "#F9E2AF"),
			"PUT":     adaptive("#1E66F5", "#89B4FA"),
			"PATCH":   adaptive("#8839EF", "#CBA6F7"),
			"DELETE":  adaptive("#D20F39", "#F38BA8"),
			"HEAD":    adaptive("#9CA0B0", "#6C7086"),
			"OPTIONS": adaptive("#9CA0B0", "#6C7086"),
		},
	},

	"solarized": {
		Name:        "solarized",
		Primary:     adaptive("#268BD2", "#268BD2"), // blue
		Dim:         adaptive("#93A1A1", "#586E75"),
		Success:     adaptive("#859900", "#859900"),
		Warn:        adaptive("#B58900", "#B58900"),
		Error:       adaptive("#DC322F", "#DC322F"),
		Info:        adaptive("#2AA198", "#2AA198"),
		Accent:      adaptive("#D33682", "#D33682"),
		Muted:       adaptive("#93A1A1", "#586E75"),
		Border:      adaptive("#EEE8D5", "#073642"),
		Input:       adaptive("#657B83", "#839496"),
		Field:       adaptive("#EEE8D5", "#073642"),
		Selection:   adaptive("#268BD2", "#268BD2"),
		OnSelection: adaptive("#FDF6E3", "#002B36"),
		Methods: map[string]lipgloss.AdaptiveColor{
			"GET":     adaptive("#859900", "#859900"),
			"POST":    adaptive("#B58900", "#B58900"),
			"PUT":     adaptive("#268BD2", "#268BD2"),
			"PATCH":   adaptive("#6C71C4", "#6C71C4"),
			"DELETE":  adaptive("#DC322F", "#DC322F"),
			"HEAD":    adaptive("#93A1A1", "#586E75"),
			"OPTIONS": adaptive("#93A1A1", "#586E75"),
		},
	},
}

// ThemeByName returns the named theme, falling back to DefaultTheme.
func ThemeByName(name string) Theme {
	if t, ok := Themes[name]; ok {
		return t
	}
	return DefaultTheme
}

// ThemeNames returns the embedded preset names in stable order.
func ThemeNames() []string {
	return []string{"dracula", "catppuccin", "solarized"}
}

// Apply rebuilds the package styles from t. This is the single place the
// global styles are set, so switching themes at runtime just calls it.
func (t Theme) Apply() {
	ColorPrimary = t.Primary
	ColorDim = t.Dim
	ColorSuccess = t.Success
	ColorWarn = t.Warn
	ColorError = t.Error
	ColorInfo = t.Info
	ColorAccent = t.Accent
	ColorMuted = t.Muted
	ColorBorder = t.Border
	ColorField = t.Field

	methodColors = map[string]lipgloss.AdaptiveColor{}
	for k, v := range t.Methods {
		methodColors[k] = v
	}

	PaneStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Border)

	ActivePaneStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Primary)

	// Overlays sit on top of the frame and are always "focused", so they
	// share the active border language.
	ModalStyle = ActivePaneStyle

	TitleStyle = lipgloss.NewStyle().Bold(true).Foreground(t.Primary)
	HintStyle = lipgloss.NewStyle().Foreground(t.Muted)
	ErrorStyle = lipgloss.NewStyle().Foreground(t.Error)
	NoticeStyle = lipgloss.NewStyle().Foreground(t.Success)
	KeyStyle = lipgloss.NewStyle().Bold(true).Foreground(t.Info)
	SectionStyle = lipgloss.NewStyle().Bold(true).Foreground(t.Primary)
	VersionStyle = lipgloss.NewStyle().Foreground(t.Input)
	LegendTitleStyle = lipgloss.NewStyle().Bold(true).Foreground(t.Muted)
	ActiveLegendTitleStyle = lipgloss.NewStyle().Bold(true).Foreground(t.Primary)
	SelectedRowStyle = lipgloss.NewStyle().
		Foreground(t.OnSelection).
		Background(t.Selection)

	TabStyle = lipgloss.NewStyle().
		Padding(0, 2).
		Foreground(t.Muted)

	ActiveTabStyle = lipgloss.NewStyle().
		Padding(0, 2).
		Foreground(t.Primary).
		Bold(true).
		Underline(true)

	// The URL input box: the raised background is part of every cell of
	// the field (padding + styled input), so it never has holes.
	FieldStyle = lipgloss.NewStyle().
		Background(t.Field)

	// URL token styles ([[Design - url bar]]).
	URLSchemeStyle = lipgloss.NewStyle().Foreground(t.Info)
	URLUserInfoStyle = lipgloss.NewStyle().Foreground(t.Warn)
	URLHostStyle = lipgloss.NewStyle().Foreground(t.Primary)
	URLPortStyle = lipgloss.NewStyle().Foreground(t.Dim)
	URLPathStyle = lipgloss.NewStyle()
	URLQueryKeyStyle = lipgloss.NewStyle().Foreground(t.Info)
	URLQueryValueStyle = lipgloss.NewStyle()
	URLQuerySepStyle = lipgloss.NewStyle().Foreground(t.Dim)
	URLFragmentStyle = lipgloss.NewStyle().Foreground(t.Dim).Italic(true)
	URLVarStyle = lipgloss.NewStyle().Foreground(t.Warn)

	// Per-section accents: one hue per pane for its focused border, legend
	// title, and active tab. The collection is the app identity (primary),
	// the request editor is information (info), the response is the result
	// (success) — modals keep the plain primary accent.
	SidebarAccent = paneAccent(t.Primary)
	EditorAccent = paneAccent(t.Info)
	ResponseAccent = paneAccent(t.Success)

	InputColor = t.Input
}

// paneAccent bundles the three focused styles of a section around one
// accent color.
func paneAccent(accent lipgloss.AdaptiveColor) PaneAccent {
	return PaneAccent{
		Active: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(accent),
		Legend: lipgloss.NewStyle().Bold(true).Foreground(accent),
		ActiveTab: lipgloss.NewStyle().
			Padding(0, 2).
			Foreground(accent).
			Bold(true).
			Underline(true),
	}
}
