package ui

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"lazypost/internal/ui/themes"
)

// PaletteItem is one selectable command in the palette.
type PaletteItem struct {
	Title    string
	Shortcut string
	// Detail renders right after the title (e.g. "= value" for env
	// variables); it is only shown when Shortcut is empty.
	Detail string
}

func (i PaletteItem) FilterValue() string { return i.Title }

type paletteDelegate struct{}

func (d paletteDelegate) Height() int  { return 1 }
func (d paletteDelegate) Spacing() int { return 0 }

func (d paletteDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d paletteDelegate) Render(w io.Writer, m list.Model, index int, li list.Item) {
	it, ok := li.(PaletteItem)
	if !ok {
		return
	}
	selected := index == m.Index()

	avail := m.Width()
	if avail < 10 {
		avail = 10
	}

	// budget: cursor(2) + title + detail/shortcut (right side gets the
	// shortcut, a detail sits right after the title)
	sw := lipgloss.Width(it.Shortcut)
	titleW := avail - 2 - sw - 2
	if titleW < 4 {
		titleW = 4
	}
	title := themes.TruncateRunes(it.Title, titleW)
	detail := ""
	if it.Shortcut == "" && it.Detail != "" {
		// "= value" hangs off the title so "key = value" reads as one
		detail = " " + it.Detail
		title = themes.TruncateRunes(it.Title, avail-2-lipgloss.Width(detail))
		if lipgloss.Width(title)+lipgloss.Width(detail) > avail-2 {
			detail = themes.TruncateRunes(detail, avail-2-lipgloss.Width(title))
		}
	}

	right := ""
	if it.Shortcut != "" {
		right = lipgloss.NewStyle().Foreground(themes.ColorMuted).Render(it.Shortcut)
	}
	if selected {
		// the whole row becomes a solid block; every segment flips to the
		// selection foreground so the fill reads as one unit
		row := "▸ " + title + detail
		if pad := avail - lipgloss.Width(row) - lipgloss.Width(it.Shortcut); pad > 0 {
			row += strings.Repeat(" ", pad)
		}
		row += it.Shortcut
		row = themes.SelectedRowStyle.Render(padRunes(row, avail))
		fmt.Fprint(w, row)
		return
	}
	row := "  " + title + detail
	if pad := avail - lipgloss.Width(row) - sw; pad > 0 {
		row += strings.Repeat(" ", pad)
	}
	row += right
	fmt.Fprint(w, row)
}

// Palette is a fuzzy-filterable command overlay: a bubbles list with
// filtering enabled, rendered over the current frame (Design - command
// palette).
type Palette struct {
	list   list.Model
	items  []PaletteItem
	width  int
	height int
}

func NewPalette(width, height int) *Palette {
	p := &Palette{width: width, height: height}
	p.list = list.New(nil, paletteDelegate{}, width, height)
	p.list.SetShowTitle(false)
	p.list.SetShowFilter(true)
	p.list.SetShowHelp(false)
	p.list.SetShowPagination(false)
	p.list.SetShowStatusBar(false)
	p.list.SetFilteringEnabled(true)
	p.list.DisableQuitKeybindings()
	p.list.FilterInput.Prompt = "› "
	p.list.FilterInput.PromptStyle = lipgloss.NewStyle().Foreground(themes.ColorPrimary)
	p.list.FilterInput.Cursor.Style = lipgloss.NewStyle().Foreground(themes.ColorPrimary)
	p.list.FilterInput.TextStyle = lipgloss.NewStyle().Foreground(themes.InputColor)
	p.list.FilterInput.PlaceholderStyle = lipgloss.NewStyle().Foreground(themes.ColorMuted)
	// the default TitleBar pads 1 line under the filter; kill it so the
	// typed query sits flush above the items
	p.list.Styles.TitleBar = lipgloss.NewStyle().Padding(0, 0, 0, 1)
	p.list.Styles.NoItems = themes.HintStyle
	return p
}

// RefreshTheme reapplies styles copied into the bubbles list at construction.
func (p *Palette) RefreshTheme() {
	p.list.FilterInput.PromptStyle = lipgloss.NewStyle().Foreground(themes.ColorPrimary)
	p.list.FilterInput.Cursor.Style = lipgloss.NewStyle().Foreground(themes.ColorPrimary)
	p.list.FilterInput.TextStyle = lipgloss.NewStyle().Foreground(themes.InputColor)
	p.list.FilterInput.PlaceholderStyle = lipgloss.NewStyle().Foreground(themes.ColorMuted)
	p.list.Styles.NoItems = themes.HintStyle
}

func (p *Palette) SetItems(items []PaletteItem) {
	p.items = items
	li := make([]list.Item, len(items))
	for i, it := range items {
		li[i] = it
	}
	_ = p.list.SetItems(li)
	if len(items) > 0 {
		p.list.Select(0)
	}
}

func (p *Palette) Resize(width, height int) {
	p.width, p.height = width, height
	p.list.SetSize(width, height)
}

// Open shows all items and focuses the filter input so typing filters
// live. SetFilterText seeds the filtered set (state FilterApplied), then
// SetFilterState(Filtering) switches to live-editing without clearing it.
func (p *Palette) Open() {
	p.list.SetFilterText("")
	p.list.SetFilterState(list.Filtering)
}

// OpenBrowsing shows all items with the filter inactive, so letters do
// not type into a search box. Used by modals where typing is an action
// (e.g. the env manager's a/r/d) and "/" explicitly starts a filter.
func (p *Palette) OpenBrowsing() {
	p.list.SetFilterText("")
	p.list.SetFilterState(list.Unfiltered)
	p.list.FilterInput.Blur()
}

// StartFiltering activates the filter input (used when "/" is pressed in
// a browsing modal).
func (p *Palette) StartFiltering() {
	p.list.SetFilterState(list.Filtering)
}

// ClearFilter empties the filter input (used when exiting the env
// manager's filter mode).
func (p *Palette) ClearFilter() {
	p.list.SetFilterText("")
	p.list.SetFilterState(list.Unfiltered)
	p.list.FilterInput.Blur()
}

func (p *Palette) Update(msg tea.Msg) (tea.Cmd, *PaletteItem) {
	var cmd tea.Cmd
	p.list, cmd = p.list.Update(msg)
	// An async FilterMatchesMsg narrows the list without clamping the
	// cursor (bubbles only GoToStart on the synchronous SetFilterText
	// path), which can leave the selection out of bounds. Reset to the top
	// on every filter result, like a real command palette.
	if _, ok := msg.(list.FilterMatchesMsg); ok {
		p.list.GoToStart()
	}
	return cmd, p.Selected()
}

// Selected returns the currently highlighted item, if any.
func (p *Palette) Selected() *PaletteItem {
	it, ok := p.list.SelectedItem().(PaletteItem)
	if !ok {
		return nil
	}
	return &it
}

// CursorUp and CursorDown move the selection. They work even while the
// filter is being edited, which bubbles' Filtering state otherwise blocks.
func (p *Palette) CursorUp() {
	p.list.CursorUp()
}

func (p *Palette) CursorDown() {
	p.list.CursorDown()
}

func (p *Palette) View() string { return p.list.View() }
