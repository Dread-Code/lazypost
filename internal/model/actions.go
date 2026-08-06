package model

import (
	tea "github.com/charmbracelet/bubbletea"

	"postgo/internal/collection"
	"postgo/internal/httpclient"
)

func (m *Model) send() (tea.Model, tea.Cmd) {
	req := m.editor.Request()
	req.Method = m.urlbar.Method()
	req.URL = m.urlbar.URL()
	if req.URL == "" {
		m.setNotice("URL is required", true)
		return m, nil
	}
	m.setNotice("", false)
	vars := m.activeVars()
	cmd := func() tea.Msg {
		res, err := httpclient.Exec(*req, vars)
		if err != nil {
			return errMsg{err}
		}
		return responseMsg{res}
	}
	return m, tea.Batch(m.response.StartLoading(), cmd)
}

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

// cycleEnv advances the active environment: none → env1 → env2 → … → none.
func (m *Model) cycleEnv() {
	if len(m.envNames) == 0 {
		return
	}
	// +1 slot for "none" at index 0
	m.envIdx = (m.envIdx + 1) % (len(m.envNames) + 1)
	if name := m.activeEnvName(); name != "" {
		m.setNotice("environment: "+name, false)
	} else {
		m.setNotice("environment: none", false)
	}
}

func (m *Model) setNotice(s string, isError bool) {
	m.notice = s
	m.noticeError = isError
}

func (m *Model) activeEnvName() string {
	if m.envIdx == 0 || len(m.envNames) == 0 {
		return ""
	}
	return m.envNames[m.envIdx-1]
}

func (m *Model) activeVars() map[string]string {
	if name := m.activeEnvName(); name != "" {
		return m.envs[name]
	}
	return nil
}
