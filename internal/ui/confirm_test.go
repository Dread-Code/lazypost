package ui

import (
	"strings"
	"testing"
)

func TestConfirmView(t *testing.T) {
	c := NewConfirm()
	c.Ask("delete request list authors?")
	v := c.View()
	if !strings.Contains(v, "delete request list authors?") {
		t.Errorf("expected question in view, got:\n%s", v)
	}
	if !strings.Contains(v, "y yes · n no") {
		t.Errorf("expected hint in view, got:\n%s", v)
	}
}
