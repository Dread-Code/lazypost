package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestPaletteItemsAndSelection(t *testing.T) {
	p := NewPalette(40, 10)
	p.SetItems([]PaletteItem{
		{Title: "Send request", Shortcut: "ctrl+r"},
		{Title: "Save request", Shortcut: "ctrl+s"},
		{Title: "Switch theme", Shortcut: ""},
	})
	p.OpenBrowsing()

	if it := p.Selected(); it == nil || it.Title != "Send request" {
		t.Errorf("initial selection = %v", it)
	}
	p.CursorDown()
	if it := p.Selected(); it == nil || it.Title != "Save request" {
		t.Errorf("after down selection = %v", it)
	}
	p.CursorUp()
	if it := p.Selected(); it == nil || it.Title != "Send request" {
		t.Errorf("after up selection = %v", it)
	}
	// the list Update also navigates
	p.Update(tea.KeyMsg{Type: tea.KeyDown})
	if it := p.Selected(); it == nil || it.Title != "Save request" {
		t.Errorf("after Update down selection = %v", it)
	}
}

func TestPaletteRendersRows(t *testing.T) {
	p := NewPalette(40, 10)
	p.SetItems([]PaletteItem{
		{Title: "Send request", Shortcut: "ctrl+r"},
		{Title: "Switch theme", Shortcut: ""},
	})
	out := p.View()
	for _, want := range []string{"Send request", "ctrl+r", "Switch theme"} {
		if !strings.Contains(out, want) {
			t.Errorf("palette view lost %q:\n%s", want, out)
		}
	}
}

// Regression: a shortcut wider than the row must not produce a negative
// strings.Repeat count (the same panic fixed in the history delegate).
func TestPaletteWideShortcutDoesNotPanic(t *testing.T) {
	p := NewPalette(20, 6)
	p.SetItems([]PaletteItem{
		{Title: "x", Shortcut: "a much wider shortcut than the row"},
	})
	_ = p.View() // must not panic
}
