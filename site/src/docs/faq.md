# // faq

Troubleshooting and behavior notes, drawn from real usage.

## My first send fails with "unsupported protocol scheme"

Most likely an unresolved URL placeholder: `{{host}}` with no active environment leaves
escape-encoded braces in the URL. Set an environment (`ctrl+e`) — unresolved URL placeholders fail
loudly and point at the environment switcher.

## Keyboard shortcuts with ctrl+arrows don't work

macOS intercepts `ctrl+←/→` for Mission Control/App Exposé. lazypost uses `alt+←/→` to switch
editor tabs.

## Vim visual selection looks broken after the first token

a known rendering quirk in early versions; the selection renderer strips ANSI codes first in
current builds — update to the latest release.

## Yank vs paste

yank copies to the system clipboard; paste reads the last internal yank/delete (not the system
clipboard).

## My body JSON doesn't format

bodies auto-format (2-space) on blur and on save; invalid JSON or placeholder-only bodies are
never touched (placeholder-aware formatting keeps `{{var}}` intact).

## The response Headers tab is confusing

the top row shows the exact executed URL (with substitutions) — check it if the response doesn't
look like the request you expected.

## --dry-run said it would import but nothing appeared

by design: --dry-run only reports and never creates the target directory.

## ctrl+e cycles my variables while I'm typing

global bindings take precedence over editor defaults by design; type `{{` placeholders manually
or bind around them — the editor powers live on the focused pane's bindings otherwise.

## A saved request shows a different name

the sidebar name comes from the filename; the YAML `name:` field is display-only in the sidebar
label context. Keep filenames descriptive.

## Docs, themes, or sample collections missing?

`./sample-collections` and `./collections` are implicit collections but never auto-created; run
lazypost in a directory and it initializes `config/config.yaml` for you.
