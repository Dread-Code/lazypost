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
	if !strings.Contains(stripAnsi(c.View()), "y yes · n no") {
		t.Errorf("expected hint in view, got:\n%s", c.View())
	}
}

// stripAnsi removes ANSI sequences so assertions work on visible text.
func stripAnsi(s string) string {
	var b strings.Builder
	in := false
	for _, r := range s {
		if r == 0x1b {
			in = true
			continue
		}
		if in {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				in = false
			}
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}
