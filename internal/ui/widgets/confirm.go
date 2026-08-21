package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/Dread-Code/lazypost/internal/ui/themes"
)

// Confirm is a small yes/no modal, opened before destructive actions like
// deleting a request or a folder. y/enter confirms, n/esc cancels. It
// renders as a centered overlay like the palette; the model routes keys
// to it while open. It is stateless between Ask calls — the model owns
// the answer, so it has no Update.
type Confirm struct {
	label  string
	styles themes.Styles
}

func NewConfirm() *Confirm {
	return &Confirm{styles: themes.NewStyles(themes.DefaultTheme)}
}

// SetStyles replaces the rendering snapshot used by the confirmation hints.
func (c *Confirm) SetStyles(styles themes.Styles) { c.styles = styles }

// RefreshTheme is retained as a compatibility helper for standalone widget
// callers; the root model uses SetStyles with its local snapshot.
func (c *Confirm) RefreshTheme() { c.SetStyles(themes.NewStyles(themes.DefaultTheme)) }

// Ask sets the question shown to the user.
func (c *Confirm) Ask(label string) {
	c.label = sanitizeLabel(label)
}

// Label returns the question, which the modal's border title renders.
func (c *Confirm) Label() string { return c.label }

func (c *Confirm) View() string {
	// the question lives on the modal's border title (model/view.go
	// renderModal), so the content is just the hint
	return lipgloss.JoinVertical(lipgloss.Left,
		c.styles.KeyHint("y", "yes", "n", "no"),
		c.styles.KeyHint("enter", "confirms", "esc", "cancels"),
	)
}

// sanitizeLabel guards against multi-line labels leaking into the overlay.
func sanitizeLabel(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	return s
}
