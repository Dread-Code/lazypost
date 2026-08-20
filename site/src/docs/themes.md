# themes

Eight presets ship built in:

`dracula`, `catppuccin`, `solarized`, `gruvbox`, `nord`, `tokyonight`, `one-dark`, `monokai`

Switch at runtime from the palette: `ctrl+/` then `Switch theme`. The chosen theme is remembered between runs.

## custom YAML themes

Custom themes live in `~/.config/lazypost/themes/<name>.yaml` (or
`$XDG_CONFIG_HOME/lazypost/themes/`). Each file mirrors the theme colors as `light`/`dark` hex
pairs; every key is optional and falls back to the default theme.

see the annotated template: example.yaml — or in the repo: docs/themes/example.yaml

```yaml
# docs/themes/example.yaml
accent: {light: "#7aa2f7", dark: "#7aa2f7"}
background: {light: "#c0caf5", dark: "#24283b"}
text: {light: "#0f111a", dark: "#c0caf5"}
```

## example.yaml

The annotated template — every key commented, every value an adaptive light/dark pair. Same file
shipped in the repo at `docs/themes/example.yaml`.

```yaml
# Example user theme for lazypost.

# Drop a file like this into ~/.config/lazypost/themes/ (or
# $XDG_CONFIG_HOME/lazypost/themes/) and it appears in the theme picker
# (ctrl+/ → Switch theme) after the built-in presets. The theme's name is
# the file name (without .yaml).

# Every key is optional: missing colors fall back to the default theme.
# Each color is an adaptive pair — light value for light terminals, dark
# value for dark terminals — as #RGB or #RRGGBB. A file with an invalid
# color is skipped with a log message; a theme can never break the app.

primary: {light: "#5A56E0", dark: "#BD93F9"}   # accent: active pane, tabs, cursor
dim: {light: "#8A8A8A", dark: "#6272A4"}
success: {light: "#008000", dark: "#50FA7B"}   # 2xx status, notices
warn: {light: "#B58900", dark: "#F1FA8C"}      # 4xx status, placeholders
error: {light: "#CC0000", dark: "#FF5555"}     # 5xx status, failures
info: {light: "#0066CC", dark: "#8BE9FD"}      # keys in hints, editor accent
accent: {light: "#F92672", dark: "#FF79C6"}    # env badge pill
key: {light: "#B4530A", dark: "#FFB86C"}        # shortcut keys in hints and the keybindings panel
muted: {light: "#999999", dark: "#6272A4"}     # hint text
border: {light: "#DDDDDD", dark: "#44475A"}    # pane borders when unfocused
input: {light: "#44475A", dark: "#F8F8F2"}     # text color in inputs/editor
field: {light: "#E9E9F0", dark: "#343746"}     # URL bar background
selection: {light: "#5A56E0", dark: "#BD93F9"} # highlighted list row background
on_selection: {light: "#FFFFFF", dark: "#282A36"} # text on the highlighted row

methods:
  GET: {light: "#008000", dark: "#50FA7B"}
  POST: {light: "#B58900", dark: "#F1FA8C"}
  PUT: {light: "#0066CC", dark: "#8BE9FD"}
  PATCH: {light: "#875FD7", dark: "#FF79C6"}
  DELETE: {light: "#CC0000", dark: "#FF5555"}
  HEAD: {light: "#999999", dark: "#6272A4"}
  OPTIONS: {light: "#999999", dark: "#6272A4"}
```
