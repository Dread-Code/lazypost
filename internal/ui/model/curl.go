package model

import (
	"encoding/base64"

	tea "github.com/charmbracelet/bubbletea"

	"lazypost/internal/app"
	"lazypost/internal/clipboard"
	"lazypost/internal/curl"
)

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
	line := app.CurlLine(*req, m.activeVars())
	m.setNotice("curl copied to clipboard", false)
	if err := clipboard.Write(line); err != nil {
		seq := "\x1b]52;c;" + base64.StdEncoding.EncodeToString([]byte(line)) + "\a"
		return m, tea.Printf("%s", seq)
	}
	return m, nil
}
