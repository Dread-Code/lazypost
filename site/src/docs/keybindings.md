# $ keybindings

Every binding, grouped by pane. Keys render literally — ctrl+l means Ctrl+L.

## global

| Key | Action |
| --- | --- |
| tab | switch panes |
| ctrl+/ | command palette |
| ? | keybindings panel |
| ctrl+h | request history |
| ctrl+r | send request |
| ctrl+e | cycle environment |
| ctrl+s | save request |
| ctrl+l | jump to the URL bar |
| ctrl+g | export the current request as curl |
| q | quit (collection / response / editor normal & visual modes) |
| ctrl+c | quit |

## collection · sidebar

| Key | Action |
| --- | --- |
| ↑/↓, ctrl+n/ctrl+p | navigate (loads the request) |
| enter | focus the URL bar / toggle folder (collection root toggles all) |
| n | new request |
| a | add request in folder; lead with / for a folder |
| d | delete (confirm with y) |
| r | rename |

## url bar

| Key | Action |
| --- | --- |
| ctrl+t | cycle method |
| enter | send |
| esc | back to previous pane |
| paste curl … | import a curl command |

## editor

NORMAL-mode field; mode shown in a colored footer row.

| Key | Action |
| --- | --- |
| i / a / A / I / o / O | enter insert mode (editing) |
| esc | back to NORMAL |
| hjkl / wbe / 0$^ / ggG / % | motions |
| x / dd / dw / d$ / d0 | delete (with counts: d2w, 3dd) |
| yy / yw / y$ | yank (copied to the system clipboard) |
| p / P | paste the last yank/delete |
| v / V | visual selection (char/line); y yanks, d deletes |
| q | quit (normal/visual mode only; in insert it types) |
| ctrl+n/ctrl+p | move between sections |
| alt+←/alt+→ | switch tabs |
| ctrl+t | cycle auth type |
| ctrl+s | save |

## response

| Key | Action |
| --- | --- |
| ←/→, b/h | switch tabs |
| ↑/↓ | scroll |
| q | quit |
