package themes

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Styles is the compiled rendering state for one theme. It is a value so
// multiple Bubble Tea models can render independently without mutating shared
// active styles.
type Styles struct {
	ColorPrimary lipgloss.AdaptiveColor
	ColorDim     lipgloss.AdaptiveColor
	ColorSuccess lipgloss.AdaptiveColor
	ColorWarn    lipgloss.AdaptiveColor
	ColorError   lipgloss.AdaptiveColor
	ColorInfo    lipgloss.AdaptiveColor
	ColorAccent  lipgloss.AdaptiveColor
	ColorMuted   lipgloss.AdaptiveColor
	ColorBorder  lipgloss.AdaptiveColor
	ColorField   lipgloss.AdaptiveColor
	InputColor   lipgloss.AdaptiveColor

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
	VersionStyle           lipgloss.Style
	TabStyle               lipgloss.Style
	ActiveTabStyle         lipgloss.Style
	FieldStyle             lipgloss.Style

	URLSchemeStyle     lipgloss.Style
	URLUserInfoStyle   lipgloss.Style
	URLHostStyle       lipgloss.Style
	URLPortStyle       lipgloss.Style
	URLPathStyle       lipgloss.Style
	URLQueryKeyStyle   lipgloss.Style
	URLQueryValueStyle lipgloss.Style
	URLQuerySepStyle   lipgloss.Style
	URLFragmentStyle   lipgloss.Style
	URLVarStyle        lipgloss.Style

	SidebarAccent  PaneAccent
	EditorAccent   PaneAccent
	ResponseAccent PaneAccent

	methodColors map[string]lipgloss.AdaptiveColor
}

// NewStyles compiles a theme into an independent rendering snapshot.
func NewStyles(t Theme) Styles {
	s := Styles{
		ColorPrimary: t.Primary,
		ColorDim:     t.Dim,
		ColorSuccess: t.Success,
		ColorWarn:    t.Warn,
		ColorError:   t.Error,
		ColorInfo:    t.Info,
		ColorAccent:  t.Accent,
		ColorMuted:   t.Muted,
		ColorBorder:  t.Border,
		ColorField:   t.Field,
		InputColor:   t.Input,
		methodColors: copyColorMap(t.Methods),
		PaneStyle: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(t.Border),
		ActivePaneStyle: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(t.Primary),
		TitleStyle:  lipgloss.NewStyle().Bold(true).Foreground(t.Primary),
		HintStyle:   lipgloss.NewStyle().Foreground(t.Muted),
		ErrorStyle:  lipgloss.NewStyle().Foreground(t.Error),
		NoticeStyle: lipgloss.NewStyle().Foreground(t.Success),
		KeyStyle:    lipgloss.NewStyle().Bold(true).Foreground(t.Key),
		SectionStyle: lipgloss.NewStyle().
			Bold(true).Foreground(t.Primary),
		VersionStyle: lipgloss.NewStyle().Foreground(t.Input),
		LegendTitleStyle: lipgloss.NewStyle().
			Bold(true).Foreground(t.Muted),
		ActiveLegendTitleStyle: lipgloss.NewStyle().
			Bold(true).Foreground(t.Primary),
		SelectedRowStyle: lipgloss.NewStyle().
			Foreground(t.OnSelection).Background(t.Selection),
		TabStyle: lipgloss.NewStyle().
			Padding(0, 2).Foreground(t.Muted),
		ActiveTabStyle: lipgloss.NewStyle().
			Padding(0, 2).Foreground(t.Primary).Bold(true).Underline(true),
		FieldStyle:         lipgloss.NewStyle().Background(t.Field),
		URLSchemeStyle:     lipgloss.NewStyle().Foreground(t.Info),
		URLUserInfoStyle:   lipgloss.NewStyle().Foreground(t.Warn),
		URLHostStyle:       lipgloss.NewStyle().Foreground(t.Primary),
		URLPortStyle:       lipgloss.NewStyle().Foreground(t.Dim),
		URLPathStyle:       lipgloss.NewStyle(),
		URLQueryKeyStyle:   lipgloss.NewStyle().Foreground(t.Info),
		URLQueryValueStyle: lipgloss.NewStyle(),
		URLQuerySepStyle:   lipgloss.NewStyle().Foreground(t.Dim),
		URLFragmentStyle:   lipgloss.NewStyle().Foreground(t.Dim).Italic(true),
		URLVarStyle:        lipgloss.NewStyle().Foreground(t.Warn),
	}
	s.ModalStyle = s.ActivePaneStyle
	s.SidebarAccent = paneAccent(t.Primary)
	s.EditorAccent = paneAccent(t.Info)
	s.ResponseAccent = paneAccent(t.Success)
	return s
}

func (s Styles) MethodStyle(method string) lipgloss.Style {
	c, ok := s.methodColors[method]
	if !ok {
		c = s.ColorPrimary
	}
	return lipgloss.NewStyle().Foreground(c).Bold(true)
}

func (s Styles) MethodBadge(method string) string {
	c, ok := s.methodColors[method]
	if !ok {
		c = s.ColorPrimary
	}
	return lipgloss.NewStyle().Bold(true).
		Foreground(methodPillFg).Background(c).Padding(0, 1).Render(method)
}

func (s Styles) EnvBadge(name string) string {
	return lipgloss.NewStyle().Bold(true).
		Foreground(methodPillFg).Background(s.ColorAccent).Padding(0, 1).Render(name)
}

func (s Styles) StatusColor(code int) lipgloss.AdaptiveColor {
	switch {
	case code >= 200 && code < 300:
		return s.ColorSuccess
	case code >= 300 && code < 400:
		return s.ColorInfo
	case code >= 400:
		return s.ColorError
	default:
		return s.ColorWarn
	}
}

func (s Styles) TabBar(tabs []string, active, maxW int, accent *PaneAccent) string {
	activeStyle := s.ActiveTabStyle
	if accent != nil {
		activeStyle = accent.ActiveTab
	}
	for _, pad := range []int{2, 1} {
		strip := s.tabRender(tabs, active, pad, activeStyle)
		if lipgloss.Width(strip) <= maxW {
			return strip
		}
	}

	pad := 1
	indices := []int{active}
	for d := 1; ; d++ {
		cand := append([]int{}, indices...)
		changed := false
		if left := active - d; left >= 0 {
			cand = append([]int{left}, cand...)
			changed = true
		}
		if right := active + d; right < len(tabs) {
			cand = append(cand, right)
			changed = true
		}
		if !changed {
			break
		}
		if tabWidthIdx(tabs, cand, pad) <= maxW {
			indices = cand
		}
	}
	var out string
	for _, index := range indices {
		style := s.TabStyle
		if index == active {
			style = activeStyle
		}
		out += style.Padding(0, pad).Render(tabs[index])
	}
	return out
}

func (s Styles) tabRender(tabs []string, active, pad int, activeStyle lipgloss.Style) string {
	var out string
	for i, tab := range tabs {
		style := s.TabStyle
		if i == active {
			style = activeStyle
		}
		out += style.Padding(0, pad).Render(tab)
	}
	return out
}

func (s Styles) Rule(width int) string {
	if width <= 0 {
		return ""
	}
	return lipgloss.NewStyle().Foreground(s.ColorBorder).Render(strings.Repeat("─", width))
}

func (s Styles) SectionLine(title string, width int) string {
	if width < 1 {
		width = 1
	}
	title = TruncateRunes(title, width-4)
	dash := lipgloss.NewStyle().Foreground(s.ColorBorder)
	fill := width - 2 - lipgloss.Width(" "+title+" ")
	if fill < 1 {
		fill = 1
	}
	return dash.Render("──") + s.SectionStyle.Render(" "+title+" ") + dash.Render(strings.Repeat("─", fill))
}

func (s Styles) KeyHint(pairs ...string) string {
	var out strings.Builder
	for i := 0; i+1 < len(pairs); i += 2 {
		if i > 0 {
			out.WriteString(" · ")
		}
		out.WriteString(s.KeyStyle.Render(pairs[i]))
		out.WriteString(" ")
		out.WriteString(s.HintStyle.Render(pairs[i+1]))
	}
	return out.String()
}
