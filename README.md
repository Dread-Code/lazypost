# postgo

An API client that lives in your terminal — a [Posting](https://posting.sh)-inspired TUI built with Go and [Bubble Tea](https://github.com/charmbracelet/bubbletea).

```
╭ Collection ────────╮ ╭─ Request ────────────────────────────╮
│ ▸ posts/           │ │ POST https://api.example.com/posts   │
│     POST create    │ │  Headers   Body   Auth               │
│     GET  get all   │ │ ┌──────────────────────────────────┐ │
│   users/           │ │ │ Content-Type: application/json   │ │
│     GET  get one   │ │ └──────────────────────────────────┘ │
╰────────────────────╯ ╰──────────────────────────────────────╯
                       ╭─ Response · 201 Created · 65 B · 524ms╮
                       │  Body   Headers                       │
                       │ {                                     │
                       │   "id": 101                           │
                       │ }                                     │
                       ╰───────────────────────────────────────╯
```

## Features (MVP)

- **Request editor** — method selector, URL bar, headers, body, and auth (none / basic / bearer / api key)
- **Collections** — requests stored as readable, version-control-friendly YAML files in a directory tree
- **Environments** — `{{variable}}` interpolation in URLs, headers, bodies, and auth, resolved from environment files
- **Response viewer** — status/time/size summary, pretty-printed JSON, headers tab
- **curl import/export** — paste a `curl` command into the URL bar to import it; `ctrl+g` copies the current request as curl
- **Command palette** — `ctrl+/` to filter and run any action

## Install

```sh
go build -o postgo .
```

Requires Go 1.23+.

## Run

```sh
./postgo                    # uses ./sample-collections if present
./postgo -dir my-collection # or point at your own collection directory
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
| `enter`        | load request / toggle folder or all folders on the collection root (collection pane) / send (URL bar) |
| `esc`          | leave the URL bar                             |
| `n`            | new request (collection pane)                 |
| `a`            | add a request in the highlighted folder (collection pane) |
| `ctrl+n`/`ctrl+p` | move between headers / body / auth         |
| `alt+←`/`alt+→` | switch headers / body / auth tabs            |
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

## Development

```sh
go build ./...
go vet ./...
go test ./...
```

## Roadmap ideas

- command palette & jump mode
- themes & custom keymaps
- pre/post-request scripting
- query params editor, cookies & trace tabs
