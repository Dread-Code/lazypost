package ui

import (
	"fmt"
	"io"
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
		name := lipgloss.NewStyle().Bold(true).Foreground(ColorMuted).Render(TruncateRunes(e.Name+"/", avail))
		line = name
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

	indent := strings.Repeat("  ", e.Depth)
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
	list    list.Model
	entries []collection.Entry
}

func NewSidebar(entries []collection.Entry, width, height int) *Sidebar {
	s := &Sidebar{entries: entries}
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

func (s *Sidebar) items() []list.Item {
	items := make([]list.Item, 0, len(s.entries))
	for _, e := range s.entries {
		items = append(items, item{entry: e})
	}
	return items
}

func (s *Sidebar) SetEntries(entries []collection.Entry) {
	s.entries = entries
	_ = s.list.SetItems(s.items())
}

// Selected returns the currently highlighted request entry, or nil.
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
