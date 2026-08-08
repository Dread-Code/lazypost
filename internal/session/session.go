// Package session persists UI state between runs: the active environment,
// the last selected request, and collapsed directories. It is a UI
// convenience, never a source of truth — requests live in the collection
// tree ([[Design - persist session state]]).
package session

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

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
	file, err := stateFile()
	if err != nil {
		return err
	}
	all := map[string]State{}
	if data, err := os.ReadFile(file); err == nil {
		_ = yaml.Unmarshal(data, &all)
	}
	all[filepath.Clean(dir)] = s
	data, err := yaml.Marshal(all)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		return err
	}
	return os.WriteFile(file, data, 0o644)
}

// stateFile returns $XDG_CONFIG_HOME/lazypost/state.yaml (or
// ~/.config/lazypost/state.yaml), falling back to the platform config dir.
func stateFile() (string, error) {
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return filepath.Join(dir, "lazypost", "state.yaml"), nil
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "lazypost", "state.yaml"), nil
}
