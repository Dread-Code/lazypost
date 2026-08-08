package model

import (
	tea "github.com/charmbracelet/bubbletea"

	"postgo/internal/collection"
	"postgo/internal/ui"
)

// openDeleteConfirm shows the confirm modal for deleting e (a request or
// a folder). The actual delete only runs if the user confirms.
func (m *Model) openDeleteConfirm(e *collection.Entry) tea.Cmd {
	kind := "request"
	if e.Kind == collection.Dir {
		kind = "folder"
	}
	label := "delete " + kind + " " + ui.TruncateRunes(e.Name, 30) + "?"
	m.confirm.widget.Ask(label)
	m.overlay = ovConfirm
	m.confirm.target = e
	return nil
}

// updateConfirm routes keys while the confirm modal is open: y/enter runs
// the pending delete, n/esc cancels.
func (m *Model) updateConfirm(msg tea.Msg) (tea.Model, tea.Cmd) {
	if km, ok := msg.(tea.KeyMsg); ok {
		switch km.String() {
		case "y", "enter":
			if m.confirm.target != nil {
				target := m.confirm.target
				m.confirm.target = nil
				m.overlay = noOverlay
				return m, m.doDelete(target)
			}
			if m.confirm.key != "" {
				env, key := m.confirm.env, m.confirm.key
				m.confirm.env, m.confirm.key = "", ""
				m.overlay = noOverlay
				return m, m.deleteVariable(env, key)
			}
		case "n", "esc", "q":
			m.overlay = noOverlay
			m.confirm.target = nil
			m.confirm.env, m.confirm.key = "", ""
		}
	}
	return m, nil
}

// doDelete removes the highlighted entry (file or folder) and reloads the
// tree. If it was the active request the editor is reset.
func (m *Model) doDelete(e *collection.Entry) tea.Cmd {
	if err := collection.Delete(m.dir, e.Path); err != nil {
		m.setNotice("delete failed: "+err.Error(), true)
		return nil
	}
	if e.Path == m.editor.ActivePath() {
		m.urlbar.New()
		m.editor.New()
	}
	entries, err := collection.Load(m.dir)
	if err == nil {
		m.sidebar.SetEntries(entries)
	}
	m.setNotice("deleted "+rel(m.dir, e.Path), false)
	return m.saveState()
}
