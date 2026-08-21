// Package session persists UI state between runs: the active environment,
// the last selected request, and collapsed directories. It is a UI
// convenience, never a source of truth — requests live in the collection
// tree ([[Design - persist session state]]).
package session

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/yaml.v3"
)

var saveMu sync.Mutex

// Writer serializes session snapshots and drops stale snapshots that have
// not started writing yet. It is shared by a Model's asynchronous save
// commands and makes quit a durable latest-state barrier.
type Writer struct {
	mu     sync.Mutex
	latest uint64
}

// NewWriter returns a session writer for one application lifecycle.
func NewWriter() *Writer { return &Writer{} }

// Reserve assigns a revision synchronously, before an asynchronous command
// is scheduled. This keeps submission order independent of command execution
// order.
func (w *Writer) Reserve() uint64 {
	w.mu.Lock()
	w.latest++
	revision := w.latest
	w.mu.Unlock()
	return revision
}

func (w *Writer) isLatest(revision uint64) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return revision == w.latest
}

// Save submits a snapshot. If a newer snapshot has already been submitted,
// this call becomes a no-op instead of allowing stale state to overwrite it.
func (w *Writer) Save(dir string, s State) error {
	return w.SaveRevision(w.Reserve(), dir, s)
}

// SaveRevision writes a snapshot only when revision is still current.
func (w *Writer) SaveRevision(revision uint64, dir string, s State) error {
	saveMu.Lock()
	defer saveMu.Unlock()
	if !w.isLatest(revision) {
		return nil
	}
	return saveLocked(dir, s)
}

// Flush synchronously writes the newest snapshot and prevents older queued
// saves from writing after the caller's lifecycle barrier.
func (w *Writer) Flush(dir string, s State) error {
	revision := w.Reserve()
	saveMu.Lock()
	defer saveMu.Unlock()
	if !w.isLatest(revision) {
		return nil
	}
	return saveLocked(dir, s)
}

// State is what lazypost remembers about one collection between runs. All
// paths are relative to the collection root.
type State struct {
	Env           string   `yaml:"env,omitempty"`
	ActivePath    string   `yaml:"active,omitempty"`
	Collapsed     []string `yaml:"collapsed,omitempty"`
	Theme         string   `yaml:"theme,omitempty"`          // theme name, "" = default
	EditorSection int      `yaml:"editor_section,omitempty"` // active editor tab
}

// Load reads the state for dir. A missing file or unknown keys yield a
// zero State.
func Load(dir string) (State, error) {
	file, err := stateFile()
	if err != nil {
		return State{}, err
	}
	data, err := os.ReadFile(file)
	if err != nil {
		if os.IsNotExist(err) {
			return State{}, nil
		}
		return State{}, err
	}
	var all map[string]State
	if err := yaml.Unmarshal(data, &all); err != nil {
		return State{}, err
	}
	return all[filepath.Clean(dir)], nil
}

// Save writes dir's state, preserving any other collections' entries.
func Save(dir string, s State) error {
	saveMu.Lock()
	defer saveMu.Unlock()
	return saveLocked(dir, s)
}

func saveLocked(dir string, s State) error {
	file, err := stateFile()
	if err != nil {
		return err
	}
	all := map[string]State{}
	if data, err := os.ReadFile(file); err == nil {
		if err := yaml.Unmarshal(data, &all); err != nil {
			return fmt.Errorf("parsing %s: %w", file, err)
		}
		if all == nil {
			all = map[string]State{}
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	all[filepath.Clean(dir)] = s
	data, err := yaml.Marshal(all)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		return err
	}
	return writeAtomic(file, data, 0o600)
}

func writeAtomic(path string, data []byte, mode os.FileMode) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".tmp-")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	cleanup := func() {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
	}
	if err := tmp.Chmod(mode); err != nil {
		cleanup()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		cleanup()
		return err
	}
	if err := tmp.Sync(); err != nil {
		cleanup()
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	defer os.Remove(tmpPath)
	return os.Rename(tmpPath, path)
}

// ConfigDir returns the lazypost config directory:
// $XDG_CONFIG_HOME/lazypost (or ~/.config/lazypost), falling back to the
// platform config dir.
func ConfigDir() (string, error) {
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return filepath.Join(dir, "lazypost"), nil
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "lazypost"), nil
}

// stateFile returns $XDG_CONFIG_HOME/lazypost/state.yaml (or
// ~/.config/lazypost/state.yaml), falling back to the platform config dir.
func stateFile() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "state.yaml"), nil
}
