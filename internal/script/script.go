// Package script runs per-request Lua hooks (pre/post) in a sandboxed
// gopher-lua state. Only safe libraries are opened — a hook cannot touch
// the filesystem or network ([[Design - scripting hooks]]).
package script

import (
	"context"
	"fmt"
	"time"

	lua "github.com/yuin/gopher-lua"

	"lazypost/internal/collection"
)

// maxPostBody caps the response body handed to a post hook; larger bodies
// are truncated (the full body still renders in the response pane).
const maxPostBody = 6 << 20 // 6 MiB

// timeout bounds any single hook so a runaway script can't hang a send.
const timeout = 5 * time.Second

// newState builds a sandboxed LState: safe base libs plus a hand-rolled
// os.time (wrapping time.Now().Unix()). The full os lib is never opened —
// it also carries execute, remove, rename, setenv.
func newState() *lua.LState {
	ls := lua.NewState()
	for _, open := range []func(*lua.LState) int{
		lua.OpenBase, lua.OpenTable, lua.OpenString, lua.OpenMath,
	} {
		open(ls)
	}
	osmod := ls.NewTable()
	ls.SetField(osmod, "time", ls.NewFunction(func(L *lua.LState) int {
		L.Push(lua.LNumber(time.Now().Unix()))
		return 1
	}))
	ls.SetGlobal("os", osmod)
	return ls
}

// run executes src with ctx bound and returns (result, nil) or
// (nil, error) on parse/run errors.
func run(ls *lua.LState, ctx context.Context, src string) (lua.LValue, error) {
	ls.SetContext(ctx)
	if err := ls.DoString(src); err != nil {
		return nil, err
	}
	return ls.Get(-1), nil
}

// Pre evaluates a pre-request hook. The hook sees req as a Lua table, env
// as its active vars, and store as a get/set table over the given map
// (mutations write back in place). It may mutate req's headers/query/body
// (on the copy passed in — the caller owns it) or return a table of vars
// that get merged into interpolation. It applies any mutations back to
// req and returns the extra vars map.
func Pre(src string, req *collection.Request, vars, store map[string]string) (map[string]string, error) {
	ls := newState()
	defer ls.Close()
	setup(ls, req, vars, store, nil)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	res, err := run(ls, ctx, src)
	if err != nil {
		return nil, fmt.Errorf("pre script: %w", err)
	}
	if reqTbl, ok := ls.GetGlobal("req").(*lua.LTable); ok {
		applyMutations(req, reqTbl)
	}
	return readVars(res), nil
}

// applyMutations copies a mutated Lua req table back onto the Go request
// (headers, query, body, url, method).
func applyMutations(req *collection.Request, t *lua.LTable) {
	if v := t.RawGetString("method"); v != lua.LNil {
		req.Method = lua.LVAsString(v)
	}
	if v := t.RawGetString("url"); v != lua.LNil {
		req.URL = lua.LVAsString(v)
	}
	if v := t.RawGetString("body"); v != lua.LNil {
		req.Body = lua.LVAsString(v)
	}
	req.Headers = tableKV(t.RawGetString("headers"))
	req.Query = tableParams(t.RawGetString("query"))
}

// tableKV converts a Lua name→value table into a Header slice.
func tableKV(v lua.LValue) []collection.Header {
	t, ok := v.(*lua.LTable)
	if !ok {
		return nil
	}
	var out []collection.Header
	t.ForEach(func(k, v lua.LValue) {
		out = append(out, collection.Header{Name: lua.LVAsString(k), Value: lua.LVAsString(v)})
	})
	return out
}

// tableParams converts a Lua name→value table into a Param slice.
func tableParams(v lua.LValue) []collection.Param {
	t, ok := v.(*lua.LTable)
	if !ok {
		return nil
	}
	var out []collection.Param
	t.ForEach(func(k, v lua.LValue) {
		out = append(out, collection.Param{Name: lua.LVAsString(k), Value: lua.LVAsString(v)})
	})
	return out
}

// Post evaluates a post-request hook against the response. It returns ""
// on pass, or a non-empty message when the hook returned falsy or an
// error string. Store mutations made by the hook are applied to store in
// place.
func Post(src string, req *collection.Request, vars, store map[string]string, statusText string, statusCode int, headers map[string][]string, body string) (string, error) {
	ls := newState()
	defer ls.Close()
	body = truncate(body, maxPostBody)
	setup(ls, req, vars, store, responseTable(ls, statusText, statusCode, headers, body))

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	res, err := run(ls, ctx, src)
	if err != nil {
		return "", fmt.Errorf("post script: %w", err)
	}
	if res == lua.LNil || res == lua.LFalse {
		return "post hook failed", nil
	}
	if s, ok := res.(lua.LString); ok {
		if s == "" {
			return "post hook failed", nil
		}
		return string(s), nil
	}
	return "", nil
}

// truncate clips s to n bytes.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
