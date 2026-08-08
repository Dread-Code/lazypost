package model

import (
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"

	"postgo/internal/collection"
)

// save persists the composed request to disk, then reloads the sidebar
// so the new/changed file appears in the tree.
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

// renameRequest rewrites the request at oldPath under its new slug path
// and removes the old file.
func (m *Model) renameRequest(oldPath, name string) (tea.Model, tea.Cmd) {
	req, err := collection.LoadFile(oldPath)
	if err != nil {
		m.setNotice("rename failed: "+err.Error(), true)
		return m, nil
	}
	req.Name = name
	newPath := filepath.Join(filepath.Dir(oldPath), collection.Slug(name)+".yaml")
	if _, err := collection.Save(m.dir, newPath, req); err != nil {
		m.setNotice("rename failed: "+err.Error(), true)
		return m, nil
	}
	if err := os.Remove(oldPath); err != nil {
		m.setNotice("rename failed: "+err.Error(), true)
		return m, nil
	}
	entries, err := collection.Load(m.dir)
	if err == nil {
		m.sidebar.SetEntries(entries)
	}
	if oldPath == m.editor.ActivePath() {
		m.editor.SetActivePath(newPath)
		m.urlbar.SetRequest(req.Method, req.URL)
		m.editor.SetRequest(req, newPath)
	}
	m.setNotice("renamed "+rel(m.dir, oldPath)+" → "+rel(m.dir, newPath), false)
	return m, m.saveState()
}

// createFolderIn makes a new directory under dir and reloads the tree.
func (m *Model) createFolderIn(dir, name string) (tea.Model, tea.Cmd) {
	path := filepath.Join(dir, collection.Slug(name))
	if err := os.MkdirAll(path, 0o755); err != nil {
		m.setNotice("create failed: "+err.Error(), true)
		return m, nil
	}
	entries, err := collection.Load(m.dir)
	if err == nil {
		m.sidebar.SetEntries(entries)
	}
	m.setNotice("created "+rel(m.dir, path), false)
	return m, nil
}

// createRequestIn writes a blank named request under dir, reloads the
// tree, and loads it into the editor so the user can fill it in.
func (m *Model) createRequestIn(dir, name string) (tea.Model, tea.Cmd) {
	req := &collection.Request{Name: name, Method: "GET"}
	path := filepath.Join(dir, collection.Slug(name)+".yaml")
	if _, err := collection.Save(m.dir, path, req); err != nil {
		m.setNotice("create failed: "+err.Error(), true)
		return m, nil
	}
	entries, err := collection.Load(m.dir)
	if err == nil {
		m.sidebar.SetEntries(entries)
	}
	m.urlbar.SetRequest(req.Method, req.URL)
	m.setNotice("created "+rel(m.dir, path), false)
	return m, tea.Batch(m.editor.SetRequest(req, path), m.enter(pBar))
}
