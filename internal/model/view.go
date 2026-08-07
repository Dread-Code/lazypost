package model

import (
	neturl "net/url"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"postgo/internal/collection"
	"postgo/internal/ui"
)

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
		help = "↑↓ navigate · enter load · n new · ctrl+e env · ctrl+l url · ctrl+g curl · tab panes · ctrl+r send · q quit"
	case pBar:
		help = "ctrl+t method · enter send · esc back · paste curl to import · ctrl+g export curl · ctrl+r send"
	case pEditor:
		help = "ctrl+n/p field · alt+←→ tab · ctrl+t auth type · ctrl+g curl · ctrl+s save · ctrl+r send"
	case pResponse:
		help = "←→ or b/h tabs · ↑↓ scroll · ctrl+g curl · tab panes · q quit"
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
