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

// The action column is padded, so the second key column starts at the
// same offset (46 = 2 + 18 key + 26 action) on every data row.
func TestHelpContentColumnsAligned(t *testing.T) {
	for _, line := range strings.Split(stripAnsiView(helpContent()), "\n") {
		if len(line) < 50 {
			continue // section header or blank
		}
		if line[46:50] != "    " {
			t.Errorf("second key column not aligned at 46: %q", line)
		}
	}
}
