package ui

import "github.com/charmbracelet/bubbles/key"

var (
	keySectionNext = key.NewBinding(key.WithKeys("ctrl+n"))
	keySectionPrev = key.NewBinding(key.WithKeys("ctrl+p"))
	keyAltLeft     = key.NewBinding(key.WithKeys("alt+left"))
	keyAltRight    = key.NewBinding(key.WithKeys("alt+right"))
	keyCtrlT       = key.NewBinding(key.WithKeys("ctrl+t"))
)
