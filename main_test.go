package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Dread-Code/lazypost/internal/collection"
	"github.com/Dread-Code/lazypost/internal/session"
	"github.com/Dread-Code/lazypost/internal/ui/model"
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

func TestMarkerOptionsWithLegacyMarker(t *testing.T) {
	opts := markerOptions(
		&collection.Marker{Name: "My API", Legacy: true},
		[]string{"/tmp/.lazypost"},
	)
	m := newModelWithOpts(opts)
	if m.CollectionName() != "My API" {
		t.Errorf("with legacy marker: got name=%q, want My API", m.CollectionName())
	}
}

func TestMarkerOptionsWithoutMarkerDoesNotPrompt(t *testing.T) {
	m := newModelWithOpts(markerOptions(nil, nil))
	if m.CollectionName() != "" {
		t.Errorf("without marker: got legacy name %q", m.CollectionName())
	}
}

func TestShouldInitializeCollection(t *testing.T) {
	if !shouldInitializeCollection("/some/dir", ".") {
		t.Error("explicit -dir should initialize")
	}
	if !shouldInitializeCollection("", ".") {
		t.Error("cwd fallback should initialize")
	}
	for _, resolved := range []string{"sample-collections", "collections"} {
		if shouldInitializeCollection("", resolved) {
			t.Errorf("resolved %q should stay implicit", resolved)
		}
	}
}

func TestInitializeCollectionCreatesConfigMarker(t *testing.T) {
	root := t.TempDir()
	marker, err := initializeCollection("/explicit", "ignored", root, nil)
	if err != nil {
		t.Fatalf("initializeCollection: %v", err)
	}
	if marker == nil || marker.Legacy || marker.Version != 1 {
		t.Fatalf("marker = %+v, want new versioned marker", marker)
	}
	if _, err := os.Stat(filepath.Join(root, collection.ConfigDir, collection.ConfigFile)); err != nil {
		t.Fatalf("config marker missing: %v", err)
	}
}

func TestInitializeCollectionLeavesImplicitRootUnchanged(t *testing.T) {
	root := t.TempDir()
	marker, err := initializeCollection("", "sample-collections", root, nil)
	if err != nil {
		t.Fatalf("initializeCollection: %v", err)
	}
	if marker != nil {
		t.Fatalf("marker = %+v, want nil for implicit root", marker)
	}
	if _, err := os.Stat(filepath.Join(root, collection.ConfigDir)); !os.IsNotExist(err) {
		t.Fatalf("implicit root was initialized: %v", err)
	}
}

// newModelWithOpts applies the options to a model just to inspect them.
func newModelWithOpts(opts []model.Option) model.Model {
	return model.New("dir", nil, nil, nil, session.State{}, opts...)
}
