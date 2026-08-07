package ui

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"postgo/internal/collection"
)

type item struct {
	entry collection.Entry
}

func (i item) FilterValue() string { return i.entry.Name }

type delegate struct{}

func (d delegate) Height() int  { return 1 }
func (d delegate) Spacing() int { return 0 }

func (d delegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d delegate) Render(w io.Writer, m list.Model, index int, li list.Item) {
	it, ok := li.(item)
	if !ok {
		return
	}
	e := it.entry
	selected := index == m.Index()

	// Budget: cursor(1) + space(1) + indent + content must fit the list
	// width or lipgloss will wrap the line and break the layout.
	avail := m.Width() - 2 - e.Depth*2
	if avail < 4 {
		avail = 4
	}

	var line string
	switch e.Kind {
	case collection.Dir:
		// the collection root renders as a plain bold label without the
		// trailing slash; nested dirs keep the muted name/ look
		if e.Depth < 0 {
			name := lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary).Render(TruncateRunes(e.Name, avail))
			if !selected {
				name = lipgloss.NewStyle().Bold(true).Foreground(ColorMuted).Render(TruncateRunes(e.Name, avail))
			}
			line = name
		} else {
			name := lipgloss.NewStyle().Bold(true).Foreground(ColorMuted).Render(TruncateRunes(e.Name+"/", avail))
			line = name
		}
	default:
		// method badge(7) + space(1)
		nameW := avail - 8
		if nameW < 3 {
			nameW = 3
		}
		method := MethodStyle(e.Req.Method).Render(pad(e.Req.Method, 7))
		name := TruncateRunes(e.Name, nameW)
		if selected {
			name = lipgloss.NewStyle().Foreground(ColorPrimary).Render(name)
		}
		line = method + " " + name
	}

	indent := ""
	if e.Depth > 0 {
		indent = strings.Repeat("  ", e.Depth)
	}
	cursor := " "
	if selected {
		cursor = lipgloss.NewStyle().Foreground(ColorPrimary).Render("▸")
	}
	fmt.Fprint(w, cursor+indent+line)
}

func pad(s string, n int) string {
	if len(s) >= n {
		return s
	}
	return s + strings.Repeat(" ", n-len(s))
}

type Sidebar struct {
	list      list.Model
	entries   []collection.Entry
	collapsed map[string]bool
	root      collection.Entry
}

// NewSidebar builds a dumb tree view: filtering, help, pagination, status
// bar, and quit keys are all disabled — the collection pane only
// navigates and selects. Directories can be collapsed with enter; the set
// of collapsed paths starts empty. The collection root is a synthetic top
// entry whose enter collapses/expands every directory at once.
func NewSidebar(entries []collection.Entry, root string, width, height int) *Sidebar {
	s := &Sidebar{
		entries:   entries,
		collapsed: make(map[string]bool),
		root: collection.Entry{
			Kind: collection.Dir, Name: filepath.Base(root), Depth: -1, Path: root,
		},
	}
	s.list = list.New(s.items(), delegate{}, width, height)
	s.list.SetShowTitle(false)
	s.list.SetShowFilter(false)
	s.list.SetShowHelp(false)
	s.list.SetShowPagination(false)
	s.list.SetShowStatusBar(false)
	s.list.SetFilteringEnabled(false)
	s.list.DisableQuitKeybindings()
	s.list.Styles.NoItems = HintStyle
	return s
}

// items returns the visible entries as list items: the collection root
// followed by anything not under a collapsed directory, so collapsing
// hides a whole subtree.
func (s *Sidebar) items() []list.Item {
	items := make([]list.Item, 0, len(s.entries)+1)
	items = append(items, item{entry: s.root})
	for _, e := range s.entries {
		if s.hidden(e) {
			continue
		}
		items = append(items, item{entry: e})
	}
	return items
}

// hidden reports whether e lives under a collapsed directory. Dirs
// themselves stay visible so they can be re-expanded.
func (s *Sidebar) hidden(e collection.Entry) bool {
	for p := range s.collapsed {
		if p != e.Path && strings.HasPrefix(e.Path, p+"/") {
			return true
		}
	}
	return false
}

// ToggleCollapsed collapses/expands the directory under the cursor. On the
// collection root it toggles every directory at once; otherwise just the
// selected one. It returns false when the cursor isn't on a directory
// (caller treats that as "load request").
func (s *Sidebar) ToggleCollapsed() bool {
	it, ok := s.list.SelectedItem().(item)
	if !ok || it.entry.Kind != collection.Dir {
		return false
	}

	// Collection root: collapse (or re-expand) every directory.
	if it.entry.Depth < 0 {
		all := true
		for _, e := range s.entries {
			if e.Kind == collection.Dir && !s.collapsed[e.Path] {
				all = false
				break
			}
		}
		for _, e := range s.entries {
			if e.Kind == collection.Dir {
				if all {
					delete(s.collapsed, e.Path)
				} else {
					s.collapsed[e.Path] = true
				}
			}
		}
		_ = s.list.SetItems(s.items())
		return true
	}

	path := it.entry.Path
	s.collapsed[path] = !s.collapsed[path]
	if !s.collapsed[path] {
		delete(s.collapsed, path)
	}
	_ = s.list.SetItems(s.items())
	return true
}

func (s *Sidebar) SetEntries(entries []collection.Entry) {
	s.entries = entries
	_ = s.list.SetItems(s.items())
}

// Selected returns the currently highlighted request entry, or nil when
// the cursor is on a directory or nothing is selected.
func (s *Sidebar) Selected() *collection.Entry {
	it, ok := s.list.SelectedItem().(item)
	if !ok || it.entry.Kind != collection.Req {
		return nil
	}
	return &it.entry
}

func (s *Sidebar) Resize(width, height int) {
	s.list.SetSize(width, height)
}

func (s *Sidebar) Update(msg tea.Msg) (*Sidebar, tea.Cmd) {
	var cmd tea.Cmd
	s.list, cmd = s.list.Update(msg)
	return s, cmd
}

func (s *Sidebar) View() string {
	return s.list.View()
}
