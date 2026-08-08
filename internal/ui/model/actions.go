package model

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"postgo/internal/ui"
)

// Action is one selectable command: the palette lists all of them, and
// globalActions are also bound to their shortcut keys. It is the single
// source of truth for global keybindings ([[Design - command palette]]).
type Action struct {
	Title    string
	Shortcut string
	binding  key.Binding
	Run      func(m *Model) (tea.Model, tea.Cmd)
}

// newAction builds an action; keys are the shortcut keys ("" if none).
func newAction(title, shortcut string, keys []string, run func(m *Model) (tea.Model, tea.Cmd)) Action {
	return Action{
		Title:    title,
		Shortcut: shortcut,
		binding:  key.NewBinding(key.WithKeys(keys...)),
		Run:      run,
	}
}

// matches reports whether km hits the action's shortcut key.
func (a Action) matches(km tea.KeyMsg) bool {
	return key.Matches(km, a.binding)
}

// globalActions are the key-bound commands handled by the root model
// before any pane sees a key.
var globalActions = []Action{
	newAction("Send request", "ctrl+r", []string{"ctrl+r"}, func(m *Model) (tea.Model, tea.Cmd) { return m.send() }),
	newAction("Save request", "ctrl+s", []string{"ctrl+s"}, func(m *Model) (tea.Model, tea.Cmd) { return m.save() }),
	newAction("Cycle environment", "ctrl+e", []string{"ctrl+e"}, func(m *Model) (tea.Model, tea.Cmd) { m.cycleEnv(); return m, m.saveState() }),
	newAction("Focus URL bar", "ctrl+l", []string{"ctrl+l"}, func(m *Model) (tea.Model, tea.Cmd) { return m, m.enter(pBar) }),
	newAction("Copy as curl", "ctrl+g", []string{"ctrl+g"}, func(m *Model) (tea.Model, tea.Cmd) { return m.exportCurl() }),
	newAction("Quit", "ctrl+c", []string{"ctrl+c"}, func(m *Model) (tea.Model, tea.Cmd) { return m, m.quit() }),
}

// paletteActions returns every command the palette offers: the global
// actions plus navigation commands that are only reachable from it.
func (m *Model) paletteActions() []Action {
	return append(globalActions,
		newAction("New request", "", nil, func(m *Model) (tea.Model, tea.Cmd) {
			m.urlbar.New()
			return m, tea.Batch(m.editor.New(), m.enter(pBar))
		}),
		newAction("Focus editor", "", nil, func(m *Model) (tea.Model, tea.Cmd) { return m, m.enter(pEditor) }),
		newAction("Focus response", "", nil, func(m *Model) (tea.Model, tea.Cmd) { return m, m.enter(pResponse) }),
		newAction("Clear chain store", "", nil, func(m *Model) (tea.Model, tea.Cmd) {
			m.store = map[string]string{}
			m.setNotice("chain store cleared", false)
			return m, nil
		}),
		newAction("Switch theme", "", nil, func(m *Model) (tea.Model, tea.Cmd) {
			return m.openThemePicker()
		}),
		newAction("Environments", "", nil, func(m *Model) (tea.Model, tea.Cmd) {
			return m.openEnvManager()
		}),
	)
}

// openThemePicker reopens the palette as a theme list; enter applies the
// selected theme and persists it in session state.
func (m *Model) openThemePicker() (tea.Model, tea.Cmd) {
	names := ui.ThemeNames()
	items := make([]ui.PaletteItem, len(names))
	for i, n := range names {
		items[i] = ui.PaletteItem{Title: n}
	}
	m.palette.widget.SetItems(items)
	w := m.paletteWidth(items)
	if w < 28 {
		w = 28
	}
	m.palette.widget.Resize(w, m.minPaletteHeight(len(items)))
	m.palette.widget.Open()
	m.palette.prev = m.focus
	m.palette.theme = true
	m.overlay = ovPalette
	return m, nil
}

// applySelectedTheme applies the theme highlighted in the picker and
// persists it in session state.
func (m *Model) applySelectedTheme() (tea.Model, tea.Cmd) {
	it := m.palette.widget.Selected()
	if it == nil {
		return m, nil
	}
	m.overlay = noOverlay
	m.palette.theme = false
	name := it.Title
	ui.ThemeByName(name).Apply()
	m.state.Theme = name
	m.setNotice("theme: "+name, false)
	return m, m.saveState()
}

// openPalette shows the command palette over the current frame.
func (m *Model) openPalette() (tea.Model, tea.Cmd) {
	actions := m.paletteActions()
	items := make([]ui.PaletteItem, len(actions))
	for i, a := range actions {
		items[i] = ui.PaletteItem{Title: a.Title, Shortcut: a.Shortcut}
	}
	m.palette.widget.SetItems(items)
	m.palette.widget.Resize(m.paletteWidth(items), m.minPaletteHeight(len(items)))
	m.palette.widget.Open()
	m.palette.prev = m.focus
	m.palette.theme = false
	m.overlay = ovPalette
	return m, nil
}

// updatePalette routes a key while the palette is open: enter runs the
// selected action, esc/q close it. Non-key messages (e.g. the list's
// FilterMatchesMsg) are passed through so async filtering works.
func (m *Model) updatePalette(msg tea.Msg) (tea.Model, tea.Cmd) {
	if km, ok := msg.(tea.KeyMsg); ok {
		switch {
		case key.Matches(km, keyEsc) || key.Matches(km, keyQuit):
			m.overlay = noOverlay
			m.palette.theme = false
			return m, m.enter(m.palette.prev)

		// bubbles routes every key to the filter input in Filtering state
		// and disables its nav bindings, so move the cursor ourselves. j/k
		// stay free for the filter query.
		case key.Matches(km, keyUp) || key.Matches(km, keyCtrlP):
			m.palette.widget.CursorUp()
			return m, nil
		case key.Matches(km, keyDown) || key.Matches(km, keyCtrlN):
			m.palette.widget.CursorDown()
			return m, nil

		case key.Matches(km, keyEnter):
			if m.palette.theme {
				return m.applySelectedTheme()
			}
			actions := m.paletteActions()
			if it := m.palette.widget.Selected(); it != nil {
				for _, a := range actions {
					if a.Title == it.Title {
						m.overlay = noOverlay
						return a.Run(m)
					}
				}
			}
			return m, nil
		}
	}
	cmd, _ := m.palette.widget.Update(msg)
	return m, cmd
}

// paletteWidth sizes the palette to its widest item (+ shortcut), capped so
// the dialog stays tight and centered instead of sprawling across the pane
// borders ([[Design - command palette]]).
func (m *Model) paletteWidth(items []ui.PaletteItem) int {
	w := m.width - 8
	longest := 0
	for _, it := range items {
		l := lipgloss.Width(it.Title) + lipgloss.Width(it.Shortcut)
		if l > longest {
			longest = l
		}
	}
	if longest+8 < w {
		w = longest + 8
	}
	if w < 20 {
		w = 20
	}
	return w
}

// minPaletteHeight sizes the palette dialog to its items (+ the filter
// row + border), capped so it never swallows the terminal.
func (m *Model) minPaletteHeight(n int) int {
	h := n + 2 // +2 for the filter row + border
	if h < 4 {
		return 4
	}
	if h > 10 {
		return 10
	}
	return h
}
