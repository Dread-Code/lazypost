package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Dread-Code/lazypost/internal/collection"
	"github.com/Dread-Code/lazypost/internal/session"
	"github.com/Dread-Code/lazypost/internal/ui/model"
	"github.com/Dread-Code/lazypost/internal/ui/themes"
)

var version = "dev"

func main() {
	if len(os.Args) > 1 && os.Args[1] == "import" {
		if err := runImport(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "lazypost import: %v\n", err)
			os.Exit(2)
		}
		return
	}

	dir := flag.String("dir", "", "collection directory (default: ./sample-collections, ./collections, or the current directory)")
	ver := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *ver {
		fmt.Printf("lazypost %s\n", version)
		return
	}

	resolved := resolveRoot(*dir)
	root := canonicalRoot(resolved)

	// Resolve the collection marker. Legacy .lazypost markers are kept alive
	// for this session; their paths are migrated on the first collection
	// write, after which name/root are intentionally discarded.
	marker, err := collection.LoadMarker(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "lazypost: cannot read marker in %q: %v\n", root, err)
		os.Exit(1)
	}
	var legacyPaths []string
	if marker != nil && marker.Legacy {
		legacyPaths = append(legacyPaths, marker.LegacyPath)
	}
	if marker != nil && marker.Root != "" {
		root = canonicalRoot(marker.Root)
		if redirected, err := collection.LoadMarker(root); err != nil {
			fmt.Fprintf(os.Stderr, "lazypost: cannot read marker in %q: %v\n", root, err)
			os.Exit(1)
		} else {
			marker = redirected
			if marker != nil && marker.Legacy {
				legacyPaths = append(legacyPaths, marker.LegacyPath)
			}
		}
	}

	// Explicit directories and the cwd fallback become collections on first
	// open. The repository's implicit sample/collections roots remain
	// markerless so launching lazypost does not dirty the checkout.
	if marker, err = initializeCollection(*dir, resolved, root, marker); err != nil {
		fmt.Fprintf(os.Stderr, "lazypost: cannot initialize collection %q: %v\n", root, err)
		os.Exit(1)
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
	// Load user themes before applying the session theme, so a user
	// theme persisted in state resolves at startup. A missing themes dir
	// or an unreadable config dir is never fatal.
	if dir, err := session.ConfigDir(); err == nil {
		_, _ = themes.LoadUserThemes(filepath.Join(dir, "themes"))
	} else {
		log.Printf("lazypost: cannot locate config dir: %v", err)
	}
	opts := markerOptions(marker, legacyPaths)
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

// shouldInitializeCollection limits automatic marker creation to roots the
// user explicitly selected or the cwd fallback. Preferred repository roots
// remain implicit collections.
func shouldInitializeCollection(dir, resolved string) bool {
	return dir != "" || resolved == "."
}

func initializeCollection(dir, resolved, root string, marker *collection.Marker) (*collection.Marker, error) {
	if marker != nil || !shouldInitializeCollection(dir, resolved) {
		return marker, nil
	}
	if err := collection.WriteMarker(root); err != nil {
		return nil, err
	}
	return collection.LoadMarker(root)
}

// markerOptions preserves legacy display names for the current session and
// carries legacy marker paths to the first write for migration.
func markerOptions(marker *collection.Marker, legacyPaths []string) []model.Option {
	var opts []model.Option
	if marker != nil && marker.Legacy && marker.Name != "" {
		opts = append(opts, model.WithCollectionName(marker.Name))
	}
	if len(legacyPaths) > 0 {
		opts = append(opts, model.WithLegacyMarkers(legacyPaths...))
	}
	return opts
}
