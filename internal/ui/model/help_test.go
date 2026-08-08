package model

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

func TestHelpContentColored(t *testing.T) {
	prev := lipgloss.DefaultRenderer().ColorProfile()
	lipgloss.DefaultRenderer().SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.DefaultRenderer().SetColorProfile(prev) })

	got := helpContent()
	if !strings.Contains(got, "\x1b[") {
		t.Error("help content should color section headers and keys")
	}
	// visible text survives the styling
	for _, want := range []string{"Global", "send request", "ctrl+r", "Collection · sidebar", "URL bar", "Editor", "Response"} {
		if !strings.Contains(stripAnsiView(got), want) {
			t.Errorf("colored help lost %q", want)
		}
	}
}
