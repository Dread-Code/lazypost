package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"postgo/internal/collection"
	"postgo/internal/session"
	"postgo/internal/ui"
	"postgo/internal/ui/model"
)

func main() {
	dir := flag.String("dir", "", "collection directory (default: ./sample-collections or ./collections)")
	flag.Parse()

	root := *dir
	if root == "" {
		root = defaultDir()
	}

	// Load the collection tree and environments once, up front; the
	// model treats them as immutable snapshots.
	entries, err := collection.Load(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "postgo: cannot load collection %q: %v\n", root, err)
		os.Exit(1)
	}
	envs, envNames, err := collection.LoadEnvironments(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "postgo: cannot load environments: %v\n", err)
		os.Exit(1)
	}

	// Restore UI state (env, last request, collapsed folders, theme) from
	// the previous run; a missing state file yields defaults.
	st, err := session.Load(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "postgo: cannot load session state: %v\n", err)
		os.Exit(1)
	}
	ui.ThemeByName(st.Theme).Apply()

	p := tea.NewProgram(model.New(root, entries, envs, envNames, st), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "postgo: %v\n", err)
		os.Exit(1)
	}
}

// defaultDir picks the first existing collection directory; falls back
// to "collections" so a fresh clone still starts cleanly.
func defaultDir() string {
	for _, d := range []string{"sample-collections", "collections"} {
		if info, err := os.Stat(d); err == nil && info.IsDir() {
			return d
		}
	}
	return "collections"
}
