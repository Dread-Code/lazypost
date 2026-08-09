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

	resolved := resolveRoot(*dir)
	root := canonicalRoot(resolved)

	// Resolve the .lazypost marker: a marker with root set points at the
	// real collection elsewhere ([[Design - collection marker file]]).
	marker, err := collection.LoadMarker(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "lazypost: cannot read marker in %q: %v\n", root, err)
		os.Exit(1)
	}
	if marker != nil && marker.Root != "" {
		root = canonicalRoot(marker.Root)
		if redirected, err := collection.LoadMarker(root); err != nil {
			fmt.Fprintf(os.Stderr, "lazypost: cannot read marker in %q: %v\n", root, err)
			os.Exit(1)
		} else {
			marker = redirected
		}
	}

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

	opts := markerOptions(*dir, resolved, marker)
	opts = append(opts, model.WithVersion(version))

	p := tea.NewProgram(model.New(root, entries, envs, envNames, st, opts...), tea.WithAltScreen())
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

// markerOptions decides how the model opens the collection. A present
// marker supplies its name; a missing marker on a user-chosen directory
// (-dir or the cwd fallback) prompts for a name on first run. The
// auto-detected sample-collections/collections stay implicit.
func markerOptions(dir, resolved string, marker *collection.Marker) []model.Option {
	if marker != nil {
		return []model.Option{model.WithCollectionName(marker.Name)}
	}
	if dir != "" || resolved == "." {
		return []model.Option{model.WithMarkerPrompt()}
	}
	return nil
}
