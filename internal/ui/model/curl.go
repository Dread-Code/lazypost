package model

import (
	"context"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"lazypost/internal/app"
	"lazypost/internal/clipboard"
	"lazypost/internal/curl"
)

// importCurl parses a pasted curl command into the bar and editor,
// replacing whatever was there ([[Design - curl import export]]).
func (m *Model) importCurl(text string) (tea.Model, tea.Cmd) {
	req, warnings, err := curl.ParseWithWarnings(text)
	if err != nil {
		m.setNotice("curl import failed: "+err.Error(), true)
		return m, nil
	}
	m.urlbar.SetRequest(req.Method, req.URL)
	m.editor.SetRequest(req, "")
	if len(warnings) > 0 {
		m.setNotice(fmt.Sprintf("imported curl request with %d warning(s)", len(warnings)), true)
	} else {
		m.setNotice("imported curl request", false)
	}
	return m, nil
}

// exportCurl writes the current request as a curl one-liner to the
// clipboard, interpolated with the active environment. Uses the platform
// tool (pbcopy etc.); falls back to OSC52, which Terminal.app ignores but
// iTerm2/Ghostty/kitty/wezterm honor.
func (m *Model) exportCurl() (tea.Model, tea.Cmd) {
	req, err := m.editor.RequestWithError()
	if err != nil {
		m.setNotice("invalid request: "+err.Error(), true)
		return m, nil
	}
	req.Method = m.urlbar.Method()
	req.URL = m.urlbar.URL()
	if req.URL == "" {
		m.setNotice("nothing to export: URL is empty", true)
		return m, nil
	}
	line := app.CurlLine(*req, m.activeVars())
	return m, clipboardCommand(line, "curl copied to clipboard")
}

func clipboardCommand(text, notice string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		return clipboardMsg{text: text, notice: notice, err: clipboard.WriteContext(ctx, text)}
	}
}
