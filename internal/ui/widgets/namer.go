package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Namer is a small modal that asks for a name, opened with `a` over a
// folder in the sidebar. Typing a leading `/` switches it from "new
// request" to "new collection" mode, so enter creates a folder instead of
// a request. It also serves the env manager for key=value edits, via
// SetLabel/SetPlaceholder. It renders as a centered overlay like the
// palette.
type Namer struct {
	input       textinput.Model
	label       string
	placeholder string
	envMode     bool
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

// SetLabel overrides the modal title (e.g. "new variable"). An empty label
// falls back to the request/collection wording.
func (n *Namer) SetLabel(label string) {
	n.label = label
}

// SetEnvMode switches the namer into env-variable mode: a leading "/"
// reads as a new environment instead of a new collection, and the title
// follows the mode.
func (n *Namer) SetEnvMode(envMode bool) {
	n.envMode = envMode
}

// Label returns the title to render for the current input state.
func (n *Namer) Label() string {
	if n.label != "" {
		if n.IsFolder() {
			if n.envMode {
				return "new environment name"
			}
			return "new collection name"
		}
		return n.label
	}
	label := "new request name"
	if n.IsFolder() {
		label = "new collection name"
	}
	return label
}

// SetPlaceholder overrides the input placeholder (e.g. "key=value"). An
// empty placeholder falls back to the default.
func (n *Namer) SetPlaceholder(ph string) {
	n.placeholder = ph
	n.input.Placeholder = ph
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
	// the label lives on the modal's border title (model/view.go
	// renderModal), so the content is just the input
	return n.input.View()
}
