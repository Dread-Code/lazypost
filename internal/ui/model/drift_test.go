package model

import (
	"slices"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// globalKeys returns the shortcut keys of the named global action, or nil
// if the action no longer exists.
func globalKeys(title string) []string {
	for _, a := range globalActions {
		if a.Title == title {
			return a.binding.Keys()
		}
	}
	return nil
}

// The status bar and keybindings panel document keys as text, while the
// bindings live in keys.go / the action registry. The panel content is
// static by design ([[Design - keybindings panel]]), so this test is the
// drift guard: every documented string must still be a live binding.
func TestDocumentedKeysMatchBindings(t *testing.T) {
	type check struct {
		doc  string
		keys []string
	}
	checks := []check{
		{"ctrl+/", keyPalette.Keys()},
		{"ctrl+h", keyHistory.Keys()},
		{"?", keyHelp.Keys()},
		{"ctrl+r", globalKeys("Send request")},
		{"ctrl+s", globalKeys("Save request")},
		{"ctrl+e", globalKeys("Cycle environment")},
		{"ctrl+l", globalKeys("Focus URL bar")},
		{"ctrl+g", globalKeys("Copy as curl")},
		{"ctrl+n", keyCtrlN.Keys()},
		{"ctrl+p", keyCtrlP.Keys()},
		{"enter", keyEnter.Keys()},
		{"esc", keyEsc.Keys()},
		{"tab", keyTab.Keys()},
		{"up", keyUp.Keys()},
		{"down", keyDown.Keys()},
	}
	for _, c := range checks {
		if len(c.keys) == 0 {
			t.Errorf("documented %q has no live binding (renamed/removed?)", c.doc)
			continue
		}
		if !slices.Contains(c.keys, c.doc) {
			t.Errorf("documented %q not among binding keys %v", c.doc, c.keys)
		}
	}
}

// Every documented key is actually shown in the panel (so the panel stays
// a faithful reference).
func TestPanelShowsDocumentedKeys(t *testing.T) {
	prev := lipgloss.DefaultRenderer().ColorProfile()
	lipgloss.DefaultRenderer().SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.DefaultRenderer().SetColorProfile(prev) })
	panel := stripAnsiView(helpContent())

	docs := []string{"ctrl+/", "ctrl+h", "?", "ctrl+r", "ctrl+s", "ctrl+e",
		"ctrl+l", "ctrl+g", "ctrl+n/p", "enter", "esc", "tab"}
	for _, doc := range docs {
		if !strings.Contains(panel, doc) {
			t.Errorf("documented key %q missing from the keybindings panel", doc)
		}
	}
}
