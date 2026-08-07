package model

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"

	"postgo/internal/collection"
	"postgo/internal/httpclient"
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

	width  int
	height int

	envs     map[string]map[string]string
	envNames []string
	envIdx   int // 0 = none

	notice      string
	noticeError bool

	keyTab      key.Binding
	keyShiftTab key.Binding
	keyCtrlL    key.Binding
	keyCtrlG    key.Binding
	keyEnter    key.Binding
}

func New(dir string, entries []collection.Entry, envs map[string]map[string]string, envNames []string) Model {
	m := Model{
		dir:      dir,
		envs:     envs,
		envNames: envNames,
	}
	m.sidebar = ui.NewSidebar(entries, 30, 20)
	m.urlbar = ui.NewURLBar(80)
	m.editor = ui.NewEditor(60, 15)
	m.response = ui.NewResponse(60, 15)
	m.focus = pSidebar
	m.keyTab = key.NewBinding(key.WithKeys("tab"))
	m.keyShiftTab = key.NewBinding(key.WithKeys("shift+tab"))
	m.keyCtrlL = key.NewBinding(key.WithKeys("ctrl+l"))
	m.keyCtrlG = key.NewBinding(key.WithKeys("ctrl+g"))
	m.keyEnter = key.NewBinding(key.WithKeys("enter"))
	return m
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

	km, isKey := msg.(tea.KeyMsg)
	if !isKey {
		return m, nil
	}

	// Global keys, handled before any pane sees them.
	switch {
	case key.Matches(km, key.NewBinding(key.WithKeys("ctrl+c"))):
		return m, tea.Quit

	case key.Matches(km, key.NewBinding(key.WithKeys("ctrl+r"))):
		return m.send()

	case key.Matches(km, key.NewBinding(key.WithKeys("ctrl+s"))):
		return m.save()

	case key.Matches(km, key.NewBinding(key.WithKeys("ctrl+e"))):
		m.cycleEnv()
		return m, nil

	case key.Matches(km, m.keyCtrlG):
		return m.exportCurl()

	case key.Matches(km, m.keyCtrlL):
		return m, m.enter(pBar)

	case key.Matches(km, m.keyTab):
		return m, m.cycleFocus(1)

	case key.Matches(km, m.keyShiftTab):
		return m, m.cycleFocus(-1)
	}

	// Remaining keys route to the focused pane only.
	switch m.focus {
	case pSidebar:
		switch km.String() {
		case "q":
			return m, tea.Quit
		case "enter":
			if e := m.sidebar.Selected(); e != nil {
				m.urlbar.SetRequest(e.Req.Method, e.Req.URL)
				return m, tea.Batch(m.editor.SetRequest(e.Req, e.Path), m.enter(pBar))
			}
			return m, nil
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
			return m, tea.Quit
		}
		var cmd tea.Cmd
		m.response, cmd = m.response.Update(msg)
		return m, cmd
	}

	return m, nil
}
