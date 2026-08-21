package themes

import (
	"embed"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"gopkg.in/yaml.v3"
)

// presetFS embeds the built-in themes as YAML (themes/*.yaml) — the same
// shape as user themes, so the format is the single source of truth and
// presets load through the identical parser.
//
//go:embed themes/*.yaml
var presetFS embed.FS

// Theme is a named set of colors. NewStyles compiles it into the immutable
// rendering snapshot consumed by widgets ([[Design - themes]]). The yaml
// tags mirror the user-theme file shape
// (~/.config/lazypost/themes/<name>.yaml); an empty AdaptiveColor pair
// means "unset, fall back to the default theme".
type Theme struct {
	Name    string                 `yaml:"name"`
	Primary lipgloss.AdaptiveColor `yaml:"primary"`
	Dim     lipgloss.AdaptiveColor `yaml:"dim"`
	Success lipgloss.AdaptiveColor `yaml:"success"`
	Warn    lipgloss.AdaptiveColor `yaml:"warn"`
	Error   lipgloss.AdaptiveColor `yaml:"error"`
	Info    lipgloss.AdaptiveColor `yaml:"info"`
	Accent  lipgloss.AdaptiveColor `yaml:"accent"`
	// Key is the shortcut keys in hints and the keybindings panel —
	// a distinct hue from the section accents so keys stand out.
	Key    lipgloss.AdaptiveColor `yaml:"key"`
	Muted  lipgloss.AdaptiveColor `yaml:"muted"`
	Border lipgloss.AdaptiveColor `yaml:"border"`
	Input  lipgloss.AdaptiveColor `yaml:"input"`
	// Field is the background of the URL input box (raised, subtle tones).
	Field lipgloss.AdaptiveColor `yaml:"field"`
	// Selection is the background fill of the highlighted row in lists
	// (sidebar, palette, history); OnSelection is the text color on it.
	Selection   lipgloss.AdaptiveColor            `yaml:"selection"`
	OnSelection lipgloss.AdaptiveColor            `yaml:"on_selection"`
	Methods     map[string]lipgloss.AdaptiveColor `yaml:"methods"`
}

// adaptive is a tiny helper for adaptive color pairs.
func adaptive(light, dark string) lipgloss.AdaptiveColor {
	return lipgloss.AdaptiveColor{Light: light, Dark: dark}
}

// copyColorMap returns a shallow copy of a method-color map (a nil map
// stays nil so Theme zero values render nothing).
func copyColorMap(src map[string]lipgloss.AdaptiveColor) map[string]lipgloss.AdaptiveColor {
	if src == nil {
		return nil
	}
	dst := make(map[string]lipgloss.AdaptiveColor, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

// defaultPresetName is the theme used when none is selected; it also
// leads the picker.
const defaultPresetName = "dracula"

// DefaultTheme is the fallback when no theme is selected. Dracula.
var DefaultTheme = Themes[defaultPresetName]

// Themes are the embedded presets, loaded from themes/*.yaml at package
// init. User themes live in ~/.config/lazypost/themes/*.yaml and are
// merged after these by LoadUserThemes — see ThemeByName/ThemeNames.
var Themes = loadPresets()

// presetOrder is the picker order of the embedded presets: the default
// theme first, then the rest in the alphabetical order go:embed reads
// them. It is derived from the files, so adding a preset is just
// dropping a YAML into themes/ — no list to keep in sync.
var presetOrder []string

// loadPresets parses every embedded themes/*.yaml into the preset
// registry, keyed by filename. These files are ours, so a parse failure
// is logged and skipped, never fatal.
func loadPresets() map[string]Theme {
	themes := map[string]Theme{}
	entries, err := presetFS.ReadDir("themes")
	if err != nil {
		log.Printf("lazypost: read embedded themes: %v", err)
		return themes
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".yaml")
		presetOrder = append(presetOrder, name)
		data, err := presetFS.ReadFile("themes/" + e.Name())
		if err != nil {
			log.Printf("lazypost: skipping preset %q: %v", name, err)
			continue
		}
		t, err := parseTheme(data)
		if err != nil {
			log.Printf("lazypost: skipping preset %q: %v", name, err)
			continue
		}
		t.Name = name
		themes[name] = t
	}
	// ReadDir yields sorted names; move the default theme to the front.
	sort.Strings(presetOrder)
	for i, n := range presetOrder {
		if n == defaultPresetName {
			presetOrder = append(presetOrder[:i], presetOrder[i+1:]...)
			presetOrder = append([]string{defaultPresetName}, presetOrder...)
			break
		}
	}
	return themes
}

// userThemes holds themes loaded from ~/.config/lazypost/themes/ at
// startup. Presets win name collisions; user themes only extend the set.
var userThemes = map[string]Theme{}

// LoadUserThemes reads every *.yaml file in dir, parses it as a Theme
// (missing keys fall back to DefaultTheme), and registers it under the
// filename. A file with an unparsable color, an unknown shape, or a
// preset's name is logged and skipped — a bad theme can never break the
// app. It returns the names it registered. A missing dir is not an error.
func LoadUserThemes(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var loaded []string
	for _, e := range entries {
		if e.IsDir() || strings.HasPrefix(e.Name(), ".") || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".yaml")
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			log.Printf("lazypost: skipping theme %q: %v", name, err)
			continue
		}
		t, err := parseTheme(data)
		if err != nil {
			log.Printf("lazypost: skipping theme %q: %v", name, err)
			continue
		}
		t = mergeTheme(DefaultTheme, t)
		if _, isPreset := Themes[name]; isPreset {
			log.Printf("lazypost: skipping theme %q: name collides with a built-in preset", name)
			continue
		}
		t.Name = name
		userThemes[name] = t
		loaded = append(loaded, name)
	}
	return loaded, nil
}

// ResetUserThemes clears the loaded user themes. Used by tests; a real
// run loads once at startup.
func ResetUserThemes() {
	userThemes = map[string]Theme{}
}

// parseTheme parses theme YAML data and validates its colors. Presets
// are complete files, so no fallback applies here; user themes merge
// over DefaultTheme via mergeTheme. Shared by both loaders, so the
// format is the single source of truth.
func parseTheme(data []byte) (Theme, error) {
	var t Theme
	if err := yaml.Unmarshal(data, &t); err != nil {
		return Theme{}, err
	}
	if !validThemeColors(t) {
		return Theme{}, fmt.Errorf("invalid color (want #RGB or #RRGGBB)")
	}
	return t, nil
}

// mergeTheme overlays the keys t sets onto base: a partially set
// AdaptiveColor pair (one side missing) inherits the base's other side;
// methods merge per entry. The result is the base theme, edited by t.
// The methods map is copied so mutating it never touches base's.
func mergeTheme(base, t Theme) Theme {
	base.Methods = copyColorMap(base.Methods)
	overlay := func(dst *lipgloss.AdaptiveColor, src lipgloss.AdaptiveColor) {
		if src.Light != "" {
			dst.Light = src.Light
		}
		if src.Dark != "" {
			dst.Dark = src.Dark
		}
	}
	overlay(&base.Primary, t.Primary)
	overlay(&base.Dim, t.Dim)
	overlay(&base.Success, t.Success)
	overlay(&base.Warn, t.Warn)
	overlay(&base.Error, t.Error)
	overlay(&base.Info, t.Info)
	overlay(&base.Accent, t.Accent)
	overlay(&base.Key, t.Key)
	overlay(&base.Muted, t.Muted)
	overlay(&base.Border, t.Border)
	overlay(&base.Input, t.Input)
	overlay(&base.Field, t.Field)
	overlay(&base.Selection, t.Selection)
	overlay(&base.OnSelection, t.OnSelection)
	for method, c := range t.Methods {
		if c.Light == "" && c.Dark == "" {
			continue
		}
		base.Methods[method] = c
	}
	return base
}

// validThemeColors reports whether every non-empty color in t is a valid
// #RGB or #RRGGBB hex string.
func validThemeColors(t Theme) bool {
	colors := []lipgloss.AdaptiveColor{
		t.Primary, t.Dim, t.Success, t.Warn, t.Error, t.Info,
		t.Accent, t.Key, t.Muted, t.Border, t.Input, t.Field,
		t.Selection, t.OnSelection,
	}
	for _, c := range colors {
		if !validHex(c.Light) || !validHex(c.Dark) {
			return false
		}
	}
	for _, c := range t.Methods {
		if !validHex(c.Light) || !validHex(c.Dark) {
			return false
		}
	}
	return true
}

// validHex reports whether s is empty (unset) or a #RGB/#RRGGBB hex
// color.
func validHex(s string) bool {
	if s == "" {
		return true
	}
	if !strings.HasPrefix(s, "#") {
		return false
	}
	h := s[1:]
	if len(h) != 3 && len(h) != 6 {
		return false
	}
	for _, r := range h {
		if !strings.ContainsRune("0123456789abcdefABCDEF", r) {
			return false
		}
	}
	return true
}

// ThemeByName returns the named theme — a user theme first (later loads
// override earlier ones), then a preset — falling back to DefaultTheme.
func ThemeByName(name string) Theme {
	if t, ok := userThemes[name]; ok {
		return t
	}
	if t, ok := Themes[name]; ok {
		return t
	}
	return DefaultTheme
}

// ThemeNames returns the embedded preset names in picker order (the
// default first, then alphabetical), followed by the loaded user theme
// names in lexicographic order.
func ThemeNames() []string {
	names := append([]string{}, presetOrder...)
	users := make([]string, 0, len(userThemes))
	for name := range userThemes {
		users = append(users, name)
	}
	sort.Strings(users)
	return append(names, users...)
}

func paneAccent(accent lipgloss.AdaptiveColor) PaneAccent {
	return PaneAccent{
		Active: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(accent),
		Legend: lipgloss.NewStyle().Bold(true).Foreground(accent),
		ActiveTab: lipgloss.NewStyle().
			Padding(0, 2).
			Foreground(accent).
			Bold(true).
			Underline(true),
	}
}
