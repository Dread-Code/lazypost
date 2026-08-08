package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNamer(t *testing.T) {
	n := NewNamer()
	n.Open()
	for _, r := range []rune("create post") {
		n.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	if got := n.Value(); got != "create post" {
		t.Errorf("expected typed value, got %q", got)
	}
	if n.IsFolder() {
		t.Error("plain name should not be folder mode")
	}
	// the label moved to the modal's border title
	if l := n.Label(); !strings.Contains(l, "new request name") {
		t.Errorf("expected request label, got %q", l)
	}
	if !strings.Contains(n.View(), "create post") {
		t.Errorf("expected typed value in view, got:\n%s", n.View())
	}
}

func TestNamerFolderMode(t *testing.T) {
	n := NewNamer()
	n.Open()
	for _, r := range []rune("/v2") {
		n.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	if !n.IsFolder() {
		t.Error("leading / should enable folder mode")
	}
	if got := n.Value(); got != "v2" {
		t.Errorf("expected value without slash, got %q", got)
	}
	// the label moved to the modal's border title
	if l := n.Label(); !strings.Contains(l, "new collection name") {
		t.Errorf("expected collection label, got %q", l)
	}
}
