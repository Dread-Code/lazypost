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

	got := helpContent(160)
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

// Every data row is padded to the exact layout width, so the two columns
// line up on every row and the section dash-runs span the same width.
func TestHelpContentColumnsAligned(t *testing.T) {
	lines := strings.Split(stripAnsiView(helpContent(160)), "\n")
	dataW := 0
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		if dataW == 0 {
			dataW = lipgloss.Width(line)
			continue
		}
		if w := lipgloss.Width(line); w != dataW {
			t.Errorf("row %d wide, want %d: %q", w, dataW, line)
		}
	}
	if dataW == 0 {
		t.Fatal("panel rendered no lines")
	}
}

// The panel must fit a narrow terminal (single-column fallback) without
// overflowing the width it is given.
func TestHelpContentFitsNarrowTerminal(t *testing.T) {
	prev := lipgloss.DefaultRenderer().ColorProfile()
	lipgloss.DefaultRenderer().SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.DefaultRenderer().SetColorProfile(prev) })

	for _, maxW := range []int{80, 60, 44} {
		for _, line := range strings.Split(stripAnsiView(helpContent(maxW)), "\n") {
			if w := lipgloss.Width(line); w > maxW {
				t.Errorf("line %d wide exceeds %d: %q", w, maxW, line)
			}
		}
	}
}
