package model

import (
	neturl "net/url"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"postgo/internal/collection"
	"postgo/internal/httpclient"
	"postgo/internal/ui"
)

type pane int

const (
	pSidebar pane = iota
	pBar
	pEditor
	pResponse
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
	m.keyEnter = key.NewBinding(key.WithKeys("enter"))
	return m
}

func (m Model) Init() tea.Cmd { return nil }

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
	return nil
}

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

	switch {
	case key.Matches(km, key.NewBinding(key.WithKeys("ctrl+c"))):
		return m, tea.Quit

	case key.Matches(km, key.NewBinding(key.WithKeys("ctrl+r"))):
		return m.send()

	case key.Matches(km, key.NewBinding(key.WithKeys("ctrl+s"))):
		return m.save()

	case key.Matches(km, key.NewBinding(key.WithKeys("ctrl+e"))):
		if len(m.envNames) > 0 {
			// +1 slot for "none" at index 0
			m.envIdx = (m.envIdx + 1) % (len(m.envNames) + 1)
			if name := m.activeEnvName(); name != "" {
				m.setNotice("environment: "+name, false)
			} else {
				m.setNotice("environment: none", false)
			}
		}
		return m, nil

	case key.Matches(km, m.keyCtrlL):
		m.setFocus(pBar)
		return m, m.focusCmd()

	case key.Matches(km, m.keyTab):
		m.cycleFocus(1)
		return m, m.focusCmd()

	case key.Matches(km, m.keyShiftTab):
		m.cycleFocus(-1)
		return m, m.focusCmd()
	}

	switch m.focus {
	case pSidebar:
		switch km.String() {
		case "q":
			return m, tea.Quit
		case "enter":
			if e := m.sidebar.Selected(); e != nil {
				m.urlbar.SetRequest(e.Req.Method, e.Req.URL)
				m.setFocus(pBar)
				return m, tea.Batch(m.editor.SetRequest(e.Req, e.Path), m.focusCmd())
			}
			return m, nil
		case "n":
			m.urlbar.New()
			m.setFocus(pBar)
			return m, tea.Batch(m.editor.New(), m.focusCmd())
		}
		var cmd tea.Cmd
		m.sidebar, cmd = m.sidebar.Update(msg)
		return m, cmd

	case pBar:
		switch {
		case key.Matches(km, m.keyEnter):
			return m.send()
		case km.Type == tea.KeyEsc:
			m.setFocus(m.prevFocus)
			return m, m.focusCmd()
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

func (m *Model) setNotice(s string, isError bool) {
	m.notice = s
	m.noticeError = isError
}

func (m *Model) cycleFocus(n int) {
	// +4 keeps the result positive for backward steps
	m.setFocus(pane((int(m.focus) + n + 4) % 4))
}

func (m *Model) setFocus(p pane) {
	if m.focus == p {
		return
	}
	if p == pBar {
		m.prevFocus = m.focus
	}
	m.urlbar.Blur()
	m.editor.Blur()
	m.response.Blur()
	m.focus = p
}

func (m *Model) focusCmd() tea.Cmd {
	switch m.focus {
	case pBar:
		return m.urlbar.Focus()
	case pEditor:
		return m.editor.Focus()
	case pResponse:
		m.response.Focus()
	}
	return nil
}

func (m *Model) layout() {
	if m.width < 10 || m.height < 6 {
		return
	}
	sidebarW := m.width / 4
	if sidebarW < 24 {
		sidebarW = 24
	}
	if sidebarW > 40 {
		sidebarW = 40
	}
	rightW := m.width - sidebarW - 1
	contentH := m.height - 3 // title bar + URL bar + status bar

	editorH := contentH * 55 / 100
	respH := contentH - editorH

	// -3 per pane: 2 border rows + 1 title row
	m.urlbar.Resize(m.width)
	m.sidebar.Resize(sidebarW-2, contentH-3)
	m.editor.Resize(rightW-2, editorH-3)
	m.response.Resize(rightW-2, respH-3)
}

// View assembles exactly terminal-height output; taller frames make the
// renderer drop lines (e.g. the title bar).
func (m Model) View() string {
	if m.width < 60 || m.height < 20 {
		return "terminal too small — resize to use postgo"
	}

	sidebarW := m.width / 4
	if sidebarW < 24 {
		sidebarW = 24
	}
	if sidebarW > 40 {
		sidebarW = 40
	}
	rightW := m.width - sidebarW - 1
	contentH := m.height - 3
	editorH := contentH * 55 / 100
	respH := contentH - editorH

	title := lipgloss.JoinHorizontal(lipgloss.Left,
		ui.TitleStyle.Render("postgo"),
		ui.HintStyle.Render("  "+ui.TruncateRunes(m.dir, m.width/2)),
	)
	envLabel := "env: none"
	if name := m.activeEnvName(); name != "" {
		envLabel = "env: " + name
	}
	envStyle := ui.HintStyle
	if m.activeEnvName() != "" {
		envStyle = lipgloss.NewStyle().Foreground(ui.ColorInfo)
	}
	gap := m.width - lipgloss.Width(title) - lipgloss.Width(envLabel)
	if gap < 1 {
		gap = 1
	}
	titleBar := title + strings.Repeat(" ", gap) + envStyle.Render(envLabel)

	bar := m.urlbar.View()
	// pad the bar to full width so the frame stays exactly terminal-sized
	if w := lipgloss.Width(bar); w < m.width {
		bar += strings.Repeat(" ", m.width-w)
	}

	sidebar := renderPane("Collection", m.sidebar.View(), m.focus == pSidebar, sidebarW, contentH)

	reqTitle := "Request"
	if p := m.editor.ActivePath(); p != "" {
		reqTitle += " · " + rel(m.dir, p)
	}
	editor := renderPane(reqTitle, m.editor.View(), m.focus == pEditor, rightW, editorH)

	respTitle := "Response"
	if s := m.response.StatusLine(); s != "" {
		respTitle += " · " + s
	}
	response := renderPane(respTitle, m.response.View(), m.focus == pResponse, rightW, respH)

	right := lipgloss.JoinVertical(lipgloss.Left, editor, response)
	content := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, right)

	status := m.statusBar()
	return lipgloss.JoinVertical(lipgloss.Left, titleBar, bar, content, status)
}

func renderPane(title, content string, focused bool, w, h int) string {
	style := ui.PaneStyle
	if focused {
		style = ui.ActivePaneStyle
	}
	title = ui.TruncateRunes(title, w-4)
	titleLine := lipgloss.NewStyle().Bold(true).Foreground(ui.ColorMuted).Render(title)
	inner := lipgloss.JoinVertical(lipgloss.Left, titleLine, content)
	return style.Width(w - 2).Height(h - 2).Render(inner)
}

func (m Model) statusBar() string {
	var help string
	switch m.focus {
	case pSidebar:
		help = "↑↓ navigate · enter load · n new · ctrl+e env · ctrl+l url · tab panes · ctrl+r send · q quit"
	case pBar:
		help = "ctrl+t method · enter send · esc back · tab panes · ctrl+r send"
	case pEditor:
		help = "ctrl+n/p field · alt+←→ tab · ctrl+t auth type · ctrl+s save · ctrl+r send"
	case pResponse:
		help = "←→ or b/h tabs · ↑↓ scroll · tab panes · q quit"
	}

	right := ""
	if m.notice != "" {
		if m.noticeError {
			right = ui.ErrorStyle.Render(ui.TruncateRunes(m.notice, m.width/3))
		} else {
			right = ui.NoticeStyle.Render(ui.TruncateRunes(m.notice, m.width/3))
		}
	}
	left := ui.HintStyle.Render(ui.TruncateRunes(help, m.width-lipgloss.Width(right)-1))

	gap := m.width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		gap = 1
	}
	return left + strings.Repeat(" ", gap) + right
}

func rel(dir, path string) string {
	return strings.TrimPrefix(path, dir+"/")
}

func defaultName(rawURL string) string {
	u, err := neturl.Parse(rawURL)
	if err == nil && u.Path != "" && u.Path != "/" {
		segs := strings.Split(strings.Trim(u.Path, "/"), "/")
		last := segs[len(segs)-1]
		if last != "" {
			return collection.Slug(last)
		}
	}
	return "untitled"
}
