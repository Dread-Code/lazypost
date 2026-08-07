package ui

import "github.com/charmbracelet/bubbles/key"

// Shared editor keybindings. ctrl+arrows were dropped because macOS
// intercepts them (see vault: Gotcha - macOS intercepts ctrl+arrow keys),
// and alt+arrows are reserved for tab switching (Gotcha - ctrl arrows
// collide with textinput word navigation).
var (
	keySectionNext = key.NewBinding(key.WithKeys("ctrl+n"))
	keySectionPrev = key.NewBinding(key.WithKeys("ctrl+p"))
	keyAltLeft     = key.NewBinding(key.WithKeys("alt+left"))
	keyAltRight    = key.NewBinding(key.WithKeys("alt+right"))
	keyCtrlT       = key.NewBinding(key.WithKeys("ctrl+t"))
)
