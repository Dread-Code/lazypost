package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Confirm is a small yes/no modal, opened before destructive actions like
// deleting a request or a folder. y/enter confirms, n/esc cancels. It
// renders as a centered overlay like the palette; the model routes keys
// to it while open. It is stateless between Ask calls — the model owns
// the answer, so it has no Update.
type Confirm struct {
	label string
}

func NewConfirm() *Confirm {
	return &Confirm{}
}

// Ask sets the question shown to the user.
func (c *Confirm) Ask(label string) {
	c.label = sanitizeLabel(label)
}

func (c *Confirm) View() string {
	question := lipgloss.NewStyle().Bold(true).Foreground(ColorError).Render(c.label)
	hint := lipgloss.NewStyle().Foreground(ColorMuted).Render("y yes · n no")
	return question + "\n" + hint
}

// sanitizeLabel guards against multi-line labels leaking into the overlay.
func sanitizeLabel(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	return s
}
