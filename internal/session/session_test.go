package session

import (
	"os"
	"path/filepath"
	"sync"
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

func TestSaveConcurrentCollectionsPreservesState(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	const count = 8
	var wg sync.WaitGroup
	for i := 0; i < count; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := Save(filepath.Join("/tmp", "collection", string(rune('a'+i))), State{Env: string(rune('a' + i))}); err != nil {
				t.Errorf("Save(%d): %v", i, err)
			}
		}()
	}
	wg.Wait()

	for i := 0; i < count; i++ {
		dir := filepath.Join("/tmp", "collection", string(rune('a'+i)))
		got, err := Load(dir)
		if err != nil {
			t.Fatalf("Load(%q): %v", dir, err)
		}
		if got.Env != string(rune('a'+i)) {
			t.Errorf("state[%q] = %+v", dir, got)
		}
	}
}

func TestSaveRejectsMalformedExistingState(t *testing.T) {
	cfg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfg)
	if err := Save("/tmp/collection", State{Env: "dev"}); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(cfg, "lazypost", "state.yaml")
	malformed := []byte("not: [valid\n")
	if err := os.WriteFile(file, malformed, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := Save("/tmp/other", State{Env: "prod"}); err == nil {
		t.Fatal("Save malformed state succeeded")
	}
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != string(malformed) {
		t.Fatalf("malformed state was replaced: %q", data)
	}
}

func TestSaveRecoversEmptyStateDocument(t *testing.T) {
	cfg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfg)
	file := filepath.Join(cfg, "lazypost", "state.yaml")
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte("null\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := Save("/tmp/collection", State{Env: "dev"}); err != nil {
		t.Fatalf("Save empty state document: %v", err)
	}
	got, err := Load("/tmp/collection")
	if err != nil {
		t.Fatal(err)
	}
	if got.Env != "dev" {
		t.Fatalf("state = %+v, want dev", got)
	}
}

func TestWriterFlushKeepsNewestSnapshot(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	w := NewWriter()
	if err := w.Save("/tmp/collection", State{Env: "old"}); err != nil {
		t.Fatal(err)
	}
	if err := w.Save("/tmp/collection", State{Env: "new"}); err != nil {
		t.Fatal(err)
	}
	if err := w.Flush("/tmp/collection", State{Env: "final"}); err != nil {
		t.Fatal(err)
	}
	got, err := Load("/tmp/collection")
	if err != nil {
		t.Fatal(err)
	}
	if got.Env != "final" {
		t.Fatalf("state = %+v, want final snapshot", got)
	}
}

func TestSaveUsesRestrictivePermissions(t *testing.T) {
	cfg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfg)
	if err := Save("/tmp/collection", State{Env: "dev"}); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(filepath.Join(cfg, "lazypost", "state.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Errorf("state mode = %o, want 600", got)
	}
}
