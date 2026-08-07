package model

import (
	"encoding/base64"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"postgo/internal/clipboard"
	"postgo/internal/collection"
	"postgo/internal/curl"
	"postgo/internal/httpclient"
	"postgo/internal/render"
	"postgo/internal/ui"
)

// Action is one selectable command: the palette lists all of them, and
// globalActions are also bound to their shortcut keys. It is the single
// source of truth for global keybindings ([[Design - command palette]]).
type Action struct {
	Title    string
	Shortcut string
	Keys     []string
	Run      func(m *Model) (tea.Model, tea.Cmd)
}

// globalActions are the key-bound commands handled by the root model
// before any pane sees a key.
var globalActions = []Action{
	{Title: "Send request", Shortcut: "ctrl+r", Keys: []string{"ctrl+r"}, Run: func(m *Model) (tea.Model, tea.Cmd) { return m.send() }},
	{Title: "Save request", Shortcut: "ctrl+s", Keys: []string{"ctrl+s"}, Run: func(m *Model) (tea.Model, tea.Cmd) { return m.save() }},
	{Title: "Cycle environment", Shortcut: "ctrl+e", Keys: []string{"ctrl+e"}, Run: func(m *Model) (tea.Model, tea.Cmd) { m.cycleEnv(); return m, nil }},
	{Title: "Focus URL bar", Shortcut: "ctrl+l", Keys: []string{"ctrl+l"}, Run: func(m *Model) (tea.Model, tea.Cmd) { return m, m.enter(pBar) }},
	{Title: "Copy as curl", Shortcut: "ctrl+g", Keys: []string{"ctrl+g"}, Run: func(m *Model) (tea.Model, tea.Cmd) { return m.exportCurl() }},
	{Title: "Quit", Shortcut: "ctrl+c", Keys: []string{"ctrl+c"}, Run: func(m *Model) (tea.Model, tea.Cmd) { return m, tea.Quit }},
}

// paletteActions returns every command the palette offers: the global
// actions plus navigation commands that are only reachable from it.
func (m *Model) paletteActions() []Action {
	return append(globalActions,
		Action{Title: "New request", Run: func(m *Model) (tea.Model, tea.Cmd) {
			m.urlbar.New()
			return m, tea.Batch(m.editor.New(), m.enter(pBar))
		}},
		Action{Title: "Focus editor", Run: func(m *Model) (tea.Model, tea.Cmd) { return m, m.enter(pEditor) }},
		Action{Title: "Focus response", Run: func(m *Model) (tea.Model, tea.Cmd) { return m, m.enter(pResponse) }},
	)
}

// matches reports whether km hits any of the action's shortcut keys.
func (a Action) matches(km tea.KeyMsg) bool {
	if len(a.Keys) == 0 {
		return false
	}
	b := key.NewBinding(key.WithKeys(a.Keys...))
	return key.Matches(km, b)
}

// openPalette shows the command palette over the current frame.
func (m *Model) openPalette() (tea.Model, tea.Cmd) {
	actions := m.paletteActions()
	items := make([]ui.PaletteItem, len(actions))
	for i, a := range actions {
		items[i] = ui.PaletteItem{Title: a.Title, Shortcut: a.Shortcut}
	}
	m.palette.SetItems(items)
	m.palette.Resize(m.paletteWidth(items), m.minPaletteHeight(len(items)))
	m.palette.Open()
	m.palettePrev = m.focus
	m.paletteOpen = true
	return m, nil
}

// updatePalette routes a key while the palette is open: enter runs the
// selected action, esc/q close it. Non-key messages (e.g. the list's
// FilterMatchesMsg) are passed through so async filtering works.
func (m *Model) updatePalette(msg tea.Msg) (tea.Model, tea.Cmd) {
	if km, ok := msg.(tea.KeyMsg); ok {
		switch {
		case km.String() == "esc" || km.String() == "q":
			m.paletteOpen = false
			return m, m.enter(m.palettePrev)

		// bubbles routes every key to the filter input in Filtering state
		// and disables its nav bindings, so move the cursor ourselves. j/k
		// stay free for the filter query.
		case km.String() == "up" || km.String() == "ctrl+p":
			m.palette.CursorUp()
			return m, nil
		case km.String() == "down" || km.String() == "ctrl+n":
			m.palette.CursorDown()
			return m, nil

		case key.Matches(km, m.keyEnter):
			actions := m.paletteActions()
			if it := m.palette.Selected(); it != nil {
				for _, a := range actions {
					if a.Title == it.Title {
						m.paletteOpen = false
						return a.Run(m)
					}
				}
			}
			return m, nil
		}
	}
	cmd, _ := m.palette.Update(msg)
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

// send composes the request (URL/method from the bar, the rest from the
// editor), then runs the HTTP call off the render loop in a closure;
// the result comes back as responseMsg or errMsg.
func (m *Model) send() (tea.Model, tea.Cmd) {
	req := m.editor.Request()
	req.Method = m.urlbar.Method()
	req.URL = m.urlbar.URL()
	if req.URL == "" {
		m.setNotice("URL is required", true)
		return m, nil
	}
	m.setNotice("", false)
	vars := m.activeVars()
	cmd := func() tea.Msg {
		res, err := httpclient.Exec(*req, vars)
		if err != nil {
			return errMsg{err}
		}
		return responseMsg{res}
	}
	return m, tea.Batch(m.response.StartLoading(), cmd)
}

// save persists the composed request to disk, then reloads the sidebar
// so the new/changed file appears in the tree.
func (m *Model) save() (tea.Model, tea.Cmd) {
	req := m.editor.Request()
	req.Method = m.urlbar.Method()
	req.URL = m.urlbar.URL()
	if req.URL == "" {
		m.setNotice("nothing to save: URL is empty", true)
		return m, nil
	}
	if req.Name == "" {
		req.Name = defaultName(req.URL)
	}
	path, err := collection.Save(m.dir, m.editor.ActivePath(), req)
	if err != nil {
		m.setNotice("save failed: "+err.Error(), true)
		return m, nil
	}
	m.editor.SetActivePath(path)
	entries, err := collection.Load(m.dir)
	if err == nil {
		m.sidebar.SetEntries(entries)
	}
	m.setNotice("saved "+rel(m.dir, path), false)
	return m, nil
}

// cycleEnv advances the active environment: none → env1 → env2 → … → none.
func (m *Model) cycleEnv() {
	if len(m.envNames) == 0 {
		return
	}
	// +1 slot for "none" at index 0
	m.envIdx = (m.envIdx + 1) % (len(m.envNames) + 1)
	if name := m.activeEnvName(); name != "" {
		m.setNotice("environment: "+name, false)
	} else {
		m.setNotice("environment: none", false)
	}
}

// importCurl parses a pasted curl command into the bar and editor,
// replacing whatever was there ([[Design - curl import export]]).
func (m *Model) importCurl(text string) (tea.Model, tea.Cmd) {
	req, err := curl.Parse(text)
	if err != nil {
		m.setNotice("curl import failed: "+err.Error(), true)
		return m, nil
	}
	m.urlbar.SetRequest(req.Method, req.URL)
	m.editor.SetRequest(req, "")
	m.setNotice("imported curl request", false)
	return m, nil
}

// exportCurl writes the current request as a curl one-liner to the
// clipboard, interpolated with the active environment. Uses the platform
// tool (pbcopy etc.); falls back to OSC52, which Terminal.app ignores but
// iTerm2/Ghostty/kitty/wezterm honor.
func (m *Model) exportCurl() (tea.Model, tea.Cmd) {
	req := m.editor.Request()
	req.Method = m.urlbar.Method()
	req.URL = m.urlbar.URL()
	if req.URL == "" {
		m.setNotice("nothing to export: URL is empty", true)
		return m, nil
	}
	line := curlExportLine(*req, m.activeVars())
	m.setNotice("curl copied to clipboard", false)
	if err := clipboard.Write(line); err != nil {
		seq := "\x1b]52;c;" + base64.StdEncoding.EncodeToString([]byte(line)) + "\a"
		return m, tea.Printf("%s", seq)
	}
	return m, nil
}

// curlExportLine renders req as a curl command with {{vars}} interpolated
// from vars (unknown placeholders pass through, per ADR-0006).
func curlExportLine(req collection.Request, vars map[string]string) string {
	return curl.Format(render.Request(req, vars))
}

func (m *Model) setNotice(s string, isError bool) {
	m.notice = s
	m.noticeError = isError
}

func (m *Model) activeEnvName() string {
	if m.envIdx == 0 || len(m.envNames) == 0 {
		return ""
	}
	return m.envNames[m.envIdx-1]
}

func (m *Model) activeVars() map[string]string {
	if name := m.activeEnvName(); name != "" {
		return m.envs[name]
	}
	// nil vars makes interpolation a pass-through
	return nil
}
