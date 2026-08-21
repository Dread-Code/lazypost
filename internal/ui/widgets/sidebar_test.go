package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Dread-Code/lazypost/internal/collection"
)

func TestSidebarRootCollapse(t *testing.T) {
	entries := []collection.Entry{
		{Kind: collection.Dir, Name: "authors", Depth: 0, Path: "col/authors"},
		{Kind: collection.Req, Name: "list", Depth: 1, Path: "col/authors/list.yaml", Req: &collection.Request{Method: "GET"}},
		{Kind: collection.Req, Name: "detail", Depth: 1, Path: "col/authors/detail.yaml", Req: &collection.Request{Method: "GET"}},
		{Kind: collection.Dir, Name: "quotes", Depth: 0, Path: "col/quotes"},
		{Kind: collection.Req, Name: "today", Depth: 1, Path: "col/quotes/today.yaml", Req: &collection.Request{Method: "GET"}},
	}
	s := NewSidebar(entries, "col", 40, 20)

	view := func() string { return s.View() }

	// cursor starts on the collection root; enter collapses every folder
	if !s.ToggleCollapsed() {
		t.Fatal("expected enter on the collection root to toggle")
	}
	if got := view(); strings.Contains(got, "list") || strings.Contains(got, "detail") || strings.Contains(got, "today") {
		t.Errorf("expected all request subtrees collapsed, got:\n%s", got)
	}
	if !strings.Contains(view(), "col") || !strings.Contains(view(), "authors") || !strings.Contains(view(), "quotes") {
		t.Errorf("collapsed folders should stay visible:\n%s", view())
	}

	// enter again re-expands everything
	if !s.ToggleCollapsed() {
		t.Fatal("expected second enter on the root to expand")
	}
	if got := view(); !strings.Contains(got, "authors") || !strings.Contains(got, "today") {
		t.Errorf("expected all folders back after re-expand:\n%s", got)
	}
}

func TestSidebarToggleCollapse(t *testing.T) {
	entries := []collection.Entry{
		{Kind: collection.Dir, Name: "authors", Depth: 0, Path: "col/authors"},
		{Kind: collection.Req, Name: "list", Depth: 1, Path: "col/authors/list.yaml", Req: &collection.Request{Method: "GET"}},
		{Kind: collection.Req, Name: "detail", Depth: 1, Path: "col/authors/detail.yaml", Req: &collection.Request{Method: "GET"}},
		{Kind: collection.Dir, Name: "quotes", Depth: 0, Path: "col/quotes"},
		{Kind: collection.Req, Name: "today", Depth: 1, Path: "col/quotes/today.yaml", Req: &collection.Request{Method: "GET"}},
	}
	s := NewSidebar(entries, "col", 40, 20)

	view := func() string { return s.View() }

	// move cursor off the root, onto the first dir; enter collapses its subtree
	s.list.Select(1)
	if !s.ToggleCollapsed() {
		t.Fatal("expected enter on a dir to toggle")
	}
	if got := view(); strings.Contains(got, "list") || strings.Contains(got, "detail") {
		t.Errorf("collapsed authors subtree still visible:\n%s", got)
	}
	if !strings.Contains(view(), "authors") {
		t.Errorf("collapsed dir itself should stay visible:\n%s", view())
	}

	// enter again re-expands
	if !s.ToggleCollapsed() {
		t.Fatal("expected second enter to toggle back")
	}
	if got := view(); !strings.Contains(got, "list") || !strings.Contains(got, "detail") {
		t.Errorf("expected authors subtree back after re-expand:\n%s", got)
	}
}

func TestSidebarCtrlNPnMovesCursor(t *testing.T) {
	entries := []collection.Entry{
		{Kind: collection.Dir, Name: "authors", Depth: 0, Path: "col/authors"},
		{Kind: collection.Req, Name: "list", Depth: 1, Path: "col/authors/list.yaml", Req: &collection.Request{Method: "GET"}},
		{Kind: collection.Req, Name: "detail", Depth: 1, Path: "col/authors/detail.yaml", Req: &collection.Request{Method: "GET"}},
	}
	s := NewSidebar(entries, "col", 40, 20)

	// cursor starts on the collection root (index 0)
	if e := s.Selected(); e != nil {
		t.Fatalf("expected root selection (no request), got %q", e.Name)
	}

	// ctrl+n moves down: first onto the authors dir (not a request)
	s.Update(tea.KeyMsg{Type: tea.KeyCtrlN})
	if s.Selected() != nil {
		t.Errorf("after ctrl+n expected the authors dir, got %v", s.Selected())
	}

	// next ctrl+n lands on the first request
	s.Update(tea.KeyMsg{Type: tea.KeyCtrlN})
	if e := s.Selected(); e == nil || e.Name != "list" {
		t.Errorf("after second ctrl+n selected %v, want list", e)
	}

	// ctrl+n again moves further down
	s.Update(tea.KeyMsg{Type: tea.KeyCtrlN})
	if e := s.Selected(); e == nil || e.Name != "detail" {
		t.Errorf("after third ctrl+n selected %v, want detail", e)
	}

	// ctrl+p moves back up
	s.Update(tea.KeyMsg{Type: tea.KeyCtrlP})
	if e := s.Selected(); e == nil || e.Name != "list" {
		t.Errorf("after ctrl+p selected %v, want list", e)
	}

	// arrows still work alongside
	s.Update(tea.KeyMsg{Type: tea.KeyUp})
	if s.Selected() != nil {
		t.Errorf("after up selected %v, want the authors dir", s.Selected())
	}
}
