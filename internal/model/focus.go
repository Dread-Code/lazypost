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

// cycleFocus moves focus n steps (±1) around the pane ring.
func (m *Model) cycleFocus(n int) tea.Cmd {
	// +4 keeps the result positive for backward steps
	return m.enter(pane((int(m.focus) + n + 4) % 4))
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
