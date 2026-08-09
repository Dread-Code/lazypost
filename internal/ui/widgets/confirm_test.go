package ui

import (
	"strings"
	"testing"
)

func TestConfirmView(t *testing.T) {
	c := NewConfirm()
	c.Ask("delete request list authors?")
	// the question moved to the modal's border title
	if l := c.Label(); !strings.Contains(l, "delete request list authors?") {
		t.Errorf("expected question in label, got %q", l)
	}
	if !strings.Contains(stripAnsiTab(c.View()), "y yes · n no") {
		t.Errorf("expected hint in view, got:\n%s", c.View())
	}
}
