package model

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/Dread-Code/lazypost/internal/app"
	"github.com/Dread-Code/lazypost/internal/collection"
)

func (m *Model) beginMutation() (uint64, []string, bool) {
	if m.mutationBusy {
		m.setNotice("another collection operation is in progress", true)
		return 0, nil, false
	}
	m.nextMutationID++
	m.activeMutationID = m.nextMutationID
	m.mutationBusy = true
	legacy := append([]string{}, m.legacyMarkers...)
	return m.activeMutationID, legacy, true
}

func runLegacyMigration(dir string, legacy []string) (bool, error) {
	if len(legacy) == 0 {
		return false, nil
	}
	if err := app.MigrateCollection(dir, legacy); err != nil {
		return false, err
	}
	return true, nil
}

func (m *Model) writeNotice(message string, isError bool) {
	if m.legacyMigrated {
		message += " (legacy .lazypost migrated; name/root discarded)"
		m.legacyMigrated = false
	}
	m.setNotice(message, isError)
}

// save persists the composed request to disk, then refreshes the sidebar
// so the new/changed file appears in the tree.
func (m *Model) save() (tea.Model, tea.Cmd) {
	m.editor.FormatBody()
	req, err := m.editor.RequestWithError()
	if err != nil {
		m.writeNotice("save failed: "+err.Error(), true)
		return m, nil
	}
	req.Method = m.urlbar.Method()
	req.URL = m.urlbar.URL()
	if req.URL == "" {
		m.setNotice("nothing to save: URL is empty", true)
		return m, nil
	}
	if req.Name == "" {
		req.Name = collection.DefaultName(req.URL)
	}
	id, legacy, ok := m.beginMutation()
	if !ok {
		return m, nil
	}
	dir := m.dir
	activePath := m.editor.ActivePath()
	request := *req
	return m, func() tea.Msg {
		migrated, err := runLegacyMigration(dir, legacy)
		if err != nil {
			return collectionMutationMsg{id: id, op: mutationSave, err: err}
		}
		path, entries, err := app.SaveRequest(dir, activePath, &request)
		return collectionMutationMsg{id: id, op: mutationSave, entries: entries, path: path, err: err, migrated: migrated}
	}
}

// renameRequest moves the request at oldPath under its new slug path and
// refreshes the sidebar; the editor follows if it was the active request.
func (m *Model) renameRequest(oldPath, name string) (tea.Model, tea.Cmd) {
	id, legacy, ok := m.beginMutation()
	if !ok {
		return m, nil
	}
	dir := m.dir
	return m, func() tea.Msg {
		migrated, err := runLegacyMigration(dir, legacy)
		if err != nil {
			return collectionMutationMsg{id: id, op: mutationRename, oldPath: oldPath, err: err}
		}
		req, newPath, entries, err := app.RenameRequest(dir, oldPath, name)
		return collectionMutationMsg{id: id, op: mutationRename, req: req, path: newPath, oldPath: oldPath, entries: entries, err: err, migrated: migrated}
	}
}

// createFolderIn makes a new directory under dir and refreshes the tree.
func (m *Model) createFolderIn(dir, name string) (tea.Model, tea.Cmd) {
	id, legacy, ok := m.beginMutation()
	if !ok {
		return m, nil
	}
	root := m.dir
	return m, func() tea.Msg {
		migrated, err := runLegacyMigration(root, legacy)
		if err != nil {
			return collectionMutationMsg{id: id, op: mutationCreateFolder, err: err}
		}
		path, entries, err := app.CreateFolder(root, dir, name)
		return collectionMutationMsg{id: id, op: mutationCreateFolder, path: path, entries: entries, err: err, migrated: migrated}
	}
}

// createRequestIn writes a blank named request under dir, refreshes the
// tree, and loads it into the editor so the user can fill it in.
func (m *Model) createRequestIn(dir, name string) (tea.Model, tea.Cmd) {
	id, legacy, ok := m.beginMutation()
	if !ok {
		return m, nil
	}
	root := m.dir
	return m, func() tea.Msg {
		migrated, err := runLegacyMigration(root, legacy)
		if err != nil {
			return collectionMutationMsg{id: id, op: mutationCreateReq, err: err}
		}
		req, path, entries, err := app.CreateRequest(root, dir, name)
		return collectionMutationMsg{id: id, op: mutationCreateReq, req: req, path: path, entries: entries, err: err, migrated: migrated}
	}
}

func (m *Model) applyCollectionMutation(msg collectionMutationMsg) (tea.Model, tea.Cmd) {
	if msg.id != m.activeMutationID {
		return m, nil
	}
	m.activeMutationID = 0
	m.mutationBusy = false
	if msg.migrated {
		m.legacyMarkers = nil
		m.legacyMigrated = true
	}
	if msg.err != nil {
		label := "collection operation"
		switch msg.op {
		case mutationSave:
			label = "save"
		case mutationRename:
			label = "rename"
		case mutationCreateFolder, mutationCreateReq:
			label = "create"
		case mutationDelete:
			label = "delete"
		}
		m.writeNotice(label+" failed: "+msg.err.Error(), true)
		return m, nil
	}
	switch msg.op {
	case mutationSave:
		m.editor.SetActivePath(msg.path)
		m.sidebar.SetEntries(msg.entries)
		m.writeNotice("saved "+rel(m.dir, msg.path), false)
	case mutationRename:
		m.sidebar.SetEntries(msg.entries)
		if msg.oldPath == m.editor.ActivePath() {
			m.editor.SetActivePath(msg.path)
			m.urlbar.SetRequest(msg.req.Method, msg.req.URL)
			m.editor.SetRequest(msg.req, msg.path)
		}
		m.writeNotice("renamed "+rel(m.dir, msg.oldPath)+" → "+rel(m.dir, msg.path), false)
		return m, m.saveState()
	case mutationCreateFolder:
		m.sidebar.SetEntries(msg.entries)
		m.writeNotice("created "+rel(m.dir, msg.path), false)
	case mutationCreateReq:
		m.sidebar.SetEntries(msg.entries)
		m.urlbar.SetRequest(msg.req.Method, msg.req.URL)
		m.writeNotice("created "+rel(m.dir, msg.path), false)
		return m, tea.Batch(m.editor.SetRequest(msg.req, msg.path), m.enter(pBar))
	case mutationDelete:
		if pathContains(msg.path, m.editor.ActivePath()) {
			m.urlbar.New()
			m.editor.New()
		}
		m.sidebar.SetEntries(msg.entries)
		m.writeNotice("deleted "+rel(m.dir, msg.path), false)
		return m, m.saveState()
	}
	return m, nil
}
