package model

import (
	"encoding/base64"

	tea "github.com/charmbracelet/bubbletea"

	"postgo/internal/clipboard"
	"postgo/internal/collection"
	"postgo/internal/curl"
	"postgo/internal/httpclient"
	"postgo/internal/render"
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

// importCurl parses a pasted curl command into the bar and editor,
// replacing whatever was there ([[Design - curl import export]]).
func (m *Model) importCurl(text string) (tea.Model, tea.Cmd) {
	req, err := curl.Parse(text)
	if err != nil {
		m.setNotice("curl import failed: "+err.Error(), true)
		return m, nil
	}
	m.urlbar.SetRequest(req.Method, req.URL)
	m.editor.SetRequest(req, "")
	m.setNotice("imported curl request", false)
	return m, nil
}

// exportCurl writes the current request as a curl one-liner to the
// clipboard, interpolated with the active environment. Uses the platform
// tool (pbcopy etc.); falls back to OSC52, which Terminal.app ignores but
// iTerm2/Ghostty/kitty/wezterm honor.
func (m *Model) exportCurl() (tea.Model, tea.Cmd) {
	req := m.editor.Request()
	req.Method = m.urlbar.Method()
	req.URL = m.urlbar.URL()
	if req.URL == "" {
		m.setNotice("nothing to export: URL is empty", true)
		return m, nil
	}
	line := curlExportLine(*req, m.activeVars())
	m.setNotice("curl copied to clipboard", false)
	if err := clipboard.Write(line); err != nil {
		seq := "\x1b]52;c;" + base64.StdEncoding.EncodeToString([]byte(line)) + "\a"
		return m, tea.Printf("%s", seq)
	}
	return m, nil
}

// curlExportLine renders req as a curl command with {{vars}} interpolated
// from vars (unknown placeholders pass through, per ADR-0006).
func curlExportLine(req collection.Request, vars map[string]string) string {
	return curl.Format(render.Request(req, vars))
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
