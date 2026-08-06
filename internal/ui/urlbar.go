package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var Methods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}

func cycleMethod(cur string, n int) string {
	for i, m := range Methods {
		if m == cur {
			return Methods[(i+n+len(Methods))%len(Methods)]
		}
	}
	return Methods[0]
}

const barHint = "  ctrl+t method · enter send · esc back"

// URLBar owns the request's method and URL — the only place they are
// edited ([[Design - request top bar]]).
type URLBar struct {
	method string
	url    textinput.Model
	width  int
}

func NewURLBar(width int) *URLBar {
	u := &URLBar{method: "GET", width: width}
	u.url = textinput.New()
	u.url.Placeholder = "https://api.example.com/endpoint"
	u.url.Prompt = ""
	u.url.CharLimit = 2048
	u.resize()
	return u
}

func (u *URLBar) resize() {
	urlW := u.width - len(u.method) - 1 // "METHOD " prefix
	if u.url.Focused() {
		urlW -= len(barHint)
	}
	if urlW < 10 {
		urlW = 10
	}
	u.url.Width = urlW
}

func (u *URLBar) Resize(width int) {
	u.width = width
	u.resize()
}

func (u *URLBar) Focus() tea.Cmd { return u.url.Focus() }
func (u *URLBar) Blur()          { u.url.Blur() }

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
func (u *URLBar) View() string {
	u.resize() // hint room depends on focus state
	method := MethodStyle(u.method).Render(u.method)
	hint := ""
	if u.url.Focused() {
		hint = HintStyle.Render(barHint)
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, method, " ", u.url.View(), hint)
}

func (u *URLBar) Method() string { return u.method }
func (u *URLBar) URL() string    { return strings.TrimSpace(u.url.Value()) }

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
