package ui

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"lazypost/internal/app"
)

// HistoryItem is one past send shown in the history overlay.
type HistoryItem struct {
	Req     app.HistoryEntry
	Summary string
	At      time.Time
}

func (i HistoryItem) FilterValue() string { return i.Req.Req.Name }

// HistoryTitle renders the row's label: the request name, or METHOD url
// when the request is unnamed.
func (i HistoryItem) HistoryTitle() string {
	if i.Req.Req.Name != "" {
		return i.Req.Req.Name
	}
	if i.Req.Req.URL != "" {
		return i.Req.Req.Method + " " + i.Req.Req.URL
	}
	return i.Req.Req.Method
}

type historyDelegate struct{}

func (d historyDelegate) Height() int  { return 1 }
func (d historyDelegate) Spacing() int { return 0 }

func (d historyDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d historyDelegate) Render(w io.Writer, m list.Model, index int, li list.Item) {
	it, ok := li.(HistoryItem)
	if !ok {
		return
	}
	selected := index == m.Index()

	avail := m.Width() - 2
	if avail < 10 {
		avail = 10
	}

	// summary is right-aligned, the title takes the rest
	sw := lipgloss.Width(it.Summary)
	titleW := avail - 2 - sw
	if titleW < 4 {
		titleW = 4
	}
	title := TruncateRunes(it.HistoryTitle(), titleW)
	summary := lipgloss.NewStyle().Foreground(ColorMuted).Render(it.Summary)

	cursor := "  "
	if selected {
		cursor = lipgloss.NewStyle().Foreground(ColorPrimary).Render("▸ ")
		title = lipgloss.NewStyle().Foreground(ColorPrimary).Render(title)
	}
	pad := avail - lipgloss.Width(title) - lipgloss.Width(summary)
	if pad < 0 {
		pad = 0 // a long summary (e.g. an error) can exceed the row
	}
	line := cursor + title + strings.Repeat(" ", pad) + summary
	fmt.Fprint(w, line)
}

// History is the ctrl+h overlay: a browsing list of past sends with no
// filter (typing is not needed here), rendered over the current frame
// ([[Design - request history]]).
type History struct {
	list   list.Model
	width  int
	height int
}

func NewHistory(width, height int) *History {
	h := &History{width: width, height: height}
	h.list = list.New(nil, historyDelegate{}, width, height)
	h.list.SetShowTitle(false) // the title lives on the modal's border legend
	h.list.SetShowFilter(false)
	h.list.SetShowHelp(false)
	h.list.SetShowPagination(false)
	h.list.SetShowStatusBar(false)
	h.list.DisableQuitKeybindings()
	return h
}

func (h *History) SetItems(entries []app.HistoryEntry) {
	li := make([]list.Item, len(entries))
	for i, e := range entries {
		li[i] = HistoryItem{Req: e, Summary: e.Summary, At: e.At}
	}
	_ = h.list.SetItems(li)
	if len(li) > 0 {
		h.list.Select(0)
	}
}

func (h *History) Resize(width, height int) {
	h.width, h.height = width, height
	h.list.SetSize(width, height)
}

// Open shows the list with the filter off so navigation keys are free.
func (h *History) Open() {
	h.list.SetFilterState(list.Unfiltered)
	h.list.FilterInput.Blur()
}

func (h *History) Update(msg tea.Msg) tea.Cmd {
	if km, ok := msg.(tea.KeyMsg); ok {
		// ctrl+n/p move the cursor like the arrows (bubbles list only
		// knows up/down/j/k), mirroring the sidebar
		switch {
		case key.Matches(km, keySectionNext):
			h.list.CursorDown()
			return nil
		case key.Matches(km, keySectionPrev):
			h.list.CursorUp()
			return nil
		}
	}
	var cmd tea.Cmd
	h.list, cmd = h.list.Update(msg)
	return cmd
}

func (h *History) Selected() *HistoryItem {
	it, ok := h.list.SelectedItem().(HistoryItem)
	if !ok {
		return nil
	}
	return &it
}

func (h *History) CursorUp()   { h.list.CursorUp() }
func (h *History) CursorDown() { h.list.CursorDown() }

func (h *History) View() string { return h.list.View() }
