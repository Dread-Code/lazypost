package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"postgo/internal/collection"
)

type Section int

const (
	SecURL Section = iota
	SecHeaders
	SecBody
	SecAuth
)

var Methods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}

var sectionTabs = []string{"Headers", "Body", "Auth"}

type Editor struct {
	method  string
	url     textinput.Model
	headers textarea.Model
	body    textarea.Model
	auth    AuthEditor
	section Section
	focused bool

	activePath string
	width      int
	height     int
}

func NewEditor(width, height int) *Editor {
	e := &Editor{method: "GET", width: width, height: height}

	e.url = textinput.New()
	e.url.Placeholder = "https://api.example.com/endpoint"
	e.url.Prompt = ""
	e.url.CharLimit = 2048

	e.headers = textarea.New()
	e.headers.Placeholder = "Content-Type: application/json\nAuthorization: Bearer ..."
	e.headers.Prompt = ""
	e.headers.ShowLineNumbers = false
	e.headers.CharLimit = -1

	e.body = textarea.New()
	e.body.Placeholder = `{"hello": "world"}`
	e.body.Prompt = ""
	e.body.ShowLineNumbers = true
	e.body.CharLimit = -1

	e.auth = NewAuthEditor()
	e.resize()
	return e
}

func (e *Editor) resize() {
	inner := e.width - 4 // pane border
	if inner < 10 {
		inner = 10
	}
	// leave room for "METHOD " before the input
	e.url.Width = inner - len(e.method) - 3
	e.headers.SetWidth(inner)
	e.body.SetWidth(inner)
	e.auth.SetWidth(inner)

	contentH := e.height - 6
	if contentH < 3 {
		contentH = 3
	}
	e.headers.SetHeight(contentH)
	e.body.SetHeight(contentH)
	e.auth.SetHeight(contentH)
}

func (e *Editor) Resize(width, height int) {
	e.width, e.height = width, height
	e.resize()
}

func (e *Editor) Focus() tea.Cmd {
	e.focused = true
	return e.focusSection()
}

func (e *Editor) Blur() {
	e.focused = false
	e.url.Blur()
	e.headers.Blur()
	e.body.Blur()
	e.auth.Blur()
}

func (e *Editor) focusSection() tea.Cmd {
	e.url.Blur()
	e.headers.Blur()
	e.body.Blur()
	e.auth.Blur()
	switch e.section {
	case SecURL:
		return e.url.Focus()
	case SecHeaders:
		return e.headers.Focus()
	case SecBody:
		return e.body.Focus()
	case SecAuth:
		return e.auth.Focus()
	}
	return nil
}

func (e *Editor) Update(msg tea.Msg) (*Editor, tea.Cmd) {
	if !e.focused {
		return e, nil
	}
	if km, ok := msg.(tea.KeyMsg); ok {
		switch {
		case key.Matches(km, keyCtrlDown):
			e.section = Section((int(e.section) + 1) % 4)
			return e, e.focusSection()
		case key.Matches(km, keyCtrlUp):
			e.section = Section((int(e.section) + 3) % 4)
			return e, e.focusSection()
		case key.Matches(km, keyAltLeft):
			// alt+arrows cycle only the Headers/Body/Auth tabs;
			// from URL jump straight to the last/first tab
			if e.section == SecURL {
				e.section = SecAuth
			} else {
				e.section = SecHeaders + Section((int(e.section)-int(SecHeaders)+2)%3)
			}
			return e, e.focusSection()
		case key.Matches(km, keyAltRight):
			if e.section == SecURL {
				e.section = SecHeaders
			} else {
				e.section = SecHeaders + Section((int(e.section)-int(SecHeaders)+1)%3)
			}
			return e, e.focusSection()
		case key.Matches(km, keyCtrlT):
			if e.section == SecURL {
				e.cycleMethod(1)
				return e, nil
			}
			if e.section == SecAuth {
				e.auth.CycleType(1)
				return e, nil
			}
		}
	}

	var cmd tea.Cmd
	switch e.section {
	case SecURL:
		e.url, cmd = e.url.Update(msg)
	case SecHeaders:
		e.headers, cmd = e.headers.Update(msg)
	case SecBody:
		e.body, cmd = e.body.Update(msg)
	case SecAuth:
		cmd = e.auth.Update(msg)
	}
	return e, cmd
}

func (e *Editor) cycleMethod(n int) {
	for i, m := range Methods {
		if m == e.method {
			e.method = Methods[(i+n+len(Methods))%len(Methods)]
			return
		}
	}
	e.method = Methods[0]
}

func (e *Editor) View() string {
	hint := ""
	urlW := e.width - 4 - len(e.method) - 3
	if e.section == SecURL && e.focused {
		hint = HintStyle.Render("  ctrl+t method")
		urlW -= 16 // reserve room for the hint so the row never wraps
	}
	if urlW < 10 {
		urlW = 10
	}
	e.url.Width = urlW
	method := MethodStyle(e.method).Render(e.method)
	urlRow := lipgloss.JoinHorizontal(lipgloss.Top, method, " ", e.url.View(), hint)

	active := -1 // -1 = no tab highlighted (URL section)
	if e.section >= SecHeaders {
		active = int(e.section) - int(SecHeaders)
	}
	tabRow := TabBar(sectionTabs, active)

	var content string
	switch e.section {
	case SecURL:
		content = HintStyle.Render(TruncateRunes("ctrl+↓ to edit headers, body or auth", e.width-4))
	case SecHeaders:
		content = e.headers.View()
	case SecBody:
		content = e.body.View()
	case SecAuth:
		content = e.auth.View()
	}

	return lipgloss.JoinVertical(lipgloss.Left, urlRow, tabRow, content)
}

// Request builds a collection.Request from the current editor state.
func (e *Editor) Request() *collection.Request {
	return &collection.Request{
		Name:    e.name(),
		Method:  e.method,
		URL:     strings.TrimSpace(e.url.Value()),
		Headers: parseHeaders(e.headers.Value()),
		Auth:    e.auth.Auth(),
		Body:    e.body.Value(),
	}
}

func (e *Editor) name() string {
	if e.activePath != "" {
		base := e.activePath
		if i := strings.LastIndex(base, "/"); i >= 0 {
			base = base[i+1:]
		}
		return strings.TrimSuffix(base, ".yaml")
	}
	return ""
}

func parseHeaders(s string) []collection.Header {
	var out []collection.Header
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		i := strings.Index(line, ":")
		if i <= 0 {
			continue
		}
		out = append(out, collection.Header{
			Name:  strings.TrimSpace(line[:i]),
			Value: strings.TrimSpace(line[i+1:]),
		})
	}
	return out
}

// SetRequest loads req into the editor. path may be empty for unsaved
// requests.
func (e *Editor) SetRequest(req *collection.Request, path string) tea.Cmd {
	e.activePath = path
	e.method = req.Method
	if e.method == "" {
		e.method = "GET"
	}
	e.url.SetValue(req.URL)

	var b strings.Builder
	for _, h := range req.Headers {
		b.WriteString(h.Name + ": " + h.Value + "\n")
	}
	e.headers.SetValue(strings.TrimSuffix(b.String(), "\n"))
	e.body.SetValue(req.Body)
	e.auth.SetAuth(req.Auth)

	e.section = SecURL
	return e.focusSection()
}

// New resets the editor to a blank request.
func (e *Editor) New() tea.Cmd {
	e.activePath = ""
	e.method = "GET"
	e.url.SetValue("")
	e.headers.SetValue("")
	e.body.SetValue("")
	e.auth.SetAuth(nil)
	e.section = SecURL
	return e.focusSection()
}

func (e *Editor) ActivePath() string        { return e.activePath }
func (e *Editor) SetActivePath(path string) { e.activePath = path }
