package model

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// historyCap bounds the in-memory ring of past sends
// ([[Design - request history]]).
const historyCap = 20

// openHistory shows the request history overlay (ctrl+h): the last sends,
// newest first, as a browsing list.
func (m *Model) openHistory() (tea.Model, tea.Cmd) {
	entries := m.history.List()
	// newest first
	for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
		entries[i], entries[j] = entries[j], entries[i]
	}
	m.historyWidget.SetItems(entries)

	w := m.width - 8
	if w > 60 {
		w = 60
	}
	if w < 20 {
		w = 20
	}
	h := len(entries) + 4
	if h > 14 {
		h = 14
	}
	m.historyWidget.Resize(w, h)
	m.historyWidget.Open()
	m.historyPrev = m.focus
	m.overlay = ovHistory
	return m, nil
}

// updateHistory routes keys while the history overlay is open: enter loads
// the selected entry into the editor + URL bar and closes (the existing
// ctrl+r then resends it); esc/q close.
func (m *Model) updateHistory(msg tea.Msg) (tea.Model, tea.Cmd) {
	if km, ok := msg.(tea.KeyMsg); ok {
		switch {
		case key.Matches(km, keyEsc) || key.Matches(km, keyQuit):
			m.overlay = noOverlay
			return m, m.enter(m.historyPrev)
		case key.Matches(km, keyEnter):
			if it := m.historyWidget.Selected(); it != nil {
				m.overlay = noOverlay
				req := it.Req.Req
				m.urlbar.SetRequest(req.Method, req.URL)
				// restoring the entry also restores its response (or
				// error) so the evidence comes back with the request
				var cmds []tea.Cmd
				if it.Req.Res != nil {
					m.response.SetResponse(it.Req.Res)
				} else if it.Req.Err != nil {
					m.response.SetError(it.Req.Err)
				}
				cmds = append(cmds, m.editor.SetRequest(&req, ""), m.enter(pBar))
				return m, tea.Batch(cmds...)
			}
			return m, nil
		}
	}
	return m, m.historyWidget.Update(msg)
}
