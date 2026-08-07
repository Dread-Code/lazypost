package ui

import (
	"strings"
	"testing"

	"postgo/internal/collection"
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
