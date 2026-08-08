package model

import (
	tea "github.com/charmbracelet/bubbletea"

	"lazypost/internal/app"
	"lazypost/internal/httpclient"
)

// send composes the request (URL/method from the bar, the rest from the
// editor), runs the whole pipeline — pre hook, HTTP call, post hook — off
// the render loop, and shows the result. Results come back as responseMsg
// or errMsg; both carry any store writes made by the hooks.
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
	store := app.CloneVars(m.store)

	cmd := func() tea.Msg {
		res, err := app.Send(httpclient.Exec, *req, vars, store)
		if err != nil {
			return errMsg{err: err, store: res.Store, req: *req}
		}
		return responseMsg{res: res.Response, store: res.Store, req: *req}
	}
	return m, tea.Batch(m.response.StartLoading(), cmd)
}
