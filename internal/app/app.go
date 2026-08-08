// Package app contains the application operations that orchestrate the
// domain packages (script, httpclient) behind the TUI. The root model
// (internal/model) invokes Send from a tea.Cmd; the pipeline is testable
// here without a terminal or a live server.
package app

import (
	"fmt"

	"lazypost/internal/collection"
	"lazypost/internal/curl"
	"lazypost/internal/httpclient"
	"lazypost/internal/render"
	"lazypost/internal/script"
)

// Client executes an already-interpolated request. httpclient.Exec is
// the production implementation; tests substitute a fake.
type Client func(req collection.Request, vars map[string]string) (*httpclient.Response, error)

// Result is the outcome of one send: the HTTP response plus any chain
// store writes the hooks produced. Store is populated even on failure
// (a pre hook may have written values before exec failed).
type Result struct {
	Response *httpclient.Response
	Store    map[string]string
}

// CurlLine renders req as a curl one-liner with {{vars}} interpolated
// (unknown placeholders pass through, per ADR-0006).
func CurlLine(req collection.Request, vars map[string]string) string {
	return curl.Format(render.Request(req, vars))
}

// Send runs the request pipeline: pre hook → HTTP exec → post hook.
//
// vars are the interpolation variables (the active environment) and
// store is the session chain store; hooks may mutate both. The caller
// merges Result.Store back into its own chain store.
func Send(c Client, req collection.Request, vars, store map[string]string) (Result, error) {
	if req.Pre != "" {
		extra, err := script.Pre(req.Pre, &req, vars, store)
		if err != nil {
			// pre-hook store writes are discarded, matching the old
			// synchronous-failure path
			return Result{}, err
		}
		vars = MergeVars(vars, extra)
	}

	// interpolation precedence: env → pre-returned vars → store
	vars = MergeVars(vars, store)

	// snapshot for the post hook: it must see the request as sent
	sent := req
	res, err := c(sent, vars)
	if err != nil {
		return Result{Store: store}, err
	}
	if sent.Post != "" {
		if fail, err := script.Post(sent.Post, &sent, vars, store,
			res.Status, res.StatusCode, res.Headers, string(res.Body)); err != nil {
			return Result{Store: store}, err
		} else if fail != "" {
			return Result{Store: store}, fmt.Errorf("post hook: %s", fail)
		}
	}
	return Result{Response: res, Store: store}, nil
}

// MergeVars layers extra over base (extra wins).
func MergeVars(base, extra map[string]string) map[string]string {
	out := make(map[string]string, len(base)+len(extra))
	for k, v := range base {
		out[k] = v
	}
	for k, v := range extra {
		out[k] = v
	}
	return out
}

// CloneVars returns a shallow copy of vars (nil-safe).
func CloneVars(vars map[string]string) map[string]string {
	out := make(map[string]string, len(vars))
	for k, v := range vars {
		out[k] = v
	}
	return out
}
