package session

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveLoadRoundTrip(t *testing.T) {
	orig := os.Getenv("XDG_CONFIG_HOME")
	cfg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfg)
	defer os.Setenv("XDG_CONFIG_HOME", orig)

	st := State{Env: "dev", ActivePath: "quotes/random.yaml", Collapsed: []string{"authors"}, EditorSection: 2}
	if err := Save("/tmp/col", st); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := Load("/tmp/col")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Env != "dev" || got.ActivePath != "quotes/random.yaml" {
		t.Errorf("round trip mismatch: %+v", got)
	}
	if len(got.Collapsed) != 1 || got.Collapsed[0] != "authors" {
		t.Errorf("collapsed mismatch: %+v", got.Collapsed)
	}
	if got.EditorSection != 2 {
		t.Errorf("editor section mismatch: %d", got.EditorSection)
	}

	// file lives under $XDG_CONFIG_HOME/lazypost/state.yaml
	file := filepath.Join(cfg, "lazypost", "state.yaml")
	if _, err := os.Stat(file); err != nil {
		t.Errorf("expected state file at %s: %v", file, err)
	}
}

func TestLoadMissingReturnsZero(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	st, err := Load("/nope")
	if err != nil {
		t.Fatalf("Load missing: %v", err)
	}
	if st.Env != "" || st.ActivePath != "" || len(st.Collapsed) != 0 {
		t.Errorf("expected zero state, got %+v", st)
	}
}
