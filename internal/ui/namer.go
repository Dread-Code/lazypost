package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Namer is a small modal that asks for a new entry's name, opened with `a`
// over a folder in the sidebar. Typing a leading `/` switches it from "new
// request" to "new collection" mode, so enter creates a folder instead of
// a request. It renders as a centered overlay like the palette.
type Namer struct {
	input textinput.Model
}

func NewNamer() *Namer {
	n := &Namer{}
	n.input = textinput.New()
	n.input.Prompt = "› "
	n.input.Placeholder = "e.g. list things"
	n.input.CharLimit = 80
	n.input.Width = 40
	n.input.PromptStyle = lipgloss.NewStyle().Foreground(ColorPrimary)
	n.input.Cursor.Style = lipgloss.NewStyle().Foreground(ColorPrimary)
	n.input.TextStyle = lipgloss.NewStyle().Foreground(InputColor)
	n.input.PlaceholderStyle = lipgloss.NewStyle().Foreground(ColorMuted)
	return n
}

// Open focuses the input and clears it, ready for a new name.
func (n *Namer) Open() tea.Cmd {
	n.input.SetValue("")
	return n.input.Focus()
}

// OpenPrefilled opens the input with name already in it (used by rename).
func (n *Namer) OpenPrefilled(name string) tea.Cmd {
	n.input.SetValue(name)
	n.input.CursorEnd()
	return n.input.Focus()
}

// IsFolder reports whether the user typed a leading /, meaning they want a
// new collection (folder) rather than a request.
func (n *Namer) IsFolder() bool {
	return strings.HasPrefix(n.input.Value(), "/")
}

// Value returns the trimmed name the user typed, without the folder-mode
// leading slash.
func (n *Namer) Value() string {
	v := strings.TrimSpace(n.input.Value())
	if strings.HasPrefix(v, "/") {
		v = strings.TrimPrefix(v, "/")
	}
	return v
}

func (n *Namer) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	n.input, cmd = n.input.Update(msg)
	return cmd
}

func (n *Namer) View() string {
	label := "new request name"
	if n.IsFolder() {
		label = "new collection name"
	}
	l := lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary).Render(label)
	return l + "\n" + n.input.View()
}
