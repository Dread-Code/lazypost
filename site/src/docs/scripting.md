# scripting

Each request can carry `pre` and `post` hooks written in sandboxed Lua — no filesystem, network,
file loading, module loading, or direct terminal output; only `os.time` is exposed. Hooks are
bounded and inherit request cancellation. Edit them in the `Scripts` tab of the request editor.

## pre

Mutate the request before it is sent; a returned table merges into `{{...}}` interpolation:

```lua
-- pre
req.headers["X-Session"] = store.get("token")
req.query["page"] = "2"
return { host = "https://api.example.com" }
```

## post

Inspect the response; returning falsy or a string fails the send with that message:

```lua
-- post
if response.status_code ~= 200 then
  return "expected 200, got " .. tostring(response.status_code)
end
local id = string.match(response.body, '"id": (%d+)')
store.set("last_id", id)
```

## globals

| Global | Meaning |
| --- | --- |
| req | method, url, body, headers, query — mutable in pre |
| env | active variables |
| store.get(key) / store.set(key, value) | session store — one response feeds the next request |
| response | status, status_code, headers, body — post only |
| os.time | the only exposed standard function |
