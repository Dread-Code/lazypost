package model

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Dread-Code/lazypost/internal/app"
	"github.com/Dread-Code/lazypost/internal/collection"
	"github.com/Dread-Code/lazypost/internal/httpclient"
)

// send composes the request (URL/method from the bar, the rest from the
// editor), runs the whole pipeline — pre hook, HTTP call, post hook — off
// the render loop, and shows the result. Results come back as responseMsg
// or errMsg; both carry any store writes made by the hooks.
func (m *Model) send() (tea.Model, tea.Cmd) {
	req, err := m.editor.RequestWithError()
	if err != nil {
		m.setNotice("invalid request: "+err.Error(), true)
		return m, nil
	}
	req.Method = m.urlbar.Method()
	req.URL = m.urlbar.URL()
	if req.URL == "" {
		m.setNotice("URL is required", true)
		return m, nil
	}
	m.setNotice("", false)
	if m.cancelSend != nil {
		m.cancelSend()
	}
	m.nextSendID++
	id := m.nextSendID
	m.activeSendID = id
	ctx, cancel := context.WithCancel(context.Background())
	m.cancelSend = cancel
	vars := app.CloneVars(m.activeVars())
	store := app.CloneVars(m.store)
	path := m.editor.ActivePath()
	executor := m.executor
	if executor == nil {
		executor = func(ctx context.Context, sent collection.Request) (*httpclient.Response, error) {
			return httpclient.ExecuteContext(ctx, sent)
		}
	}

	cmd := func() tea.Msg {
		res, err := app.SendContext(ctx, executor, *req, vars, store)
		if err != nil {
			return errMsg{id: id, err: err, store: res.Store, req: *req, path: path}
		}
		return responseMsg{id: id, res: res.Response, store: res.Store, req: *req, path: path}
	}
	return m, tea.Batch(m.response.StartLoading(), cmd)
}
