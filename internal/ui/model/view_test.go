package model

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

// stripAnsiView removes ANSI sequences so assertions work on visible text.
func stripAnsiView(s string) string {
	var b strings.Builder
	in := false
	for _, r := range s {
		if r == 0x1b {
			in = true
			continue
		}
		if in {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				in = false
			}
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func TestRenderModalTitleRightAligned(t *testing.T) {
	// content wider than the title, so the slack lands on the left
	box := renderModal("History", "a line that is much wider than the title")
	lines := strings.Split(stripAnsiView(box), "\n")
	if len(lines) != 3 { // border + content + border
		t.Fatalf("unexpected line count %d:\n%q", len(lines), box)
	}
	top := lines[0]
	if !strings.HasPrefix(top, "╭────") {
		t.Errorf("title should be preceded by a dash run, got %q", top)
	}
	if !strings.Contains(top, "History") {
		t.Errorf("title missing from border, got %q", top)
	}
	// right-aligned: the title is followed only by the corner, not a dash run
	titleIdx := strings.Index(top, "History")
	after := top[titleIdx+len("History"):]
	if !strings.HasSuffix(after, "╮") {
		t.Errorf("title should sit flush right before the corner, got %q", top)
	}
}

func TestRenderModalWidthFitsTitle(t *testing.T) {
	// a title wider than the content widens the box
	box := renderModal("delete a really long request name", "y")
	top := strings.Split(stripAnsiView(box), "\n")[0]
	if !strings.Contains(top, "delete a really long request name") {
		t.Errorf("wide title should not be clipped, got %q", top)
	}
	if lipgloss.Width(top) < lipgloss.Width("delete a really long request name")+2 {
		t.Error("box should be at least title + corners wide")
	}
}
