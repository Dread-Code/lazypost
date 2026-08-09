package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"

	"lazypost/internal/collection"
	"lazypost/internal/session"
	"lazypost/internal/ui/model"
	"lazypost/internal/ui/widgets"
)

var version = "dev"

func main() {
	dir := flag.String("dir", "", "collection directory (default: ./sample-collections, ./collections, or the current directory)")
	ver := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *ver {
		fmt.Printf("lazypost %s\n", version)
		return
	}

	root := resolveRoot(*dir)
	root = canonicalRoot(root)

	// Load the collection tree and environments once, up front; the
	// model treats them as immutable snapshots.
	entries, err := collection.Load(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "lazypost: cannot load collection %q: %v\n", root, err)
		os.Exit(1)
	}
	envs, envNames, err := collection.LoadEnvironments(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "lazypost: cannot load environments: %v\n", err)
		os.Exit(1)
	}

	// Restore UI state (env, last request, collapsed folders, theme) from
	// the previous run; a missing state file yields defaults.
	st, err := session.Load(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "lazypost: cannot load session state: %v\n", err)
		os.Exit(1)
	}
	ui.ThemeByName(st.Theme).Apply()

	p := tea.NewProgram(model.New(root, entries, envs, envNames, st), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "lazypost: %v\n", err)
		os.Exit(1)
	}
}

// resolveRoot picks the collection directory. An explicit -dir wins; a
// bare launch prefers ./sample-collections or ./collections (the repo's
// own layout), otherwise the current directory is the collection — even
// when it is empty ([[Design - open the current directory as a
// collection]]).
func resolveRoot(dir string) string {
	if dir != "" {
		return dir
	}
	for _, d := range []string{"sample-collections", "collections"} {
		if info, err := os.Stat(d); err == nil && info.IsDir() {
			return d
		}
	}
	return "."
}

// canonicalRoot makes the root absolute so session state (keyed by the
// cleaned root path) is unique per collection, even when it was opened
// via the current directory.
func canonicalRoot(root string) string {
	if abs, err := filepath.Abs(root); err == nil {
		return abs
	}
	return root
}
