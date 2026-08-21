package ui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Dread-Code/lazypost/internal/app"
	"github.com/Dread-Code/lazypost/internal/collection"
)

func newReq(name, method, url string) collection.Request {
	return collection.Request{Name: name, Method: method, URL: url}
}

func TestHistoryRendersRows(t *testing.T) {
	h := NewHistory(40, 10)
	h.SetItems([]app.HistoryEntry{
		{Req: newReq("list authors", "GET", "https://api.test/authors"), Summary: "200 OK · 1.2 KiB · 12ms", At: time.Unix(0, 0)},
		{Req: newReq("", "POST", "https://api.test/login"), Summary: "401 · 0 B · 3ms", At: time.Unix(1, 0)},
	})
	out := h.View()
	if !strings.Contains(out, "list authors") {
		t.Errorf("named request missing:\n%s", out)
	}
	if !strings.Contains(out, "200 OK") {
		t.Errorf("summary missing:\n%s", out)
	}
	// unnamed requests fall back to METHOD url (row is width-truncated)
	if !strings.Contains(out, "POST https://api.tes") {
		t.Errorf("unnamed request fallback missing:\n%s", out)
	}
}

func TestHistoryNavigation(t *testing.T) {
	h := NewHistory(40, 10)
	h.SetItems([]app.HistoryEntry{
		{Req: newReq("a", "GET", "https://api.test/a"), Summary: "200 OK", At: time.Unix(0, 0)},
		{Req: newReq("b", "GET", "https://api.test/b"), Summary: "200 OK", At: time.Unix(1, 0)},
	})
	h.Open()
	if it := h.Selected(); it == nil || it.HistoryTitle() != "a" {
		t.Errorf("initial selection = %v", it)
	}
	h.Update(tea.KeyMsg{Type: tea.KeyDown})
	if it := h.Selected(); it == nil || it.HistoryTitle() != "b" {
		t.Errorf("after down selection = %v", it)
	}
	h.Update(tea.KeyMsg{Type: tea.KeyCtrlP})
	if it := h.Selected(); it == nil || it.HistoryTitle() != "a" {
		t.Errorf("after ctrl+p selection = %v", it)
	}
}
