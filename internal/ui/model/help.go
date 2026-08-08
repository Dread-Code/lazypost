package model

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	ui "lazypost/internal/ui/widgets"
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

// helpContent renders the static grouped keybinding reference: two-column
// rows per pane, aligned like the README table. Section headers are
// accent-colored, the key column is cyan so "what do I press" jumps out,
// and the actions stay in the default foreground.
func helpContent() string {
	header := func(s string) string {
		return ui.TitleStyle.Render(s)
	}
	key := func(s string) string {
		return lipgloss.NewStyle().Bold(true).Foreground(ui.ColorInfo).Render(s)
	}
	row := func(k1, a1, k2, a2 string) string {
		// both the key and the action columns are padded to fixed widths,
		// so the second key column starts at the same offset on every row
		return "  " + key(pad(k1, 18)) + pad(a1, 26) + "    " + key(pad(k2, 18)) + a2
	}
	var b strings.Builder
	fmt.Fprintln(&b, header("Global"))
	fmt.Fprintln(&b, row("ctrl+/", "command palette", "?", "keybindings panel"))
	fmt.Fprintln(&b, row("ctrl+h", "request history", "ctrl+r", "send request"))
	fmt.Fprintln(&b, row("ctrl+e", "cycle environment", "ctrl+s", "save request"))
	fmt.Fprintln(&b, row("ctrl+l", "jump to URL bar", "ctrl+g", "export as curl"))
	fmt.Fprintln(&b, row("tab", "switch panes", "", ""))
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, header("Collection · sidebar"))
	fmt.Fprintln(&b, row("↑/↓ ctrl+n/p", "navigate + load", "enter", "url bar / toggle folder"))
	fmt.Fprintln(&b, row("a", "add request/folder", "n", "new request"))
	fmt.Fprintln(&b, row("d", "delete", "r", "rename"))
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, header("URL bar"))
	fmt.Fprintln(&b, row("ctrl+t", "cycle method", "enter", "send"))
	fmt.Fprintln(&b, row("esc", "back to previous pane", "paste", "import curl"))
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, header("Editor"))
	fmt.Fprintln(&b, row("ctrl+n/p", "section", "alt+←→", "tabs"))
	fmt.Fprintln(&b, row("ctrl+t", "auth type", "ctrl+s", "save"))
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, header("Response"))
	fmt.Fprintln(&b, row("←/→ or b/h", "tabs", "↑/↓", "scroll"))
	fmt.Fprintln(&b, row("q", "quit", "", ""))
	return strings.TrimSuffix(b.String(), "\n")
}

// pad right-pads s to w display columns (rune-count based).
func pad(s string, w int) string {
	if n := utf8.RuneCountInString(s); n < w {
		return s + strings.Repeat(" ", w-n)
	}
	return s
}
