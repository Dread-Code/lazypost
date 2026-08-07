package model

import (
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"

	"postgo/internal/collection"
	"postgo/internal/httpclient"
	"postgo/internal/session"
	"postgo/internal/ui"
)

type responseMsg struct{ res *httpclient.Response }
type errMsg struct{ err error }

type Model struct {
	dir      string
	sidebar  *ui.Sidebar
	urlbar   *ui.URLBar
	editor   *ui.Editor
	response *ui.Response
	focus    pane
	// prevFocus is where esc in the URL bar returns to
	prevFocus pane

	palette     *ui.Palette
	paletteOpen bool
	// palettePrev is the pane to restore when the palette closes
	palettePrev pane

	namer     *ui.Namer
	namerOpen bool
	// namerDir is the folder the new request will be created in
	namerDir string

	width  int
	height int

	envs     map[string]map[string]string
	envNames []string
	envIdx   int // 0 = none

	state session.State

	notice      string
	noticeError bool

	keyTab      key.Binding
	keyShiftTab key.Binding
	keyPalette  key.Binding
	keyEnter    key.Binding
}

func New(dir string, entries []collection.Entry, envs map[string]map[string]string, envNames []string, st session.State) Model {
	m := Model{
		dir:      dir,
		envs:     envs,
		envNames: envNames,
		state:    st,
	}
	m.sidebar = ui.NewSidebar(entries, dir, 30, 20)
	m.urlbar = ui.NewURLBar(80)
	m.editor = ui.NewEditor(60, 15)
	m.response = ui.NewResponse(60, 15)
	m.palette = ui.NewPalette(40, 10)
	m.namer = ui.NewNamer()
	m.focus = pSidebar
	m.keyTab = key.NewBinding(key.WithKeys("tab"))
	m.keyShiftTab = key.NewBinding(key.WithKeys("shift+tab"))
	// ctrl+/ is delivered as ctrl+_ (0x1F) by terminals; accept both
	m.keyPalette = key.NewBinding(key.WithKeys("ctrl+_", "ctrl+/"))
	m.keyEnter = key.NewBinding(key.WithKeys("enter"))
	m.restore(st)
	return m
}

// restore applies persisted session state: active environment, collapsed
// folders, and the last selected request.
func (m *Model) restore(st session.State) {
	if idx := indexOf(st.Env, m.envNames); idx >= 0 {
		m.envIdx = idx + 1
	}
	m.sidebar.SetCollapsed(m.dir, st.Collapsed)
	if st.ActivePath != "" && m.sidebar.SelectPath(filepath.Join(m.dir, st.ActivePath)) {
		if e := m.sidebar.Selected(); e != nil {
			m.urlbar.SetRequest(e.Req.Method, e.Req.URL)
			m.editor.SetRequest(e.Req, e.Path)
		}
	}
}

func indexOf(s string, list []string) int {
	for i, v := range list {
		if v == s {
			return i
		}
	}
	return -1
}

func (m Model) Init() tea.Cmd { return nil }

// Update is a thin router: typed messages first, then global keys, then
// the focused pane. Actions live in actions.go, focus in focus.go,
// rendering in view.go.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.layout()
		return m, nil

	case responseMsg:
		m.response.SetResponse(msg.res)
		return m, nil

	case errMsg:
		m.response.SetError(msg.err)
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.response, cmd = m.response.Update(msg)
		return m, cmd
	}

	// Palette is modal: every message (keys, filter matches, spinner)
	// goes to it while open.
	if m.paletteOpen {
		return m.updatePalette(msg)
	}

	// Namer is modal too: while asking for a name, every key goes to it.
	if m.namerOpen {
		return m.updateNamer(msg)
	}

	km, isKey := msg.(tea.KeyMsg)
	if !isKey {
		return m, nil
	}

	// Toggle the command palette.
	if key.Matches(km, m.keyPalette) {
		return m.openPalette()
	}

	// Global keys, handled before any pane sees them. Driven by the
	// action registry so the palette and keybindings can't drift.
	for _, a := range globalActions {
		if a.matches(km) {
			return a.Run(&m)
		}
	}

	// Pane cycling is focus routing, not a command.
	if key.Matches(km, m.keyTab) {
		return m, m.cycleFocus(1)
	}
	if key.Matches(km, m.keyShiftTab) {
		return m, m.cycleFocus(-1)
	}

	// Remaining keys route to the focused pane only.
	switch m.focus {
	case pSidebar:
		switch km.String() {
		case "q":
			return m, m.quit()
		case "enter":
			// enter on a directory collapses/expands it; on a request it loads
			if m.sidebar.ToggleCollapsed() {
				return m, m.saveState()
			}
			if e := m.sidebar.Selected(); e != nil {
				m.urlbar.SetRequest(e.Req.Method, e.Req.URL)
				return m, tea.Batch(m.editor.SetRequest(e.Req, e.Path), m.enter(pBar), m.saveState())
			}
			return m, nil

		case "a":
			// add a new request under the highlighted folder, or inside
			// the parent folder of the highlighted request; the namer
			// asks for its name
			if d := m.sidebar.SelectedDir(); d != nil {
				m.namerDir = d.Path
			} else if e := m.sidebar.Selected(); e != nil {
				m.namerDir = filepath.Dir(e.Path)
			} else {
				return m, nil
			}
			m.namerOpen = true
			return m, m.namer.Open()
		case "n":
			m.urlbar.New()
			return m, tea.Batch(m.editor.New(), m.enter(pBar))
		}
		var cmd tea.Cmd
		m.sidebar, cmd = m.sidebar.Update(msg)
		return m, cmd

	case pBar:
		switch {
		case key.Matches(km, m.keyEnter):
			return m.send()
		case km.Type == tea.KeyEsc:
			return m, m.enter(m.prevFocus)
		case km.Paste:
			// a pasted curl command is imported; other pastes insert
			if text := strings.TrimSpace(string(km.Runes)); strings.HasPrefix(text, "curl ") || text == "curl" {
				return m.importCurl(text)
			}
		}
		var cmd tea.Cmd
		m.urlbar, cmd = m.urlbar.Update(msg)
		return m, cmd

	case pEditor:
		var cmd tea.Cmd
		m.editor, cmd = m.editor.Update(msg)
		return m, cmd

	case pResponse:
		switch km.String() {
		case "q":
			return m, m.quit()
		}
		var cmd tea.Cmd
		m.response, cmd = m.response.Update(msg)
		return m, cmd
	}

	return m, nil
}
