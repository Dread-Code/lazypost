package model

import "github.com/charmbracelet/bubbles/key"

// Root-model keybindings. Single place for the model's routing keys;
// pane/editor bindings live in internal/ui/keys.go. The action registry
// (actions.go) owns the global ctrl shortcuts; these are focus and
// modal keys. Prefer ctrl+letter over modifier+arrow (Gotcha - macOS
// intercepts ctrl+arrow keys).
var (
	keyTab      = key.NewBinding(key.WithKeys("tab"))
	keyShiftTab = key.NewBinding(key.WithKeys("shift+tab"))
	// ctrl+/ is delivered as ctrl+_ (0x1F) by terminals; accept both
	keyPalette = key.NewBinding(key.WithKeys("ctrl+_", "ctrl+/"))
	keyEnter   = key.NewBinding(key.WithKeys("enter"))
	keyEsc     = key.NewBinding(key.WithKeys("esc"))
	keyQuit    = key.NewBinding(key.WithKeys("q"))

	keyUp      = key.NewBinding(key.WithKeys("up"))
	keyDown    = key.NewBinding(key.WithKeys("down"))
	keyCtrlN   = key.NewBinding(key.WithKeys("ctrl+n"))
	keyCtrlP   = key.NewBinding(key.WithKeys("ctrl+p"))
	keyCtrlE   = key.NewBinding(key.WithKeys("ctrl+e"))
	keySlash   = key.NewBinding(key.WithKeys("/"))
	keyHistory = key.NewBinding(key.WithKeys("ctrl+h"))

	keyAdd    = key.NewBinding(key.WithKeys("a")) // sidebar: add request/folder
	keyDelete = key.NewBinding(key.WithKeys("d")) // sidebar/env manager: delete
	keyRename = key.NewBinding(key.WithKeys("r")) // sidebar/env manager: rename
	keyN      = key.NewBinding(key.WithKeys("n")) // sidebar: new request; confirm: no
	keyYes    = key.NewBinding(key.WithKeys("y")) // confirm: yes
)
