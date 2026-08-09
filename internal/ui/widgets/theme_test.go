package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestThemePresets(t *testing.T) {
	if len(Themes) != 3 {
		t.Errorf("expected 3 embedded presets, got %d", len(Themes))
	}
	if _, ok := Themes["dracula"]; !ok {
		t.Error("dracula preset missing")
	}
	if _, ok := Themes["catppuccin"]; !ok {
		t.Error("catppuccin preset missing")
	}
	if _, ok := Themes["solarized"]; !ok {
		t.Error("solarized preset missing")
	}
}

func TestThemeByNameFallback(t *testing.T) {
	if ThemeByName("nope").Name != "dracula" {
		t.Error("unknown theme should fall back to default")
	}
	if ThemeByName("solarized").Name != "solarized" {
		t.Error("known theme should be returned")
	}
}

func TestThemeApplyRebuildsStyles(t *testing.T) {
	// styles start as the default (dracula) via init()
	solarized := ThemeByName("solarized")
	solarized.Apply()
	if lipgloss.AdaptiveColor(ColorPrimary) != solarized.Primary {
		t.Error("ColorPrimary not rebuilt by Apply")
	}
	if lipgloss.AdaptiveColor(ColorBorder) != solarized.Border {
		t.Error("ColorBorder not rebuilt by Apply")
	}
	if lipgloss.AdaptiveColor(ColorAccent) != solarized.Accent {
		t.Error("ColorAccent not rebuilt by Apply")
	}
	// the selection pair must reach the selected-row style
	if got := SelectedRowStyle.GetBackground(); got != solarized.Selection {
		t.Errorf("SelectedRowStyle background = %v, want %v", got, solarized.Selection)
	}
	if got := SelectedRowStyle.GetForeground(); got != solarized.OnSelection {
		t.Errorf("SelectedRowStyle foreground = %v, want %v", got, solarized.OnSelection)
	}
	if ModalStyle.GetBorderTopForeground() != solarized.Primary {
		t.Error("ModalStyle should wear the active (primary) border")
	}
	// per-section accents: sidebar stays the app identity (primary), the
	// request editor is information (info), the response is the result
	// (success)
	if got := SidebarAccent.Active.GetBorderTopForeground(); got != solarized.Primary {
		t.Errorf("SidebarAccent border = %v, want primary %v", got, solarized.Primary)
	}
	if got := EditorAccent.Active.GetBorderTopForeground(); got != solarized.Info {
		t.Errorf("EditorAccent border = %v, want info %v", got, solarized.Info)
	}
	if got := ResponseAccent.Active.GetBorderTopForeground(); got != solarized.Success {
		t.Errorf("ResponseAccent border = %v, want success %v", got, solarized.Success)
	}
	if EditorAccent.Legend.GetForeground() != solarized.Info {
		t.Error("EditorAccent legend should share the editor hue")
	}
	if EditorAccent.ActiveTab.GetForeground() != solarized.Info {
		t.Error("EditorAccent active tab should share the editor hue")
	}
	// restore default so later tests render the standard look
	DefaultTheme.Apply()
	if lipgloss.AdaptiveColor(ColorPrimary) != DefaultTheme.Primary {
		t.Error("default not restored")
	}
}

// TabBar must never wrap: it shrinks its padding and drops the farthest
// tabs, keeping the active one visible.
func TestTabBarFitsNarrowWidths(t *testing.T) {
	tabs := []string{"Query", "Headers", "Body", "Auth", "Scripts"}
	for _, active := range []int{0, 2, 4} {
		for _, maxW := range []int{40, 30, 20} {
			strip := TabBar(tabs, active, maxW, &EditorAccent)
			if w := lipgloss.Width(strip); w > maxW {
				t.Errorf("active=%d maxW=%d: strip %d wide: %q", active, maxW, w, strip)
			}
			// the active tab must stay visible at any width
			if !strings.Contains(stripAnsiTab(strip), tabs[active]) {
				t.Errorf("active tab %q dropped at maxW=%d: %q", tabs[active], maxW, strip)
			}
		}
	}
}

func stripAnsiTab(s string) string {
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
