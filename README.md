# lazypost

[![CI](https://github.com/Dread-Code/lazypost/actions/workflows/ci.yml/badge.svg)](https://github.com/Dread-Code/lazypost/actions/workflows/ci.yml)
[![Go](https://img.shields.io/github/go-mod/go-version/Dread-Code/lazypost)](https://go.dev)
[![Release](https://img.shields.io/github/v/release/Dread-Code/lazypost)](https://github.com/Dread-Code/lazypost/releases)

An API client that lives in your terminal — a [Posting](https://posting.sh)-inspired TUI built with Go and [Bubble Tea](https://github.com/charmbracelet/bubbletea).

Requests are plain YAML files in a directory tree, so a collection is just a folder you can version-control, diff, and share.

<img width="3024" height="1912" alt="image" src="https://github.com/user-attachments/assets/4a1e695d-ed02-4970-9d5c-162a5f65b6b3" />



## Features

- **Request top bar** — method + URL always visible, editable from any pane with `ctrl+l`; `enter` sends
- **Request editor** — query params, headers, body, auth (none / basic / bearer / api key), and per-request Lua scripts
- **Vim editing modes** — every editor field (query, headers, body, scripts) lands in NORMAL mode on focus: `hjkl wbe 0$^ ggG %` motions, `dd yy dw y$` operators with counts, visual selection (`v`/`V`) with yank to the system clipboard, `p`/`P` paste; mode shown in a themed footer row (`q` quits from normal/visual mode)
- **Scripting & chaining** — sandboxed Lua `pre`/`post` hooks share a session `store` (`store.get` / `store.set`), so one response can feed the next request
- **Collections** — requests as readable YAML in a directory tree; `enter` collapses folders, `a`/`d`/`r` add / delete / rename with confirmation
- **Open any directory** — run lazypost anywhere and the current directory (or `-dir`) becomes a collection, marked with `config/config.yaml`
- **Collection importers** — migrate from Postman v2.1 JSON and Insomnia v4 JSON / v5 YAML (single files or export directories) with `lazypost import`; workspaces become top-level folders, requests, environments, headers, query params, bodies, and supported auth are converted to lazypost YAML, with `--dry-run` previews, `--strict` warning enforcement, and staged, collision-safe output
- **Environments** — `{{variable}}` interpolation in URLs, headers, bodies, and auth, resolved from `environments/*.yaml`
- **Response viewer** — status / time / size summary, theme-colored JSON, headers tab, and response retention capped at 16 MiB
- **Request history** — last 20 sends (request + response) in memory; `ctrl+h` browses, enter restores, `ctrl+r` resends
- **Keybindings panel** — `?` for a grouped reference of every binding
- **curl import/export** — paste a `curl` command to import it; `ctrl+g` copies the current request as curl, including `-G` query-data semantics and warnings for unsupported request flags
- **Command palette** — `ctrl+/` to run any action, including switching themes and managing environments
- **Themes** — dracula / catppuccin / solarized / gruvbox / nord / tokyonight / one-dark / monokai presets, plus custom YAML themes
- **Session persistence** — active environment, last request, collapsed folders, active editor tab, and theme survive relaunch

## Install

One command (checksum-verified; installs to `~/.local/bin`, override with `PREFIX`):

```sh
curl -fsSL https://raw.githubusercontent.com/Dread-Code/lazypost/main/install.sh | sh
# pin a version, or pick an install directory:
curl -fsSL https://raw.githubusercontent.com/Dread-Code/lazypost/main/install.sh | sh -s -- v0.4.0
PREFIX=/usr/local sh install.sh
```

Pre-built binaries for macOS and Linux (arm64 + amd64) are attached to every [GitHub Release](https://github.com/Dread-Code/lazypost/releases). Re-running the install command updates to the latest release.

Or build from source (Go 1.25+):

```sh
go build -o lazypost .
```

Check what you're running with `lazypost -version`.

## Quick start

1. `lazypost` in any directory — the current directory is initialized with `config/config.yaml` when needed
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
| `q`            | quit (collection / response / editor normal & visual modes) |
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

Every code field (query, headers, body, scripts) is vim-modal: it lands in **NORMAL** when you focus it, and editing is an explicit `i` away. The current mode shows in a colored footer row at the bottom of the editor.

| Key                | Action                          |
| ------------------ | ------------------------------- |
| `i` / `a` / `A` / `I` / `o` / `O` | enter insert mode (editing) |
| `esc`              | back to NORMAL                  |
| `hjkl` / `wbe` / `0$^` / `ggG` / `%` | motions           |
| `x` / `dd` / `dw` / `d$` / `d0` | delete (with counts: `d2w`, `3dd`) |
| `yy` / `yw` / `y$` | yank (copied to the system clipboard) |
| `p` / `P`          | paste the last yank/delete      |
| `v` / `V`          | visual selection (char/line); `y` yanks, `d` deletes |
| `q`                | quit (normal/visual mode only; in insert it types) |
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

A collection is a directory of YAML files; subdirectories become folders. A root-level `config/config.yaml` marks a directory as a collection:

```yaml
version: 1
```

Open any directory you choose (`-dir` or the current directory) and lazypost initializes it automatically. Existing `.lazypost` markers remain readable for the current session and migrate to `config/config.yaml` on the first write; their legacy name/root fields are discarded. `./sample-collections` / `./collections` are treated as implicit collections without creating a marker.

Collection writes are root-confined and atomic. New requests, folders, environments, and renames refuse path collisions instead of replacing existing data; newly created request and environment files default to owner-only permissions.

### Import existing collections

`lazypost import` converts Postman and Insomnia exports into a lazypost collection **without any TUI interaction**. Run `lazypost import --help` for the full usage:

```sh
lazypost import postman-collection.json \
  -env postman-dev.json -env postman-prod.json \
  -dir ./collections/my-api

lazypost import insomnia-export.yaml \
  -dir ./collections/my-api \
  --dry-run

lazypost import insomnia-export-folder/ \
  -dir ./collections/my-api

lazypost import postman-collection.json \
  -dir ./collections/my-api \
  --dry-run --strict
```

#### Flags

| Flag | Description |
| --- | --- |
| `-dir <target>` | **Required.** Target collection directory. The import refuses to touch an existing directory unless `--force`. |
| `-env <file>` | Import an environment file (Postman environment export JSON, or an Insomnia environment YAML). Repeatable — pass once per environment. |
| `--format postman\|insomnia` | Override automatic format detection. |
| `--dry-run` | Parse, validate, and print what **would** be imported (counts + warnings) without writing anything. |
| `--force` | Replace an existing target: the old directory is moved aside and removed only after the new tree is in place. |
| `--strict` | Fail the import instead of succeeding when any warning is produced. |

#### Behavior

- **Supported sources:** Postman Collection v2.1 JSON, Insomnia v4 JSON exports (`__export_format: 4`), and Insomnia v5 YAML — either a single collection file or a full export directory.
- **Format detection** is automatic from the file contents; `--format` is only needed for ambiguous inputs.
- **Workspaces:** a Postman collection name or an Insomnia workspace becomes a top-level folder in the imported tree. Insomnia export directories combine **all** workspaces found inside and skip unrelated resources (mock servers, OpenAPI documents) with warnings.
- **Environments:** collection/base variables become a `base` environment, plus one per named environment (`-env` files included). With multiple workspaces, environment names are logically namespaced as `<workspace>--<environment>` (slugged filenames use `<workspace>-<environment>`); unscoped directory environments use `shared-<environment>` with a warning. Insomnia's `{{ _.var }}` placeholders are normalized to `{{var}}`.
- **URL queries:** structured Postman/Insomnia query parameters become the request's canonical `query` list, so raw URL queries are not sent twice; intentional repeated values are preserved.
- **Writer safety:** the import validates everything first, stages the full tree in a temporary sibling directory, and only then renames it into place. Workspace, environment, and request filename collisions receive deterministic suffixes with warnings. A `config/config.yaml` marker is created automatically.
- **Warnings:** unsupported features — JavaScript pre/test scripts (not translatable to Lua), multipart/binary/GraphQL bodies, and unsupported auth schemes — are reported per request and omitted, never guessed. `--strict` promotes any warning to a failure.

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

Each request can carry `pre` and `post` hooks written in sandboxed Lua (no filesystem, network, file-loading, module-loading, or direct terminal output — only `os.time` is exposed from the standard `os` table). Hooks are bounded and inherit request cancellation. Edit them in the **Scripts** tab of the request editor.

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

Eight presets ship built in — dracula (default), catppuccin, solarized, gruvbox, nord, tokyonight, one-dark, monokai — switchable at runtime from the palette (`ctrl+/` → **Switch theme**); the chosen theme is remembered between runs.

Custom themes live in `~/.config/lazypost/themes/<name>.yaml` (or `$XDG_CONFIG_HOME/lazypost/themes/`). Each file mirrors the theme colors as `light`/`dark` hex pairs; every key is optional and falls back to the default theme. See [`docs/themes/example.yaml`](docs/themes/example.yaml) for an annotated template.

## Development

```sh
gofmt -l . && go vet ./... && go test ./... && go test -race ./...
go vet ./lib/codeeditor && go test ./lib/codeeditor && go test -race ./lib/codeeditor
```

The editor widget lives in [`lib/codeeditor/`](lib/codeeditor) as a package in the root Go module — lazypost wires it to the chroma + theme highlighters in `internal/ui/widgets/highlighters.go`.

CI (`.github/workflows/ci.yml`) runs the same checks plus `go mod tidy`, `go build`, and race tests on every push to `main` and pull request, including `lib/codeeditor` in the root module; a green `test` check is required on `main`.

## Roadmap

### Done

- Request history, keybindings panel, response highlighting, script editor, themes, session persistence
- Open the current directory as a collection, versioned `config/config.yaml` marker
- Postman and Insomnia collection import with workspace preservation (workspaces → top-level folders, environment namespacing)
- Vim editing modes in every editor field (motions, operators, visual selection, yank)
- CI on push, releases + `install.sh`, `-version` stamp

### Later

- **Cookies & trace tabs** · **LSP for scripts**
- **Copy / cut / paste** in the sidebar
