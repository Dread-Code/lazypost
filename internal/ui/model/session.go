package model

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Dread-Code/lazypost/internal/session"
)

// restore applies persisted session state: active environment, collapsed
// folders, the last selected request, and the editor section.
func (m *Model) restore(st session.State) {
	if idx := slices.Index(m.envNames, st.Env); idx >= 0 {
		m.envIdx = idx + 1
	}
	m.sidebar.SetCollapsed(m.dir, st.Collapsed)
	if st.ActivePath != "" && m.sidebar.RevealPath(filepath.Join(m.dir, st.ActivePath)) {
		if e := m.sidebar.Selected(); e != nil {
			m.urlbar.SetRequest(e.Req.Method, e.Req.URL)
			m.editor.SetRequest(e.Req, e.Path)
		}
	}
	m.editor.SetSection(st.EditorSection)
}

// quit persists state synchronously (the program is about to exit, so an
// async save could be cut off) then quits.
func (m *Model) quit() tea.Cmd {
	if m.cancelSend != nil {
		m.cancelSend()
		m.cancelSend = nil
	}
	m.activeSendID = 0
	writer := m.sessionWriter
	if writer == nil {
		writer = session.NewWriter()
		m.sessionWriter = writer
	}
	if err := writer.Flush(m.dir, m.snapshot()); err != nil {
		fmt.Fprintln(os.Stderr, "lazypost: save state failed: "+err.Error())
	}
	return tea.Quit
}

// snapshot captures the persisted UI state (env, active request, collapsed
// dirs, editor section) without writing it.
func (m *Model) snapshot() session.State {
	st := m.state
	st.Env = m.activeEnvName()
	if e := m.sidebar.Selected(); e != nil {
		if rel, err := filepath.Rel(m.dir, e.Path); err == nil {
			st.ActivePath = rel
		}
	}
	st.Collapsed = m.sidebar.CollapsedPaths(m.dir)
	st.EditorSection = m.editor.Section()
	return st
}

// saveState snapshots the persisted UI state (env, active request,
// collapsed dirs) to disk off the render loop. Failures surface as a
// status-bar notice, never the response pane (errMsg renders there).
func (m *Model) saveState() tea.Cmd {
	st := m.snapshot()
	writer := m.sessionWriter
	if writer == nil {
		writer = session.NewWriter()
		m.sessionWriter = writer
	}
	revision := writer.Reserve()
	return func() tea.Msg {
		if err := writer.SaveRevision(revision, m.dir, st); err != nil {
			return saveErrMsg{err: err}
		}
		return nil
	}
}
