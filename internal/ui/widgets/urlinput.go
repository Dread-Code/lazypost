package ui

import (
	"strings"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// urlField is a minimal single-line URL editor with per-token syntax
// coloring, replacing bubbles textinput for the URL bar ([[Design - url
// bar]]). The stock textinput applies one style to the whole value, so
// semantic URL parts need a custom widget.
//
// Scope: typing, backspace/delete, arrows, home/end, ctrl+a/e/u/k/w,
// paste, cursor-follow horizontal scrolling. The styling is recomputed on
// every render, so colors track edits live.
type urlField struct {
	value       string
	cursor      int // rune offset into value
	offset      int // first visible rune in the scroll window
	width       int
	focused     bool
	placeholder string
}

// urlCharLimit caps pasted/typed input like the old textinput CharLimit.
const urlCharLimit = 2048

func newURLField(placeholder string) *urlField {
	return &urlField{placeholder: placeholder}
}

func (f *urlField) Value() string { return f.value }

func (f *urlField) SetValue(v string) {
	f.value = v
	if f.cursor > utf8.RuneCountInString(v) {
		f.cursor = utf8.RuneCountInString(v)
	}
	f.ensureVisible()
}

func (f *urlField) CursorPos() int { return f.cursor }

func (f *urlField) SetCursor(pos int) {
	if n := utf8.RuneCountInString(f.value); pos > n {
		pos = n
	}
	if pos < 0 {
		pos = 0
	}
	f.cursor = pos
	f.ensureVisible()
}

func (f *urlField) Focus() {
	f.focused = true
	f.cursor = utf8.RuneCountInString(f.value)
	f.ensureVisible()
}

func (f *urlField) Blur() { f.focused = false }

func (f *urlField) SetWidth(w int) {
	f.width = w
	f.ensureVisible()
}

// runes returns the value as runes (editing works in rune space).
func (f *urlField) runes() []rune { return []rune(f.value) }

// insert puts s at the cursor, capping the total length.
func (f *urlField) insert(s string) {
	if s == "" {
		return
	}
	r := f.runes()
	in := []rune(s)
	room := urlCharLimit - len(r)
	if room <= 0 {
		return
	}
	if len(in) > room {
		in = in[:room]
	}
	out := make([]rune, 0, len(r)+len(in))
	out = append(out, r[:f.cursor]...)
	out = append(out, in...)
	out = append(out, r[f.cursor:]...)
	f.value = string(out)
	f.cursor += len(in)
	f.ensureVisible()
}

func (f *urlField) deleteBefore() {
	r := f.runes()
	if f.cursor <= 0 {
		return
	}
	out := make([]rune, 0, len(r)-1)
	out = append(out, r[:f.cursor-1]...)
	out = append(out, r[f.cursor:]...)
	f.value = string(out)
	f.cursor--
	f.ensureVisible()
}

func (f *urlField) deleteAt() {
	r := f.runes()
	if f.cursor >= len(r) {
		return
	}
	out := make([]rune, 0, len(r)-1)
	out = append(out, r[:f.cursor]...)
	out = append(out, r[f.cursor+1:]...)
	f.value = string(out)
	f.ensureVisible()
}

func (f *urlField) deleteToStart() {
	f.value = string(f.runes()[f.cursor:])
	f.cursor = 0
	f.ensureVisible()
}

func (f *urlField) deleteToEnd() {
	f.value = string(f.runes()[:f.cursor])
	f.ensureVisible()
}

// deleteWordBackward removes the word left of the cursor (ctrl+w /
// alt+backspace).
func (f *urlField) deleteWordBackward() {
	r := f.runes()
	if f.cursor <= 0 {
		return
	}
	i := f.cursor - 1
	for i >= 0 && isWordChar(r[i]) {
		i--
	}
	for i >= 0 && !isWordChar(r[i]) {
		i--
	}
	i++
	out := make([]rune, 0, len(r)-(f.cursor-i))
	out = append(out, r[:i]...)
	out = append(out, r[f.cursor:]...)
	f.value = string(out)
	f.cursor = i
	f.ensureVisible()
}

func isWordChar(r rune) bool {
	return r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' || r == '-'
}

func (f *urlField) moveCursor(d int) {
	if n := utf8.RuneCountInString(f.value); f.cursor+d >= 0 && f.cursor+d <= n {
		f.cursor += d
	}
	f.ensureVisible()
}

// ensureVisible scrolls the window so the cursor stays in view; the
// window is width+1 columns wide (one column for the cursor block).
func (f *urlField) ensureVisible() {
	if f.width < 1 {
		return
	}
	if f.cursor < f.offset {
		f.offset = f.cursor
	}
	if f.cursor >= f.offset+f.width+1 {
		f.offset = f.cursor - f.width
	}
}

func (f *urlField) Update(msg tea.Msg) (urlField, tea.Cmd) {
	if !f.focused {
		return *f, nil
	}
	km, ok := msg.(tea.KeyMsg)
	if !ok {
		return *f, nil
	}
	switch km.Type {
	case tea.KeyRunes, tea.KeySpace:
		// pasted text (non-curl) inserts like typing
		f.insert(string(km.Runes))
	case tea.KeyBackspace:
		f.deleteBefore()
	case tea.KeyDelete:
		f.deleteAt()
	case tea.KeyLeft:
		f.moveCursor(-1)
	case tea.KeyRight:
		f.moveCursor(1)
	case tea.KeyHome, tea.KeyCtrlA:
		f.cursor = 0
		f.ensureVisible()
	case tea.KeyEnd, tea.KeyCtrlE:
		f.cursor = utf8.RuneCountInString(f.value)
		f.ensureVisible()
	case tea.KeyCtrlU:
		f.deleteToStart()
	case tea.KeyCtrlK:
		f.deleteToEnd()
	case tea.KeyCtrlW:
		f.deleteWordBackward()
	}
	return *f, nil
}

// View renders the scroll window: per-token colors on the field
// background, with a cursor block at the edit position, exactly width+1
// columns so the field never shifts. The background is part of every
// cell — per-char style resets would otherwise punch holes in it.
func (f *urlField) View() string {
	if f.value == "" {
		return f.placeholderView()
	}
	tokens := tokenizeURL(f.value)
	refs := make([]runeRef, 0, len(f.value))
	for _, t := range tokens {
		for _, r := range t.text {
			refs = append(refs, runeRef{r: r, style: t.style})
		}
	}
	total := len(refs)
	window := f.offset + f.width + 1
	if window > total {
		window = total
	}
	var b strings.Builder
	for i := f.offset; i < window; i++ {
		var ch rune
		if i < total {
			ch = refs[i].r
		}
		style := refs[i].style.Background(ColorField)
		if f.focused && i == f.cursor {
			style = style.Reverse(true)
		}
		b.WriteString(style.Render(string(ch)))
	}
	for i := window; i < f.offset+f.width+1; i++ {
		if f.focused && i == f.cursor {
			b.WriteString(lipgloss.NewStyle().Background(ColorField).Reverse(true).Render(" "))
		} else {
			b.WriteString(lipgloss.NewStyle().Background(ColorField).Render(" "))
		}
	}
	return b.String()
}

// runeRef is one rune of the styled URL stream.
type runeRef struct {
	r     rune
	style lipgloss.Style
}

func (f *urlField) placeholderView() string {
	w := f.width + 1
	bg := lipgloss.NewStyle().Background(ColorField)
	var b strings.Builder
	if f.focused {
		b.WriteString(bg.Reverse(true).Render(" "))
		b.WriteString(bg.Render(TruncateRunes(f.placeholder, max(0, w-1))))
	} else {
		b.WriteString(bg.Render(TruncateRunes(f.placeholder, w)))
	}
	for i := lipgloss.Width(b.String()); i < w; i++ {
		b.WriteString(bg.Render(" "))
	}
	return b.String()
}

// urlToken is one styled span of the URL stream.
type urlToken struct {
	text  string
	style lipgloss.Style
}

// tokenizeURL splits a URL into semantic, styled parts: scheme://,
// userinfo@, host, :port, /path, ?query, #fragment — plus {{var}}
// placeholders anywhere in warn. The layout follows common URL parsing
// (userinfo ends at the last @, port at the last : of the authority).
func tokenizeURL(s string) []urlToken {
	var out []urlToken
	rest := s

	// scheme://
	if i := strings.Index(rest, "://"); i > 0 {
		out = append(out, urlToken{rest[:i+3], URLSchemeStyle})
		rest = rest[i+3:]
	}

	// authority (userinfo@host:port) up to / ? #
	authEnd := strings.IndexAny(rest, "/?#")
	if authEnd < 0 {
		authEnd = len(rest)
	}
	auth := rest[:authEnd]
	rest = rest[authEnd:]

	if i := strings.LastIndexByte(auth, '@'); i >= 0 {
		out = append(out, urlToken{auth[:i+1], URLUserInfoStyle})
		auth = auth[i+1:]
	}
	if i := strings.LastIndexByte(auth, ':'); i >= 0 {
		out = append(out, urlToken{auth[:i], URLHostStyle})
		out = append(out, urlToken{auth[i:], URLPortStyle})
	} else {
		out = append(out, urlToken{auth, URLHostStyle})
	}

	// path + query + fragment
	if i := strings.IndexByte(rest, '?'); i >= 0 {
		out = append(out, urlToken{rest[:i], URLPathStyle})
		rest = rest[i:]
		if j := strings.IndexByte(rest, '#'); j >= 0 {
			out = append(out, tokenizeQuery(rest[:j])...)
			out = append(out, urlToken{rest[j:], URLFragmentStyle})
		} else {
			out = append(out, tokenizeQuery(rest)...)
		}
	} else if i := strings.IndexByte(rest, '#'); i >= 0 {
		out = append(out, urlToken{rest[:i], URLPathStyle})
		out = append(out, urlToken{rest[i:], URLFragmentStyle})
	} else {
		out = append(out, urlToken{rest, URLPathStyle})
	}

	// {{var}} placeholders pop in warn wherever they appear
	var withVars []urlToken
	for _, t := range out {
		withVars = append(withVars, splitVars(t)...)
	}
	return withVars
}

// tokenizeQuery styles "?key=value&key2=value2": separators dim, keys in
// info, values at the default foreground.
func tokenizeQuery(s string) []urlToken {
	out := []urlToken{{"?", URLQuerySepStyle}}
	for i := 1; i < len(s); {
		j := i
		for j < len(s) && s[j] != '=' && s[j] != '&' {
			j++
		}
		if j > i {
			out = append(out, urlToken{s[i:j], URLQueryKeyStyle})
		}
		if j < len(s) && s[j] == '=' {
			out = append(out, urlToken{"=", URLQuerySepStyle})
			j++
			k := j
			for k < len(s) && s[k] != '&' {
				k++
			}
			if k > j {
				out = append(out, urlToken{s[j:k], URLQueryValueStyle})
			}
			j = k
		}
		if j < len(s) && s[j] == '&' {
			out = append(out, urlToken{"&", URLQuerySepStyle})
			j++
		}
		i = j
	}
	return out
}

// splitVars splits one token's text on {{…}} runs, restyling them in warn.
func splitVars(t urlToken) []urlToken {
	if !strings.Contains(t.text, "{{") {
		return []urlToken{t}
	}
	var out []urlToken
	rest := t.text
	for {
		i := strings.Index(rest, "{{")
		if i < 0 {
			break
		}
		if i > 0 {
			out = append(out, urlToken{rest[:i], t.style})
		}
		j := strings.Index(rest[i+2:], "}}")
		if j < 0 { // unclosed brace: treat the tail as a var
			out = append(out, urlToken{rest[i:], URLVarStyle})
			return out
		}
		end := i + 2 + j + 2
		out = append(out, urlToken{rest[i:end], URLVarStyle})
		rest = rest[end:]
	}
	if rest != "" {
		out = append(out, urlToken{rest, t.style})
	}
	return out
}
