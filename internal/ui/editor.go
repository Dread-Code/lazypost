package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"postgo/internal/collection"
)

type Section int

const (
	SecHeaders Section = iota
	SecBody
	SecAuth
)

var sectionTabs = []string{"Headers", "Body", "Auth"}

type Editor struct {
	headers textarea.Model
	body    textarea.Model
	auth    AuthEditor
	section Section
	focused bool

	activePath string
	width      int
	height     int
}

// NewEditor builds the three-section editor. Method and URL are not
// here — they live in the URLBar ([[ADR-0010]]).
func NewEditor(width, height int) *Editor {
	e := &Editor{width: width, height: height}

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
	e.headers.SetWidth(inner)
	e.body.SetWidth(inner)
	e.auth.SetWidth(inner)

	contentH := e.height - 1 // tab row
	if contentH < 1 {
		contentH = 1
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
	e.headers.Blur()
	e.body.Blur()
	e.auth.Blur()
}

func (e *Editor) focusSection() tea.Cmd {
	e.headers.Blur()
	e.body.Blur()
	e.auth.Blur()
	switch e.section {
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
		// ctrl+n/p move across sections; modulo keeps the cycle wrapping
		case key.Matches(km, keySectionNext):
			e.section = Section((int(e.section) + 1) % 3)
			return e, e.focusSection()
		case key.Matches(km, keySectionPrev):
			e.section = Section((int(e.section) + 2) % 3)
			return e, e.focusSection()
		// alt+arrows cycle within the three tabs only
		case key.Matches(km, keyAltLeft):
			e.section = SecHeaders + Section((int(e.section)-int(SecHeaders)+2)%3)
			return e, e.focusSection()
		case key.Matches(km, keyAltRight):
			e.section = SecHeaders + Section((int(e.section)-int(SecHeaders)+1)%3)
			return e, e.focusSection()
		case key.Matches(km, keyCtrlT):
			if e.section == SecAuth {
				e.auth.CycleType(1)
				return e, nil
			}
		}
	}

	var cmd tea.Cmd
	switch e.section {
	case SecHeaders:
		e.headers, cmd = e.headers.Update(msg)
	case SecBody:
		e.body, cmd = e.body.Update(msg)
	case SecAuth:
		cmd = e.auth.Update(msg)
	}
	return e, cmd
}

func (e *Editor) View() string {
	tabRow := TabBar(sectionTabs, int(e.section))

	var content string
	switch e.section {
	case SecHeaders:
		content = e.headers.View()
	case SecBody:
		content = e.body.View()
	case SecAuth:
		content = e.auth.View()
	}

	return lipgloss.JoinVertical(lipgloss.Left, tabRow, content)
}

// Request builds a collection.Request from the editor state. Method and
// URL come from the URLBar; the root model fills them in.
func (e *Editor) Request() *collection.Request {
	return &collection.Request{
		Name:    e.name(),
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
// requests. Focus is decided by the caller; widgets are blurred here.
func (e *Editor) SetRequest(req *collection.Request, path string) tea.Cmd {
	e.activePath = path

	var b strings.Builder
	for _, h := range req.Headers {
		b.WriteString(h.Name + ": " + h.Value + "\n")
	}
	e.headers.SetValue(strings.TrimSuffix(b.String(), "\n"))
	e.body.SetValue(req.Body)
	e.auth.SetAuth(req.Auth)

	e.section = SecHeaders
	e.Blur()
	return nil
}

// New resets the editor to a blank request.
func (e *Editor) New() tea.Cmd {
	e.activePath = ""
	e.headers.SetValue("")
	e.body.SetValue("")
	e.auth.SetAuth(nil)
	e.section = SecHeaders
	e.Blur()
	return nil
}

func (e *Editor) ActivePath() string        { return e.activePath }
func (e *Editor) SetActivePath(path string) { e.activePath = path }
