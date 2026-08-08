package main

import (
	"os"
	"path/filepath"
	"testing"
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
