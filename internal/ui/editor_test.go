package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"postgo/internal/collection"
)

func TestEditorCarriesHooks(t *testing.T) {
	e := NewEditor(60, 15)
	req := &collection.Request{
		Name:   "thing",
		Method: "GET",
		URL:    "https://api.test/things",
		Pre:    "req.headers['X-Ts'] = os.time()",
		Post:   "return response.status_code == 200",
	}
	e.SetRequest(req, "/col/thing.yaml")
	got := e.Request()
	if got.Pre != req.Pre || got.Post != req.Post {
		t.Errorf("hooks not carried: pre=%q post=%q", got.Pre, got.Post)
	}
	if got.Name != "thing" {
		t.Errorf("name lost: %q", got.Name)
	}

	e.New()
	if e.Request().Pre != "" || e.Request().Post != "" {
		t.Error("New should clear hooks")
	}
}

func TestEditorScriptsTab(t *testing.T) {
	e := NewEditor(60, 20)
	// navigate to the Scripts tab (index 4) via ctrl+n
	e.Focus()
	for i := 0; i < 4; i++ {
		e.Update(tea.KeyMsg{Type: tea.KeyCtrlN})
	}
	if e.section != SecScripts {
		t.Fatalf("expected Scripts section, got %d", e.section)
	}
	if !strings.Contains(e.View(), "Scripts") {
		t.Errorf("Scripts tab not rendered:\n%s", e.View())
	}
	if !strings.Contains(e.View(), "pre") || !strings.Contains(e.View(), "post") {
		t.Errorf("pre/post labels not rendered:\n%s", e.View())
	}

	// focus starts on pre; ctrl+t moves to post, again back to pre
	e.Update(tea.KeyMsg{Type: tea.KeyCtrlT})
	if e.field != 1 {
		t.Errorf("expected field post after ctrl+t, got %d", e.field)
	}
	e.Update(tea.KeyMsg{Type: tea.KeyCtrlT})
	if e.field != 0 {
		t.Errorf("expected field pre after second ctrl+t, got %d", e.field)
	}

	// typing goes to the focused field
	e.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("return true")})
	if got := e.Request().Pre; got != "return true" {
		t.Errorf("expected pre set by typing, got %q", got)
	}

	// arrows move the cursor inside the textarea, not the hook toggle
	e.Update(tea.KeyMsg{Type: tea.KeyUp})
	if e.field != 0 {
		t.Errorf("up arrow should not change the field, got %d", e.field)
	}
}
