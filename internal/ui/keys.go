package ui

import "github.com/charmbracelet/bubbles/key"

var (
	keyCtrlDown = key.NewBinding(key.WithKeys("ctrl+down"))
	keyCtrlUp   = key.NewBinding(key.WithKeys("ctrl+up"))
	keyAltLeft  = key.NewBinding(key.WithKeys("alt+left"))
	keyAltRight = key.NewBinding(key.WithKeys("alt+right"))
	keyCtrlT    = key.NewBinding(key.WithKeys("ctrl+t"))
)
