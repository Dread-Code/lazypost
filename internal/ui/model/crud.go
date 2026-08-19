package model

import (
	tea "github.com/charmbracelet/bubbletea"

	"lazypost/internal/app"
	"lazypost/internal/collection"
)

// prepareWrite migrates legacy marker paths before the first collection
// mutation. New config markers intentionally do not retain name/root.
func (m *Model) prepareWrite() bool {
	if len(m.legacyMarkers) == 0 {
		return true
	}
	if err := app.MigrateCollection(m.dir, m.legacyMarkers); err != nil {
		m.setNotice("migrate collection failed: "+err.Error(), true)
		return false
	}
	m.legacyMarkers = nil
	m.legacyMigrated = true
	return true
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
	req := m.editor.Request()
	req.Method = m.urlbar.Method()
	req.URL = m.urlbar.URL()
	if req.URL == "" {
		m.setNotice("nothing to save: URL is empty", true)
		return m, nil
	}
	if req.Name == "" {
		req.Name = collection.DefaultName(req.URL)
	}
	if !m.prepareWrite() {
		return m, nil
	}
	path, entries, err := app.SaveRequest(m.dir, m.editor.ActivePath(), req)
	if err != nil {
		m.writeNotice("save failed: "+err.Error(), true)
		return m, nil
	}
	m.editor.SetActivePath(path)
	m.sidebar.SetEntries(entries)
	m.writeNotice("saved "+rel(m.dir, path), false)
	return m, nil
}

// renameRequest moves the request at oldPath under its new slug path and
// refreshes the sidebar; the editor follows if it was the active request.
func (m *Model) renameRequest(oldPath, name string) (tea.Model, tea.Cmd) {
	if !m.prepareWrite() {
		return m, nil
	}
	req, newPath, entries, err := app.RenameRequest(m.dir, oldPath, name)
	if err != nil {
		m.writeNotice("rename failed: "+err.Error(), true)
		return m, nil
	}
	m.sidebar.SetEntries(entries)
	if oldPath == m.editor.ActivePath() {
		m.editor.SetActivePath(newPath)
		m.urlbar.SetRequest(req.Method, req.URL)
		m.editor.SetRequest(req, newPath)
	}
	m.writeNotice("renamed "+rel(m.dir, oldPath)+" → "+rel(m.dir, newPath), false)
	return m, m.saveState()
}

// createFolderIn makes a new directory under dir and refreshes the tree.
func (m *Model) createFolderIn(dir, name string) (tea.Model, tea.Cmd) {
	if !m.prepareWrite() {
		return m, nil
	}
	path, entries, err := app.CreateFolder(m.dir, dir, name)
	if err != nil {
		m.writeNotice("create failed: "+err.Error(), true)
		return m, nil
	}
	m.sidebar.SetEntries(entries)
	m.writeNotice("created "+rel(m.dir, path), false)
	return m, nil
}

// createRequestIn writes a blank named request under dir, refreshes the
// tree, and loads it into the editor so the user can fill it in.
func (m *Model) createRequestIn(dir, name string) (tea.Model, tea.Cmd) {
	if !m.prepareWrite() {
		return m, nil
	}
	req, path, entries, err := app.CreateRequest(m.dir, dir, name)
	if err != nil {
		m.writeNotice("create failed: "+err.Error(), true)
		return m, nil
	}
	m.sidebar.SetEntries(entries)
	m.urlbar.SetRequest(req.Method, req.URL)
	m.writeNotice("created "+rel(m.dir, path), false)
	return m, tea.Batch(m.editor.SetRequest(req, path), m.enter(pBar))
}
