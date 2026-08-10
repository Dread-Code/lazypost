package model

import (
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"

	"lazypost/internal/app"
	"lazypost/internal/collection"
)

// openCollectionMarker opens the namer to name a markerless collection
// on first run ([[Design - collection marker file]]).
func (m *Model) openCollectionMarker() (tea.Model, tea.Cmd) {
	m.overlay = ovNamer
	m.namer.marker = true
	m.namer.widget.SetLabel("Collection name")
	return m, m.namer.widget.OpenPrefilled(filepath.Base(m.dir))
}

// createCollection writes the .lazypost marker for the root and adopts
// the name; the void dir stays open with just that file on disk.
func (m *Model) createCollection(name string) (tea.Model, tea.Cmd) {
	_, err := app.CreateCollection(m.dir, name)
	if err != nil {
		m.setNotice("create collection failed: "+err.Error(), true)
		return m, nil
	}
	m.needsMarker = false
	m.collectionName = name
	m.overlay = noOverlay
	m.setNotice("collection '"+name+"' created", false)
	return m, nil
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
	path, entries, err := app.SaveRequest(m.dir, m.editor.ActivePath(), req)
	if err != nil {
		m.setNotice("save failed: "+err.Error(), true)
		return m, nil
	}
	m.editor.SetActivePath(path)
	m.sidebar.SetEntries(entries)
	m.setNotice("saved "+rel(m.dir, path), false)
	return m, nil
}

// renameRequest moves the request at oldPath under its new slug path and
// refreshes the sidebar; the editor follows if it was the active request.
func (m *Model) renameRequest(oldPath, name string) (tea.Model, tea.Cmd) {
	req, newPath, entries, err := app.RenameRequest(m.dir, oldPath, name)
	if err != nil {
		m.setNotice("rename failed: "+err.Error(), true)
		return m, nil
	}
	m.sidebar.SetEntries(entries)
	if oldPath == m.editor.ActivePath() {
		m.editor.SetActivePath(newPath)
		m.urlbar.SetRequest(req.Method, req.URL)
		m.editor.SetRequest(req, newPath)
	}
	m.setNotice("renamed "+rel(m.dir, oldPath)+" → "+rel(m.dir, newPath), false)
	return m, m.saveState()
}

// createFolderIn makes a new directory under dir and refreshes the tree.
func (m *Model) createFolderIn(dir, name string) (tea.Model, tea.Cmd) {
	path, entries, err := app.CreateFolder(m.dir, dir, name)
	if err != nil {
		m.setNotice("create failed: "+err.Error(), true)
		return m, nil
	}
	m.sidebar.SetEntries(entries)
	m.setNotice("created "+rel(m.dir, path), false)
	return m, nil
}

// createRequestIn writes a blank named request under dir, refreshes the
// tree, and loads it into the editor so the user can fill it in.
func (m *Model) createRequestIn(dir, name string) (tea.Model, tea.Cmd) {
	req, path, entries, err := app.CreateRequest(m.dir, dir, name)
	if err != nil {
		m.setNotice("create failed: "+err.Error(), true)
		return m, nil
	}
	m.sidebar.SetEntries(entries)
	m.urlbar.SetRequest(req.Method, req.URL)
	m.setNotice("created "+rel(m.dir, path), false)
	return m, tea.Batch(m.editor.SetRequest(req, path), m.enter(pBar))
}
