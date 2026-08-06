package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"postgo/internal/collection"
	"postgo/internal/model"
)

func main() {
	dir := flag.String("dir", "", "collection directory (default: ./sample-collections or ./collections)")
	flag.Parse()

	root := *dir
	if root == "" {
		root = defaultDir()
	}

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

	p := tea.NewProgram(model.New(root, entries, envs, envNames), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "postgo: %v\n", err)
		os.Exit(1)
	}
}

func defaultDir() string {
	for _, d := range []string{"sample-collections", "collections"} {
		if info, err := os.Stat(d); err == nil && info.IsDir() {
			return d
		}
	}
	return "collections"
}
