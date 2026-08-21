// Package app contains the application operations that orchestrate the
// domain packages (script, httpclient) behind the TUI. The root model
// (internal/model) invokes Send from a tea.Cmd; the pipeline is testable
// here without a terminal or a live server.
package app

import (
	"context"
	"fmt"

	"github.com/Dread-Code/lazypost/internal/collection"
	"github.com/Dread-Code/lazypost/internal/curl"
	"github.com/Dread-Code/lazypost/internal/httpclient"
	"github.com/Dread-Code/lazypost/internal/render"
	"github.com/Dread-Code/lazypost/internal/script"
)

// Client is the legacy execution seam. It receives the request after pre-hook
// mutation plus the variables used for interpolation. New UI code should use
// ContextClient through SendContext.
type Client func(req collection.Request, vars map[string]string) (*httpclient.Response, error)

// ContextClient executes the fully rendered request with cancellation.
type ContextClient func(context.Context, collection.Request) (*httpclient.Response, error)

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
	return sendPipeline(context.Background(), req, vars, store, func(_ context.Context, raw, _ collection.Request, effectiveVars map[string]string) (*httpclient.Response, error) {
		return c(raw, effectiveVars)
	})
}

// SendContext runs the request pipeline with a context-aware executor. The
// executor receives the exact rendered request that post hooks observe.
func SendContext(ctx context.Context, c ContextClient, req collection.Request, vars, store map[string]string) (Result, error) {
	return sendPipeline(ctx, req, vars, store, func(ctx context.Context, _, sent collection.Request, _ map[string]string) (*httpclient.Response, error) {
		return c(ctx, sent)
	})
}

func sendPipeline(ctx context.Context, req collection.Request, vars, store map[string]string, execute func(context.Context, collection.Request, collection.Request, map[string]string) (*httpclient.Response, error)) (Result, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	workingStore := CloneVars(store)
	if req.Pre != "" {
		extra, err := script.PreContext(ctx, req.Pre, &req, vars, workingStore)
		if err != nil {
			// pre-hook store writes are discarded, matching the old
			// synchronous-failure path
			return Result{}, err
		}
		vars = MergeVars(vars, extra)
	}

	// interpolation precedence: env → pre-returned vars → store
	vars = MergeVars(vars, workingStore)

	// Render once for the canonical executor and post hook. The legacy
	// executor receives the pre-rendered request plus variables through its
	// compatibility path.
	sent := render.Request(req, vars)
	res, err := execute(ctx, req, sent, vars)
	if err != nil {
		return Result{Store: workingStore}, err
	}
	if sent.Post != "" {
		if fail, err := script.PostContext(ctx, sent.Post, &sent, vars, workingStore,
			res.Status, res.StatusCode, res.Headers, string(res.Body)); err != nil {
			return Result{Store: workingStore}, err
		} else if fail != "" {
			return Result{Store: workingStore}, fmt.Errorf("post hook: %s", fail)
		}
	}
	return Result{Response: res, Store: workingStore}, nil
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
