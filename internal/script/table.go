package script

import (
	"strings"

	lua "github.com/yuin/gopher-lua"

	"github.com/Dread-Code/lazypost/internal/collection"
)

// setup populates the sandbox globals for a hook: env (active vars), req
// (a table mirroring the request), and store (get/set functions reading
// and writing the given map — mutations are written back in place). When
// res is non-nil it is exposed as the response table for post hooks.
func setup(ls *lua.LState, req *collection.Request, vars map[string]string, store map[string]string, res *lua.LTable) {
	if store == nil {
		store = map[string]string{}
	}
	env := ls.NewTable()
	for k, v := range vars {
		env.RawSetString(k, lua.LString(v))
	}
	ls.SetGlobal("env", env)

	storeTbl := ls.NewTable()
	ls.SetField(storeTbl, "get", ls.NewFunction(func(L *lua.LState) int {
		key := L.CheckString(1)
		if v, ok := store[key]; ok {
			L.Push(lua.LString(v))
		} else {
			L.Push(lua.LNil)
		}
		return 1
	}))
	ls.SetField(storeTbl, "set", ls.NewFunction(func(L *lua.LState) int {
		key := L.CheckString(1)
		val := L.CheckString(2)
		store[key] = val
		return 0
	}))
	ls.SetGlobal("store", storeTbl)

	reqTbl := ls.NewTable()
	reqTbl.RawSetString("method", lua.LString(req.Method))
	reqTbl.RawSetString("url", lua.LString(req.URL))
	reqTbl.RawSetString("body", lua.LString(req.Body))
	reqTbl.RawSetString("headers", kvTable(ls, req.Headers))
	reqTbl.RawSetString("query", kvTable(ls, req.Query))
	ls.SetGlobal("req", reqTbl)

	if res != nil {
		ls.SetGlobal("response", res)
	}
}

// kvTable converts a name/value slice into a Lua map.
func kvTable(ls *lua.LState, list any) *lua.LTable {
	t := ls.NewTable()
	switch items := list.(type) {
	case []collection.Header:
		for _, h := range items {
			t.RawSetString(h.Name, lua.LString(h.Value))
		}
	case []collection.Param:
		for _, p := range items {
			t.RawSetString(p.Name, lua.LString(p.Value))
		}
	}
	return t
}

// readVars extracts a returned vars table (name → scalar) from a hook
// result. Non-table results yield nil.
func readVars(res lua.LValue) map[string]string {
	t, ok := res.(*lua.LTable)
	if !ok {
		return nil
	}
	out := make(map[string]string, t.Len())
	t.ForEach(func(k, v lua.LValue) {
		out[lua.LVAsString(k)] = lua.LVAsString(v)
	})
	return out
}

// responseTable builds the response table for post hooks: status (text),
// status_code (number), headers (name → string), body (string).
func responseTable(ls *lua.LState, statusText string, statusCode int, headers map[string][]string, body string) *lua.LTable {
	t := ls.NewTable()
	t.RawSetString("status", lua.LString(statusText))
	t.RawSetString("status_code", lua.LNumber(statusCode))
	h := ls.NewTable()
	for name, vals := range headers {
		h.RawSetString(name, lua.LString(strings.Join(vals, ", ")))
	}
	t.RawSetString("headers", h)
	t.RawSetString("body", lua.LString(body))
	return t
}
