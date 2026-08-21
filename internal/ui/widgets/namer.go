package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Dread-Code/lazypost/internal/ui/themes"
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
	styles      themes.Styles
}

func NewNamer() *Namer {
	styles := themes.NewStyles(themes.DefaultTheme)
	n := &Namer{styles: styles}
	n.input = textinput.New()
	n.input.Prompt = "› "
	n.input.Placeholder = "e.g. list things"
	n.input.CharLimit = 80
	n.input.Width = 40
	n.SetStyles(styles)
	return n
}

// SetStyles replaces the rendering snapshot used by the input.
func (n *Namer) SetStyles(styles themes.Styles) {
	n.styles = styles
	n.input.PromptStyle = lipgloss.NewStyle().Foreground(styles.ColorPrimary)
	n.input.Cursor.Style = lipgloss.NewStyle().Foreground(styles.ColorPrimary)
	n.input.TextStyle = lipgloss.NewStyle().Foreground(styles.InputColor)
	n.input.PlaceholderStyle = lipgloss.NewStyle().Foreground(styles.ColorMuted)
}

// RefreshTheme is retained as a compatibility helper for standalone widget
// callers; the root model uses SetStyles with its local snapshot.
func (n *Namer) RefreshTheme() {
	n.SetStyles(themes.NewStyles(themes.DefaultTheme))
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
	// renderModal), so the content is the input plus the key hint
	return lipgloss.JoinVertical(lipgloss.Left,
		n.input.View(),
		n.styles.KeyHint("enter", "confirm", "esc", "cancel"),
	)
}
