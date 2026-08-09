# lazypost

[![CI](https://github.com/Dread-Code/lazypost/actions/workflows/ci.yml/badge.svg)](https://github.com/Dread-Code/lazypost/actions/workflows/ci.yml)
[![Go](https://img.shields.io/github/go-mod/go-version/Dread-Code/lazypost)](https://go.dev)
[![Release](https://img.shields.io/github/v/release/Dread-Code/lazypost)](https://github.com/Dread-Code/lazypost/releases)

An API client that lives in your terminal — a [Posting](https://posting.sh)-inspired TUI built with Go and [Bubble Tea](https://github.com/charmbracelet/bubbletea).

Requests are plain YAML files in a directory tree, so a collection is just a folder you can version-control, diff, and share.

```
 GET  https://api.example.com/posts                              env: dev
╭ Collection ────────────────╮ ╭─ Request · posts/create ────────────────╮
│ ▸ posts/                   │ │  Query  Headers  Body  Auth  Scripts    │
│     POST create            │ │ ──────────────────────────────────────  │
│     GET  get all           │ │ Content-Type: application/json          │
│   users/                   │ └─────────────────────────────────────────┘
│     GET  get one           │ ╭─ Response · 201 Created · 65 B · 524ms─╮
╰────────────────────────────╯ │  Body  Headers                          │
                               │ ──────────────────────────────────────  │
                               │  { "id": 101 }                          │
                               ╰─────────────────────────────────────────╯
```

## Features

- **Request top bar** — method + URL always visible, editable from any pane with `ctrl+l`; `enter` sends
- **Request editor** — query params, headers, body, auth (none / basic / bearer / api key), and per-request Lua scripts
- **Scripting & chaining** — sandboxed Lua `pre`/`post` hooks share a session `store` (`store.get` / `store.set`), so one response can feed the next request
- **Collections** — requests as readable YAML in a directory tree; `enter` collapses folders, `a`/`d`/`r` add / delete / rename with confirmation
- **Open any directory** — run lazypost anywhere and the current directory (or `-dir`) becomes a collection, marked with a `.lazypost` file
- **Environments** — `{{variable}}` interpolation in URLs, headers, bodies, and auth, resolved from `environments/*.yaml`
- **Response viewer** — status / time / size summary, theme-colored JSON, headers tab
- **Request history** — last 20 sends (request + response) in memory; `ctrl+h` browses, enter restores, `ctrl+r` resends
- **Keybindings panel** — `?` for a grouped reference of every binding
- **curl import/export** — paste a `curl` command to import it; `ctrl+g` copies the current request as curl
- **Command palette** — `ctrl+/` to run any action, including switching themes and managing environments
- **Themes** — dracula / catppuccin / solarized presets
- **Session persistence** — active environment, last request, collapsed folders, active editor tab, and theme survive relaunch

## Install

One command (checksum-verified; installs to `~/.local/bin`, override with `PREFIX`):

```sh
curl -fsSL https://raw.githubusercontent.com/Dread-Code/lazypost/main/install.sh | sh
# pin a version, or pick an install directory:
curl -fsSL https://raw.githubusercontent.com/Dread-Code/lazypost/main/install.sh | sh -s -- v0.1.0
PREFIX=/usr/local sh install.sh
```

Pre-built binaries for macOS and Linux (arm64 + amd64) are attached to every [GitHub Release](https://github.com/Dread-Code/lazypost/releases). Re-running the install command updates to the latest release.

Or build from source (Go 1.25+):

```sh
go build -o lazypost .
```

Check what you're running with `lazypost -version`.

## Quick start

1. `lazypost` in any directory — when asked, give the collection a name (this writes a `.lazypost` marker)
2. `n` creates a request; `ctrl+l` edits the URL, `ctrl+t` cycles the method
3. `ctrl+s` saves, `ctrl+r` sends
4. Drop environment files in `environments/` and switch with `ctrl+e`

## Run

```sh
./lazypost                    # uses ./sample-collections or ./collections if present, else the current directory
./lazypost -dir my-collection # or point at your own collection directory
```

## Keybindings

### Global

| Key            | Action                                  |
| -------------- | --------------------------------------- |
| `tab`          | switch panes                            |
| `ctrl+/`       | command palette                         |
| `?`            | keybindings panel                       |
| `ctrl+h`       | request history                         |
| `ctrl+r`       | send request                            |
| `ctrl+e`       | cycle environment                       |
| `ctrl+s`       | save request                            |
| `ctrl+l`       | jump to the URL bar                     |
| `ctrl+g`       | export current request as curl          |
| `q`            | quit (collection / response pane)       |
| `ctrl+c`       | quit                                    |

### Collection · sidebar

| Key              | Action                                                        |
| ---------------- | ------------------------------------------------------------- |
| `↑`/`↓`, `ctrl+n`/`ctrl+p` | navigate (loads the request)                    |
| `enter`          | focus the URL bar / toggle folder (collection root toggles all) |
| `n`              | new request                                                  |
| `a`              | add request in folder; lead with `/` for a folder             |
| `d`              | delete (confirm with `y`)                                     |
| `r`              | rename                                                       |

### URL bar

| Key            | Action                        |
| -------------- | ----------------------------- |
| `ctrl+t`       | cycle method                  |
| `enter`        | send                          |
| `esc`          | back to previous pane         |
| paste `curl …` | import a curl command         |

### Editor

| Key                | Action                          |
| ------------------ | ------------------------------- |
| `ctrl+n`/`ctrl+p`  | move between sections           |
| `alt+←`/`alt+→`    | switch tabs                     |
| `ctrl+t`           | cycle auth type                 |
| `ctrl+s`           | save                            |

### Response

| Key              | Action              |
| ---------------- | ------------------- |
| `←`/`→`, `b`/`h` | switch tabs         |
| `↑`/`↓`          | scroll              |
| `q`              | quit                |

## Collections

A collection is a directory of YAML files; subdirectories become folders. A `.lazypost` marker marks a directory as a collection and supplies its display name:

```yaml
name: My API collection      # title bar name
root: ~/APIs/main            # optional: point here at the real collection root
```

Open any directory you choose (`-dir` or the current directory) and lazypost becomes that collection: without a marker it asks for a name and writes the marker on confirm (`esc` opens it anyway, writing nothing). `./sample-collections` / `./collections` are treated as collections without a marker.

### Request format

```yaml
name: create post
method: POST
url: "{{host}}/posts"
headers:
  - name: Content-Type
    value: application/json
auth:
  type: bearer        # none | basic | bearer | apikey
  token: "{{api_token}}"
body: |
  {"title": "hello"}
```

Auth variants:

```yaml
auth: { type: basic, username: u, password: p }
auth: { type: bearer, token: t }
auth: { type: apikey, keyName: X-Api-Key, keyValue: k, keyIn: header } # or keyIn: query
```

## Environments

Put YAML files in `<collection>/environments/`:

```yaml
# environments/dev.yaml
variables:
  host: https://api.dev.example.com
  api_token: secret
```

Select an environment with `ctrl+e`; `{{host}}`-style placeholders are substituted at send time. Unknown placeholders are left as-is.

Manage variables in the TUI: `ctrl+/` → **Environments** opens the environment manager (tab bar of environments; `ctrl+e` cycles tabs, `a`/`r`/`d` add/edit/delete `key=value` variables, `enter` activates the tab's env). A leading `/` in the add-variable prompt creates a new empty environment instead.

## Scripting

Each request can carry `pre` and `post` hooks written in sandboxed Lua (no filesystem, no network — only `os.time` is exposed). Edit them in the **Scripts** tab of the request editor.

**Pre** — mutate the request before it's sent; a returned table merges into `{{...}}` interpolation:

```lua
req.headers["X-Session"] = store.get("token")
req.query["page"] = "2"
return { host = "https://api.example.com" }
```

**Post** — inspect the response; returning falsy or a string fails the send with that message:

```lua
if response.status_code ~= 200 then
  return "expected 200, got " .. tostring(response.status_code)
end
local id = string.match(response.body, '"id": (%d+)')
store.set("last_id", id)
```

Globals: `req` (method, url, body, headers, query — mutable in `pre`), `env` (active variables), `store.get(key)` / `store.set(key, value)` (session store), `response` (status, status_code, headers, body — `post` only), `os.time`.

## Themes

Three presets ship built in — dracula (default), catppuccin, solarized — switchable at runtime from the palette (`ctrl+/` → **Switch theme**); the chosen theme is remembered between runs.

Custom themes live in `~/.config/lazypost/themes/<name>.yaml` (or `$XDG_CONFIG_HOME/lazypost/themes/`). Each file mirrors the theme colors as `light`/`dark` hex pairs; every key is optional and falls back to the default theme. See [`docs/themes/example.yaml`](docs/themes/example.yaml) for an annotated template.

## Development

```sh
gofmt -l . && go vet ./... && go test ./...
```

CI (`.github/workflows/ci.yml`) runs the same checks plus `go mod tidy` and `go build` on every push to `main` and pull request; a green `test` check is required on `main`.

## Roadmap

### Done

- Request history, keybindings panel, response highlighting, script editor, themes, session persistence
- Open the current directory as a collection, `.lazypost` marker
- CI on push, releases + `install.sh`, `-version` stamp

### Later

- **Importers (Postman/Insomnia)** · **cookies & trace tabs** · **vim modes** · **LSP for scripts**
- **Copy / cut / paste** in the sidebar
