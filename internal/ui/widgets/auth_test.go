package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Dread-Code/lazypost/internal/collection"
)

func TestAuthEditorRoundTrip(t *testing.T) {
	a := NewAuthEditor()

	// cycle none -> basic
	a.CycleType(1)
	a.username.SetValue("user")
	a.password.SetValue("pass")
	auth := a.Auth()
	if auth.Type != "basic" || auth.Username != "user" || auth.Password != "pass" {
		t.Errorf("basic auth round trip failed: %+v", auth)
	}

	// SetAuth restores the fields
	a.SetAuth(auth)
	if a.username.Value() != "user" || a.password.Value() != "pass" {
		t.Errorf("SetAuth did not restore fields: %q %q", a.username.Value(), a.password.Value())
	}

	// apikey with the key-in toggle
	a.SetAuth(&collection.Auth{Type: "apikey", KeyName: "X-Key", KeyValue: "v", KeyIn: "query"})
	got := a.Auth()
	if got.Type != "apikey" || got.KeyName != "X-Key" || got.KeyValue != "v" || got.KeyIn != "query" {
		t.Errorf("apikey round trip failed: %+v", got)
	}

	// toggling the key-in row flips header -> query
	a.focused = true
	a.field = 2 // apikey keyIn toggle row
	a.Update(tea.KeyMsg{Type: tea.KeySpace})
	if a.Auth().KeyIn != "header" {
		t.Errorf("expected keyIn header after toggle, got %q", a.Auth().KeyIn)
	}
}

func TestAuthEditorNone(t *testing.T) {
	a := NewAuthEditor()
	if a.Auth() != nil {
		t.Errorf("none auth should be nil, got %+v", a.Auth())
	}
	a.CycleType(1) // basic
	a.username.SetValue("u")
	a.SetAuth(nil)
	if a.Auth() != nil {
		t.Errorf("SetAuth(nil) should reset to none, got %+v", a.Auth())
	}
	if a.authType != "none" {
		t.Errorf("expected type none, got %q", a.authType)
	}
}
