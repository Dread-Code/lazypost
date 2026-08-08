package model

import (
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"

	"lazypost/internal/app"
	"lazypost/internal/collection"
	"lazypost/internal/httpclient"
	"lazypost/internal/session"
	"lazypost/internal/ui/widgets"
)

type responseMsg struct {
	res   *httpclient.Response
	store map[string]string // post-hook store writes to merge
	req   collection.Request
}
type errMsg struct {
	err   error
	store map[string]string // store writes even when the send fails
	req   collection.Request
}

// overlay is which modal currently sits on top of the frame. At most one
// overlay is open at a time; while open, every message routes to it
// before the panes see anything.
type overlay int

const (
	noOverlay overlay = iota
	ovPalette
	ovNamer
	ovConfirm
	ovEnv
	ovHistory
	ovHelp
)

// paletteState is the shared palette widget in all its modes: the command
// palette, the theme picker (theme), and the environment manager
// (envTab/envFiltering browse its variables).
type paletteState struct {
	widget       *ui.Palette
	prev         pane // pane to restore when the overlay closes
	theme        bool // palette is a theme picker instead of commands
	envTab       int  // active environment index into envNames
	envFiltering bool // "/" typed: letters filter instead of acting
}

// namerState is the name-input modal, used for new requests/folders
// (dir/rename/old) and for key=value variable edits (envEdit/envNew).
type namerState struct {
	widget  *ui.Namer
	dir     string // folder the new request will be created in
	rename  bool   // renaming an existing request instead of creating one
	old     string // path of the request being renamed
	envEdit string // environment whose variables are being edited
	envNew  bool   // "a" (add) instead of "r" (edit): a leading "/" creates an environment
}

// confirmState is the y/n modal for destructive actions: deleting a
// request or folder (target) or a variable (env+key).
type confirmState struct {
	widget *ui.Confirm
	target *collection.Entry // entry to delete if confirmed; kept as data (not a closure)
	env    string            // environment whose variable is being deleted
	key    string            // variable to delete
}

type Model struct {
	dir      string
	sidebar  *ui.Sidebar
	urlbar   *ui.URLBar
	editor   *ui.Editor
	response *ui.Response
	focus    pane
	// prevFocus is where esc in the URL bar returns to
	prevFocus pane

	overlay overlay

	palette paletteState
	namer   namerState
	confirm confirmState

	width  int
	height int

	envs     map[string]map[string]string
	envNames []string
	envIdx   int // 0 = none

	// store holds values chained between requests by script hooks
	// ([[Design - request chaining store]]); memory-only per session.
	store map[string]string

	// history is the in-memory ring of past sends (memory-only by design,
	// [[Design - request history]]).
	history       *app.History
	historyWidget *ui.History
	historyPrev   pane // pane to restore when the history overlay closes
	helpPrev      pane // pane to restore when the keybindings panel closes

	state session.State

	notice      string
	noticeError bool
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
	m.palette.widget = ui.NewPalette(40, 10)
	m.namer.widget = ui.NewNamer()
	m.confirm.widget = ui.NewConfirm()
	m.history = app.NewHistory(historyCap)
	m.historyWidget = ui.NewHistory(40, 10)
	m.focus = pSidebar
	m.restore(st)
	return m
}

func (m Model) Init() tea.Cmd { return nil }

// Update is a thin router: typed messages first, then the open overlay,
// then global keys, then the focused pane. Actions live in actions.go,
// focus in focus.go, rendering in view.go, modal handling in their own
// files.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.layout()
		return m, nil

	case responseMsg:
		m.response.SetResponse(msg.res)
		m.store = app.MergeVars(m.store, msg.store)
		m.history.Add(app.HistoryEntry{Req: msg.req, Summary: msg.res.Summary(), At: time.Now(), Res: msg.res})
		return m, nil

	case errMsg:
		m.response.SetError(msg.err)
		m.store = app.MergeVars(m.store, msg.store)
		m.history.Add(app.HistoryEntry{Req: msg.req, Summary: msg.err.Error(), At: time.Now(), Err: msg.err})
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.response, cmd = m.response.Update(msg)
		return m, cmd
	}

	// A modal overlay is open: every message (keys, filter matches,
	// spinner) goes to it while it is up.
	switch m.overlay {
	case ovPalette:
		return m.updatePalette(msg)
	case ovNamer:
		return m.updateNamer(msg)
	case ovConfirm:
		return m.updateConfirm(msg)
	case ovEnv:
		return m.updateEnvManager(msg)
	case ovHistory:
		return m.updateHistory(msg)
	case ovHelp:
		return m.updateHelp(msg)
	}

	km, isKey := msg.(tea.KeyMsg)
	if !isKey {
		return m, nil
	}

	// Toggle the command palette.
	if key.Matches(km, keyPalette) {
		return m.openPalette()
	}

	// Request history overlay.
	if key.Matches(km, keyHistory) {
		return m.openHistory()
	}

	// Global keys, handled before any pane sees them. Driven by the
	// action registry so the palette and keybindings can't drift.
	for _, a := range globalActions {
		if a.matches(km) {
			return a.Run(&m)
		}
	}

	// Pane cycling is focus routing, not a command.
	if key.Matches(km, keyTab) {
		return m, m.cycleFocus(1)
	}
	if key.Matches(km, keyShiftTab) {
		return m, m.cycleFocus(-1)
	}

	// Remaining keys route to the focused pane only.
	switch m.focus {
	case pSidebar:
		switch {
		case key.Matches(km, keyQuit):
			return m, m.quit()
		case key.Matches(km, keyHelp):
			// "?" is printable, so it stays scoped to non-input panes
			return m.openHelp()
		case key.Matches(km, keyEnter):
			// enter on a directory collapses/expands it; on a request it
			// moves focus to the URL bar — loading already happened with
			// navigation (the sidebar auto-loads the selected request)
			if m.sidebar.ToggleCollapsed() {
				return m, m.saveState()
			}
			if m.sidebar.Selected() != nil {
				return m, m.enter(pBar)
			}
			return m, nil

		case key.Matches(km, keyAdd):
			// add a new request under the highlighted folder, or inside
			// the parent folder of the highlighted request; the namer
			// asks for its name
			if d := m.sidebar.SelectedDir(); d != nil {
				m.namer.dir = d.Path
			} else if e := m.sidebar.Selected(); e != nil {
				m.namer.dir = filepath.Dir(e.Path)
			} else {
				return m, nil
			}
			m.overlay = ovNamer
			m.namer.widget.SetLabel("")
			m.namer.widget.SetPlaceholder("e.g. list things")
			m.namer.widget.SetEnvMode(false)
			return m, m.namer.widget.Open()
		case key.Matches(km, keyDelete):
			// deleting is destructive: confirm first. Folders and requests
			// both get the modal, with the folder case expanded in Delete.
			if e := m.sidebar.Selected(); e != nil {
				return m, m.openDeleteConfirm(e)
			}
			if d := m.sidebar.SelectedDir(); d != nil {
				return m, m.openDeleteConfirm(d)
			}
			return m, nil
		case key.Matches(km, keyRename):
			if e := m.sidebar.Selected(); e != nil {
				m.namer.rename = true
				m.namer.old = e.Path
				m.namer.dir = filepath.Dir(e.Path)
				m.overlay = ovNamer
				m.namer.widget.SetLabel("")
				m.namer.widget.SetPlaceholder("e.g. list things")
				m.namer.widget.SetEnvMode(false)
				return m, m.namer.widget.OpenPrefilled(e.Req.Name)
			}
			return m, nil
		case key.Matches(km, keyN):
			m.urlbar.New()
			return m, tea.Batch(m.editor.New(), m.enter(pBar))
		}
		var cmd tea.Cmd
		// navigating to a request loads it into the URL bar and editor
		// immediately, without Enter (paths are compared because the
		// sidebar rebuilds its item pointers on every call)
		prev := ""
		if e := m.sidebar.Selected(); e != nil {
			prev = e.Path
		}
		m.sidebar, cmd = m.sidebar.Update(msg)
		if e := m.sidebar.Selected(); e != nil && e.Path != prev {
			m.urlbar.SetRequest(e.Req.Method, e.Req.URL)
			return m, tea.Batch(m.editor.SetRequest(e.Req, e.Path), m.saveState())
		}
		return m, cmd

	case pBar:
		switch {
		case key.Matches(km, keyEnter):
			return m.send()
		case key.Matches(km, keyEsc):
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
		if key.Matches(km, keyQuit) {
			return m, m.quit()
		}
		if key.Matches(km, keyHelp) {
			return m.openHelp()
		}
		var cmd tea.Cmd
		m.response, cmd = m.response.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m *Model) setNotice(s string, isError bool) {
	m.notice = s
	m.noticeError = isError
}
