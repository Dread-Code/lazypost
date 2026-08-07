package model

import tea "github.com/charmbracelet/bubbletea"

type pane int

const (
	pSidebar pane = iota
	pBar
	pEditor
	pResponse
)

// enter focuses pane p and returns its focus cmd. It is the only way to
// change focus, so a pane can never be left focused-but-dead (setFocus
// alone only blurs; it never focuses the incoming pane).
func (m *Model) enter(p pane) tea.Cmd {
	m.setFocus(p)
	return m.focusCmd()
}

// cycleFocus moves focus n steps (±1) around the pane ring. The URL bar
// is deliberately excluded: the only ways to reach it are ctrl+l or
// loading a request from the sidebar, and esc returns to the last focus.
func (m *Model) cycleFocus(n int) tea.Cmd {
	order := []pane{pSidebar, pEditor, pResponse}
	i := 0
	for idx, p := range order {
		if p == m.focus {
			i = idx
			break
		}
	}
	// when the bar is focused, forward goes to the editor, backward to the
	// sidebar, so tab keeps flowing through the body panes
	if m.focus == pBar && n > 0 {
		return m.enter(pEditor)
	}
	if m.focus == pBar && n < 0 {
		return m.enter(pSidebar)
	}
	i = (i + n + len(order)) % len(order)
	return m.enter(order[i])
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
