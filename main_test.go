package main

import (
	"os"
	"path/filepath"
	"testing"

	"lazypost/internal/collection"
	"lazypost/internal/session"
	"lazypost/internal/ui/model"
)

func TestResolveRootExplicitWins(t *testing.T) {
	root := resolveRoot("/some/explicit/path")
	if root != "/some/explicit/path" {
		t.Errorf("explicit -dir should win, got %q", root)
	}
}

func TestResolveRootPrefersCollections(t *testing.T) {
	t.Chdir(t.TempDir())
	for _, d := range []string{"sample-collections", "collections"} {
		if err := os.Mkdir(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if got := resolveRoot(""); got != "sample-collections" {
		t.Errorf("expected sample-collections preferred, got %q", got)
	}
	_ = os.Remove("sample-collections")
	if got := resolveRoot(""); got != "collections" {
		t.Errorf("expected collections fallback, got %q", got)
	}
}

func TestResolveRootFallsBackToCurrentDir(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	if got := resolveRoot(""); got != "." {
		t.Errorf("expected current dir fallback, got %q", got)
	}
}

func TestCanonicalRootMakesAbsolute(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	if got := canonicalRoot("."); got != dir {
		t.Errorf("canonicalRoot('.') = %q, want %q", got, dir)
	}
	if got := canonicalRoot(filepath.Join(dir, "sub")); got != filepath.Join(dir, "sub") {
		t.Errorf("canonicalRoot of an absolute path changed it: %q", got)
	}
}

func TestMarkerOptionsWithMarker(t *testing.T) {
	opts := markerOptions("", ".", &collection.Marker{Name: "My API"})
	m := newModelWithOpts(opts)
	if m.CollectionName() != "My API" || m.NeedsMarker() {
		t.Errorf("with marker: got name=%q prompt=%v, want name=My API prompt=false", m.CollectionName(), m.NeedsMarker())
	}
}

func TestMarkerOptionsExplicitDirPrompts(t *testing.T) {
	m := newModelWithOpts(markerOptions("/some/dir", ".", nil))
	if !m.NeedsMarker() {
		t.Error("explicit -dir without marker should prompt")
	}
}

func TestMarkerOptionsCwdFallbackPrompts(t *testing.T) {
	m := newModelWithOpts(markerOptions("", ".", nil))
	if !m.NeedsMarker() {
		t.Error("cwd fallback without marker should prompt")
	}
}

func TestMarkerOptionsImplicitCollectionsNoPrompt(t *testing.T) {
	for _, resolved := range []string{"sample-collections", "collections"} {
		m := newModelWithOpts(markerOptions("", resolved, nil))
		if m.NeedsMarker() {
			t.Errorf("resolved %q should stay implicit (no prompt)", resolved)
		}
	}
}

// newModelWithOpts applies the options to a model just to inspect them.
func newModelWithOpts(opts []model.Option) model.Model {
	return model.New("dir", nil, nil, nil, session.State{}, opts...)
}
