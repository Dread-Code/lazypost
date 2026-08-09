package model

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"lazypost/internal/ui/themes"
)

// openHelp shows the keybindings reference (?) as a read-only overlay.
func (m *Model) openHelp() (tea.Model, tea.Cmd) {
	m.helpPrev = m.focus
	m.overlay = ovHelp
	return m, nil
}

// updateHelp routes keys while the keybindings panel is open: esc/q close
// and restore the pane that had focus; everything else is ignored (the
// panel is read-only, [[Design - keybindings panel]]).
func (m *Model) updateHelp(msg tea.Msg) (tea.Model, tea.Cmd) {
	if km, ok := msg.(tea.KeyMsg); ok {
		switch {
		case key.Matches(km, keyEsc) || key.Matches(km, keyQuit):
			m.overlay = noOverlay
			return m, m.enter(m.helpPrev)
		}
	}
	return m, nil
}

// helpRow is one binding pair: two key/action cells side by side. The
// right cell may be empty (its slot renders blank).
type helpRow struct {
	k1, a1 string
	k2, a2 string
}

// helpSection is a titled group of binding rows.
type helpSection struct {
	title string
	rows  []helpRow
}

// helpSections is the single source of the keybindings panel's content.
// It must stay in sync with keys.go and the action registry — the drift
// tests enforce that every documented key is a live binding and appears
// here.
func helpSections() []helpSection {
	return []helpSection{
		{"Global", []helpRow{
			{"ctrl+/", "command palette", "?", "keybindings"},
			{"ctrl+h", "request history", "ctrl+r", "send request"},
			{"ctrl+e", "cycle environment", "ctrl+s", "save request"},
			{"ctrl+l", "focus URL bar", "ctrl+g", "export as curl"},
			{"tab", "switch panes", "", ""},
		}},
		{"Collection · sidebar", []helpRow{
			{"↑/↓ ctrl+n/p", "navigate + load", "enter", "url / toggle folder"},
			{"a", "add request/folder", "n", "new request"},
			{"d", "delete", "r", "rename"},
		}},
		{"URL bar", []helpRow{
			{"ctrl+t", "cycle method", "enter", "send"},
			{"esc", "back to previous pane", "paste", "import curl"},
		}},
		{"Editor", []helpRow{
			{"ctrl+n/p", "section", "alt+←→", "tabs"},
			{"ctrl+t", "auth type", "ctrl+s", "save"},
		}},
		{"Response", []helpRow{
			{"←/→ or b/h", "tabs", "↑/↓", "scroll"},
			{"q", "quit", "", ""},
		}},
	}
}

// helpContent renders the grouped keybinding reference as an aligned
// two-column table whose section headers speak the pane-legend language
// ("── Title ──"). maxW caps the content width; on a terminal too narrow
// for two columns it falls back to one column (rows instead of pairs).
// The panel hugs its content, so it never overflows horizontally.
func helpContent(maxW int) string {
	layout := helpLayoutFor(helpSections())
	if layout.width > maxW {
		layout = helpLayoutSingle(helpSections())
	}

	var b strings.Builder
	for _, s := range helpSections() {
		// section headers are dash runs, so they double as separators —
		// no blank lines needed, keeping the panel short on small screens
		fmt.Fprintln(&b, themes.SectionLine(s.title, layout.width))
		if layout.single {
			for _, r := range s.rows {
				if r.k1 == "" {
					continue
				}
				fmt.Fprintln(&b, "  "+themes.KeyStyle.Render(pad(r.k1, layout.keyW))+"  "+r.a1)
			}
			continue
		}
		for _, r := range s.rows {
			fmt.Fprintln(&b, "  "+themes.KeyStyle.Render(pad(r.k1, layout.keyW))+"  "+pad(r.a1, layout.actW)+
				"    "+themes.KeyStyle.Render(pad(r.k2, layout.keyW2))+"  "+pad(r.a2, layout.actW2))
		}
	}
	return strings.TrimSuffix(b.String(), "\n")
}

// helpColumns is the two-column layout: per-column key/action widths
// derived from the content, so alignment survives edits.
type helpLayout struct {
	single bool
	width  int
	keyW   int
	actW   int
	keyW2  int
	actW2  int
}

// helpColumnsSingle is the narrow-terminal fallback: one column of rows.
func helpLayoutSingle(sections []helpSection) helpLayout {
	keyW, actW := 0, 0
	for _, s := range sections {
		for _, r := range s.rows {
			keyW = max(keyW, lipgloss.Width(r.k1))
			actW = max(actW, lipgloss.Width(r.a1))
		}
	}
	return helpLayout{
		single: true,
		width:  2 + keyW + 2 + actW,
		keyW:   keyW,
		actW:   actW,
	}
}

// helpLayoutFor computes the two-column widths from the actual content.
func helpLayoutFor(sections []helpSection) helpLayout {
	keyW, actW, keyW2, actW2 := 0, 0, 0, 0
	for _, s := range sections {
		for _, r := range s.rows {
			keyW = max(keyW, lipgloss.Width(r.k1))
			actW = max(actW, lipgloss.Width(r.a1))
			keyW2 = max(keyW2, lipgloss.Width(r.k2))
			actW2 = max(actW2, lipgloss.Width(r.a2))
		}
	}
	col1 := 2 + keyW + 2 + actW
	// col2 excludes its own leading indent — the row already carries the
	// gap between the columns
	col2 := keyW2 + 2 + actW2
	return helpLayout{
		width: col1 + 4 + col2,
		keyW:  keyW,
		actW:  actW,
		keyW2: keyW2,
		actW2: actW2,
	}
}

// pad right-pads s to w display columns (rune-count based).
func pad(s string, w int) string {
	if n := lipgloss.Width(s); n < w {
		return s + strings.Repeat(" ", w-n)
	}
	return s
}
