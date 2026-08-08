package model

import (
	"path/filepath"
	"slices"

	tea "github.com/charmbracelet/bubbletea"

	"postgo/internal/session"
)

// restore applies persisted session state: active environment, collapsed
// folders, and the last selected request.
func (m *Model) restore(st session.State) {
	if idx := slices.Index(m.envNames, st.Env); idx >= 0 {
		m.envIdx = idx + 1
	}
	m.sidebar.SetCollapsed(m.dir, st.Collapsed)
	if st.ActivePath != "" && m.sidebar.SelectPath(filepath.Join(m.dir, st.ActivePath)) {
		if e := m.sidebar.Selected(); e != nil {
			m.urlbar.SetRequest(e.Req.Method, e.Req.URL)
			m.editor.SetRequest(e.Req, e.Path)
		}
	}
}

// quit persists state synchronously (the program is about to exit, so an
// async save could be cut off) then quits.
func (m *Model) quit() tea.Cmd {
	_ = session.Save(m.dir, m.snapshot())
	return tea.Quit
}

// snapshot captures the persisted UI state (env, active request, collapsed
// dirs) without writing it.
func (m *Model) snapshot() session.State {
	st := m.state
	st.Env = m.activeEnvName()
	if e := m.sidebar.Selected(); e != nil {
		if rel, err := filepath.Rel(m.dir, e.Path); err == nil {
			st.ActivePath = rel
		}
	}
	st.Collapsed = m.sidebar.CollapsedPaths(m.dir)
	return st
}

// saveState snapshots the persisted UI state (env, active request,
// collapsed dirs) to disk off the render loop.
func (m *Model) saveState() tea.Cmd {
	st := m.snapshot()
	return func() tea.Msg {
		if err := session.Save(m.dir, st); err != nil {
			return errMsg{err: err}
		}
		return nil
	}
}
