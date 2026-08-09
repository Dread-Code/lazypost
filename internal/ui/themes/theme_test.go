package themes

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestThemePresets(t *testing.T) {
	if len(Themes) != 3 {
		t.Errorf("expected 3 embedded presets, got %d", len(Themes))
	}
	for _, name := range []string{"dracula", "catppuccin", "solarized"} {
		th, ok := Themes[name]
		if !ok {
			t.Errorf("%s preset missing", name)
			continue
		}
		if th.Name != name {
			t.Errorf("preset %s carries name %q", name, th.Name)
		}
		// presets are complete files: every color pair must be set, or a
		// partially written preset YAML would silently render blank
		for _, c := range []lipgloss.AdaptiveColor{th.Primary, th.Dim, th.Success, th.Warn,
			th.Error, th.Info, th.Accent, th.Muted, th.Border, th.Input, th.Field,
			th.Selection, th.OnSelection} {
			if c.Light == "" || c.Dark == "" {
				t.Errorf("preset %s has a partially set color: %+v", name, c)
			}
		}
		if len(th.Methods) < 7 {
			t.Errorf("preset %s should define all method colors, got %d", name, len(th.Methods))
		}
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

func writeTheme(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestLoadUserThemes exercises the YAML loader: valid files register,
// missing keys fall back to the default theme, and anything bad (bad
// color, preset collision, wrong extension) is skipped without breaking
// the rest.
func TestLoadUserThemes(t *testing.T) {
	t.Cleanup(func() { userThemes = map[string]Theme{} })

	dir := t.TempDir()
	writeTheme(t, dir, "midnight.yaml", `
primary: {light: "#111111", dark: "#222222"}
muted: {dark: "#6272A4"}
methods:
  GET: {light: "#008000", dark: "#50FA7B"}
`)
	writeTheme(t, dir, "broken.yaml", "primary: {light: \"purple\", dark: \"#222222\"}\n")
	writeTheme(t, dir, "dracula.yaml", "primary: {light: \"#111111\"}\n") // preset name
	writeTheme(t, dir, "ignored.yml", "primary: {light: \"#111111\"}\n")
	writeTheme(t, dir, ".hidden.yaml", "primary: {light: \"#111111\"}\n")

	loaded, err := LoadUserThemes(dir)
	if err != nil {
		t.Fatalf("LoadUserThemes: %v", err)
	}
	if len(loaded) != 1 || loaded[0] != "midnight" {
		t.Fatalf("expected only midnight to load, got %v", loaded)
	}

	// the user theme resolves by name, with missing keys from dracula
	th := ThemeByName("midnight")
	if th.Primary != (lipgloss.AdaptiveColor{Light: "#111111", Dark: "#222222"}) {
		t.Errorf("primary = %+v", th.Primary)
	}
	// partially set pair: the unset side inherits the default
	if th.Muted.Dark != "#6272A4" || th.Muted.Light != DefaultTheme.Muted.Light {
		t.Errorf("muted = %+v (dark from file, light should fall back to default)", th.Muted)
	}
	// unset fields keep the default theme's values
	if th.Success != DefaultTheme.Success || th.Border != DefaultTheme.Border {
		t.Error("unset fields should fall back to the default theme")
	}
	// method overrides merge over the defaults
	if th.Methods["GET"] != (lipgloss.AdaptiveColor{Light: "#008000", Dark: "#50FA7B"}) {
		t.Errorf("GET override = %+v", th.Methods["GET"])
	}
	if th.Methods["POST"] != DefaultTheme.Methods["POST"] {
		t.Error("unset methods should keep the default theme's")
	}
	// merging must never mutate the preset's own method map
	if DefaultTheme.Methods["GET"] != Themes["dracula"].Methods["GET"] {
		t.Error("user theme merge mutated the dracula preset's methods")
	}
	// a theme file whose name matches a preset still can't shadow it
	if got := ThemeByName("dracula").Primary; got != Themes["dracula"].Primary {
		t.Errorf("preset name collision must resolve to the preset, got primary %+v", got)
	}
}

func TestLoadUserThemesMissingDir(t *testing.T) {
	t.Cleanup(func() { userThemes = map[string]Theme{} })
	loaded, err := LoadUserThemes(filepath.Join(t.TempDir(), "does-not-exist"))
	if err != nil {
		t.Fatalf("missing dir should not error, got %v", err)
	}
	if len(loaded) != 0 {
		t.Errorf("expected no themes, got %v", loaded)
	}
}

func TestThemeNamesListsUserThemesAfterPresets(t *testing.T) {
	t.Cleanup(func() { userThemes = map[string]Theme{} })
	dir := t.TempDir()
	writeTheme(t, dir, "zen.yaml", "primary: {light: \"#111111\"}\n")
	if _, err := LoadUserThemes(dir); err != nil {
		t.Fatalf("LoadUserThemes: %v", err)
	}
	names := ThemeNames()
	if len(names) != 4 {
		t.Fatalf("expected 3 presets + 1 user theme, got %v", names)
	}
	if names[0] != "dracula" || names[1] != "catppuccin" || names[2] != "solarized" {
		t.Errorf("presets should lead in canonical order, got %v", names)
	}
	if names[3] != "zen" {
		t.Errorf("user theme should follow the presets, got %v", names)
	}
}

func TestValidHex(t *testing.T) {
	for _, ok := range []string{"", "#fff", "#FfF", "#FFFFFF", "#0a1b2c"} {
		if !validHex(ok) {
			t.Errorf("validHex(%q) = false, want true", ok)
		}
	}
	for _, bad := range []string{"fff", "#ff", "#ffff", "#ggg", "#FFFF", "#FFFFFFF", "white"} {
		if validHex(bad) {
			t.Errorf("validHex(%q) = true, want false", bad)
		}
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
