package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var Methods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}

// cycleMethod returns the method n steps around the Methods ring.
func cycleMethod(cur string, n int) string {
	for i, m := range Methods {
		if m == cur {
			return Methods[(i+n+len(Methods))%len(Methods)]
		}
	}
	return Methods[0]
}

// URLBar owns the request's method and URL — the only place they are
// edited ([[Design - request top bar]]). The URL renders in a raised
// field box (FieldStyle) with semantic parts colored live as it is
// edited. A right adornment (the env badge) can be attached via SetRight;
// its width is reserved from the field so the bar never overflows. Key
// hints are not rendered here — the status bar shows them for the focused
// pane.
type URLBar struct {
	method string
	url    urlField
	width  int
	right  string
}

func NewURLBar(width int) *URLBar {
	u := &URLBar{method: "GET", width: width}
	u.url = *newURLField("https://api.example.com/endpoint")
	u.resize()
	return u
}

// SetRight attaches a right-aligned adornment (the env badge); its width
// is reserved from the URL field.
func (u *URLBar) SetRight(s string) {
	u.right = s
	u.resize()
}

// resize sizes the URL field: total width minus the "METHOD " pill, the
// field's side padding, the right adornment, and the cursor column.
func (u *URLBar) resize() {
	// bar = pill + gap + field(padding 2 + width+1) + gap + right
	urlW := u.width - lipgloss.Width(MethodBadge(u.method)) - 1 - 3 - 2
	if u.right != "" {
		urlW -= lipgloss.Width(u.right) + 2
	}
	if urlW < 10 {
		urlW = 10
	}
	u.url.SetWidth(urlW)
}

func (u *URLBar) Resize(width int) {
	u.width = width
	u.resize()
}

func (u *URLBar) Focus() tea.Cmd {
	// entering the bar always puts the cursor at the end, so the URL is
	// ready to edit
	u.url.Focus()
	u.resize()
	return nil
}
func (u *URLBar) Blur() {
	u.url.Blur()
	u.resize()
}

func (u *URLBar) Update(msg tea.Msg) (*URLBar, tea.Cmd) {
	if km, ok := msg.(tea.KeyMsg); ok && key.Matches(km, keyCtrlT) {
		u.method = cycleMethod(u.method, 1)
		u.resize()
		return u, nil
	}
	var cmd tea.Cmd
	u.url, cmd = u.url.Update(msg)
	return u, cmd
}

// View renders one line; width math guarantees it never exceeds u.width.
// The width is maintained by Resize/Focus/Blur/SetRequest, never here —
// View must be free of side effects.
func (u *URLBar) View() string {
	line := lipgloss.JoinHorizontal(lipgloss.Top, MethodBadge(u.method), " ", u.fieldView())
	if u.right != "" {
		line += "  " + u.right
	}
	return line
}

// fieldView wraps the URL input in one background column of breathing
// room on each side (the input's own cells carry the background too).
func (u *URLBar) fieldView() string {
	pad := FieldStyle.Render(" ")
	return pad + u.url.View() + pad
}

func (u *URLBar) Method() string { return u.method }
func (u *URLBar) URL() string    { return strings.TrimSpace(u.url.Value()) }

// SetRequest loads a method + URL into the bar (from a request load or a
// curl import). Empty method falls back to GET.
func (u *URLBar) SetRequest(method, url string) {
	if method == "" {
		method = "GET"
	}
	u.method = method
	u.url.SetValue(url)
	u.resize()
}

func (u *URLBar) New() {
	u.method = "GET"
	u.url.SetValue("")
	u.resize()
}
