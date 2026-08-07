package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Namer is a small modal that asks for a new request's name, opened with
// `a` over a folder in the sidebar. It renders as a centered overlay like
// the palette; enter confirms, esc cancels.
type Namer struct {
	input textinput.Model
	label string
}

func NewNamer() *Namer {
	n := &Namer{label: "new request name"}
	n.input = textinput.New()
	n.input.Prompt = "› "
	n.input.Placeholder = "e.g. list things"
	n.input.CharLimit = 80
	n.input.Width = 40
	n.input.PromptStyle = lipgloss.NewStyle().Foreground(ColorPrimary)
	n.input.Cursor.Style = lipgloss.NewStyle().Foreground(ColorPrimary)
	n.input.TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF"))
	n.input.PlaceholderStyle = lipgloss.NewStyle().Foreground(ColorMuted)
	return n
}

// Open focuses the input and clears it, ready for a new name.
func (n *Namer) Open() tea.Cmd {
	n.input.SetValue("")
	return n.input.Focus()
}

// Value returns the trimmed name the user typed.
func (n *Namer) Value() string { return strings.TrimSpace(n.input.Value()) }

func (n *Namer) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	n.input, cmd = n.input.Update(msg)
	return cmd
}

func (n *Namer) View() string {
	label := lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary).Render(n.label)
	return label + "\n" + n.input.View()
}
