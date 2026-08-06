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
| `tab`/`shift+tab` | switch panes (collection / request / response) |
| `ctrl+r`       | send request                                  |
| `ctrl+s`       | save request to collection                    |
| `ctrl+e`       | cycle environment                             |
| `enter`        | load highlighted request (collection pane)    |
| `n`            | new request (collection pane)                 |
| `ctrl+↓`/`ctrl+↑` | move between URL / headers / body / auth   |
| `alt+←`/`alt+→` | switch headers / body / auth tabs            |
| `ctrl+t`       | cycle HTTP method (URL field) or auth type    |
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

- curl import/export
- command palette & jump mode
- themes & custom keymaps
- pre/post-request scripting
- query params editor, cookies & trace tabs
