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
	if !strings.Contains(n.View(), "new request name") {
		t.Errorf("expected label in view, got:\n%s", n.View())
	}
}
