package ui

import (
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
	// restore default so later tests render the standard look
	DefaultTheme.Apply()
	if lipgloss.AdaptiveColor(ColorPrimary) != DefaultTheme.Primary {
		t.Error("default not restored")
	}
}
