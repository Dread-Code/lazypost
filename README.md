# lazypost

An API client that lives in your terminal — a [Posting](https://posting.sh)-inspired TUI built with Go and [Bubble Tea](https://github.com/charmbracelet/bubbletea).

```
lazypost  sample-collections                          env: dev
GET https://api.example.com/posts          ctrl+t method
╭ Collection ────────────────╮ ╭─ Request · posts/create ────────────────╮
│ ▸ posts/                   │ │  Query  Headers  Body  Auth  Scripts    │
│     POST create            │ │ ┌─────────────────────────────────────┐ │
│     GET  get all           │ │ │ Content-Type: application/json      │ │
│   users/                   │ │ └─────────────────────────────────────┘ │
│     GET  get one           │ ╰─────────────────────────────────────────╯
╰────────────────────────────╯ ╭─ Response · 201 Created · 65 B · 524ms─╮
                               │  Body  Headers                          │
                               │  { "id": 101 }                          │
                               ╰─────────────────────────────────────────╯
```

## Features

- **Request top bar** — method + URL always visible, reachable from any pane with `ctrl+l`; `enter` sends
- **Request editor** — query params, headers, body, auth (none / basic / bearer / api key), and per-request Lua scripts (`pre`/`post` hooks)
- **Scripting & chaining** — sandboxed Lua `pre`/`post` hooks per request share a session `store` (`store.get` / `store.set`), so one response can feed the next request
- **Collections** — requests stored as readable, version-control-friendly YAML files in a directory tree; navigating the sidebar loads the selected request, `enter` collapses folders, `a`/`d`/`r` add / delete / rename with confirmation
- **Environments** — `{{variable}}` interpolation in URLs, headers, bodies, and auth, resolved from environment files
- **Response viewer** — status/time/size summary, pretty-printed JSON, headers tab
- **curl import/export** — paste a `curl` command into the URL bar to import it; `ctrl+g` copies the current request as curl
- **Command palette** — `ctrl+/` to filter and run any action, incl. switching themes and managing environments
- **Themes** — dracula / catppuccin / solarized presets, switched from the palette
- **Session persistence** — active environment, last request, collapsed folders, and theme survive relaunch

## Install

```sh
go build -o lazypost .
```

Requires Go 1.25+.

## Run

```sh
./lazypost                    # uses ./sample-collections if present
./lazypost -dir my-collection # or point at your own collection directory
```

## Keybindings

| Key            | Action                                        |
| -------------- | --------------------------------------------- |
| `tab`/`shift+tab` | switch panes (collection / request / response; the URL bar is reached with `ctrl+l` or `enter` on a request) |
| `ctrl+r`       | send request                                  |
| `ctrl+s`       | save request to collection                    |
| `ctrl+e`       | cycle environment                             |
| `ctrl+l`       | jump to the URL bar                           |
| `ctrl+/`       | open the command palette                     |
| `ctrl+g`       | copy current request as curl (clipboard)      |
| paste `curl …` | import a curl command (URL bar)               |
| `enter`        | focus the URL bar / toggle folder or all folders on the collection root (collection pane) / send (URL bar); navigating with `↑`/`↓` loads the selected request into the URL bar and editor |
| `esc`          | leave the URL bar                             |
| `n`            | new request (collection pane)                 |
| `a`            | add a request in the highlighted folder; lead with `/` to create a folder (collection pane) |
| `d`            | delete the highlighted request/folder (confirm with `y`, cancel with `n`) |
| `r`            | rename the highlighted request (collection pane)   |
| `ctrl+n`/`ctrl+p` | move between query / headers / body / auth / scripts |
| `alt+←`/`alt+→` | switch query / headers / body / auth / scripts tabs |
| `ctrl+t`       | cycle HTTP method (URL bar) or auth type      |
| `b` / `h`      | body / headers tab (response pane)            |
| `q`            | quit (collection or response pane)            |
| `ctrl+c`       | quit                                          |

## Collection format

A collection is a directory of YAML files (subdirectories become tree nodes):

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

## Development

```sh
go build ./...
go vet ./...
go test ./...
```

## Roadmap

### Next — usability

- **Request history** — resend with one key

### Later — power

- **Importers (Postman/Insomnia)**
- **Cookies & trace tabs**
- **Vim modes** — editor/response normal·visual·insert
- **Help panel window** — toggleable keybinding reference

### Parked ideas

- **Shared collection scripts** — collection-level `scripts/*.lua`
- **Jump mode / custom keymaps** — deferred at MVP (see ADR-0004); revisit via the keymap registry

### Housekeeping

- First benchmark (startup + collection load), CI on push, release process when sharing
