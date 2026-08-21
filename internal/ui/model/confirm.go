package model

import (
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"lazypost/internal/app"
	"lazypost/internal/collection"

	"lazypost/internal/ui/themes"
)

// openDeleteConfirm shows the confirm modal for deleting e (a request or
// a folder). The actual delete only runs if the user confirms.
func (m *Model) openDeleteConfirm(e *collection.Entry) tea.Cmd {
	kind := "request"
	if e.Kind == collection.Dir {
		kind = "folder"
	}
	label := "delete " + kind + " " + themes.TruncateRunes(e.Name, 30) + "?"
	m.confirm.widget.Ask(label)
	m.overlay = ovConfirm
	m.confirm.target = e
	return nil
}

// updateConfirm routes keys while the confirm modal is open: y/enter runs
// the pending delete, n/esc cancels.
func (m *Model) updateConfirm(msg tea.Msg) (tea.Model, tea.Cmd) {
	if km, ok := msg.(tea.KeyMsg); ok {
		switch {
		case key.Matches(km, keyYes) || key.Matches(km, keyEnter):
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
		case key.Matches(km, keyN) || key.Matches(km, keyEsc) || key.Matches(km, keyQuit):
			m.overlay = noOverlay
			m.confirm.target = nil
			m.confirm.env, m.confirm.key = "", ""
		}
	}
	return m, nil
}

// doDelete removes the highlighted entry (file or folder) and refreshes
// the tree. If it was the active request the editor is reset.
func (m *Model) doDelete(e *collection.Entry) tea.Cmd {
	id, legacy, ok := m.beginMutation()
	if !ok {
		return nil
	}
	root := m.dir
	target := *e
	return func() tea.Msg {
		migrated, err := runLegacyMigration(root, legacy)
		if err != nil {
			return collectionMutationMsg{id: id, op: mutationDelete, path: target.Path, err: err}
		}
		entries, err := app.DeleteEntry(root, &target)
		return collectionMutationMsg{id: id, op: mutationDelete, path: target.Path, entries: entries, err: err, migrated: migrated}
	}
}

func pathContains(parent, child string) bool {
	if parent == "" || child == "" {
		return false
	}
	parent = filepath.Clean(parent)
	child = filepath.Clean(child)
	rel, err := filepath.Rel(parent, child)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}
