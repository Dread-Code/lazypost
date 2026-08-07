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
	SecQuery Section = iota
	SecHeaders
	SecBody
	SecAuth
)

var sectionTabs = []string{"Query", "Headers", "Body", "Auth"}

type Editor struct {
	query   textarea.Model
	headers textarea.Model
	body    textarea.Model
	auth    AuthEditor
	section Section
	focused bool

	activePath string
	// pre/post are opaque Lua hook sources ([[Design - scripting hooks]]).
	// They have no UI section yet — carried verbatim between load and save.
	pre    string
	post   string
	width  int
	height int
}

// NewEditor builds the four-section editor. Method and URL are not
// here — they live in the URLBar ([[ADR-0010]]).
func NewEditor(width, height int) *Editor {
	e := &Editor{width: width, height: height}

	e.query = textarea.New()
	e.query.Placeholder = "tag: news\n# one Name: Value per line"
	e.query.Prompt = ""
	e.query.ShowLineNumbers = false
	e.query.CharLimit = -1

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
	e.query.SetWidth(inner)
	e.headers.SetWidth(inner)
	e.body.SetWidth(inner)
	e.auth.SetWidth(inner)

	contentH := e.height - 1 // tab row
	if contentH < 1 {
		contentH = 1
	}
	e.query.SetHeight(contentH)
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
	e.query.Blur()
	e.headers.Blur()
	e.body.Blur()
	e.auth.Blur()
}

func (e *Editor) focusSection() tea.Cmd {
	e.query.Blur()
	e.headers.Blur()
	e.body.Blur()
	e.auth.Blur()
	switch e.section {
	case SecQuery:
		return e.query.Focus()
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
		// ctrl+n/p and alt+arrows all cycle across the four tabs
		// (sections == tabs since URL left the editor in ADR-0010)
		case key.Matches(km, keySectionNext) || key.Matches(km, keyAltRight):
			e.section = Section((int(e.section) + 1) % 4)
			return e, e.focusSection()
		case key.Matches(km, keySectionPrev) || key.Matches(km, keyAltLeft):
			e.section = Section((int(e.section) + 3) % 4)
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
	case SecQuery:
		e.query, cmd = e.query.Update(msg)
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
	case SecQuery:
		content = e.query.View()
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
		Query:   parseParams(e.query.Value()),
		Headers: parseHeaders(e.headers.Value()),
		Auth:    e.auth.Auth(),
		Body:    e.body.Value(),
		Pre:     e.pre,
		Post:    e.post,
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

// parseParams parses query params from one "Name: Value" per line,
// mirroring the headers format.
func parseParams(s string) []collection.Param {
	var out []collection.Param
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		i := strings.Index(line, ":")
		if i <= 0 {
			continue
		}
		out = append(out, collection.Param{
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
	e.pre = req.Pre
	e.post = req.Post

	var qb strings.Builder
	for _, p := range req.Query {
		qb.WriteString(p.Name + ": " + p.Value + "\n")
	}
	e.query.SetValue(strings.TrimSuffix(qb.String(), "\n"))

	var b strings.Builder
	for _, h := range req.Headers {
		b.WriteString(h.Name + ": " + h.Value + "\n")
	}
	e.headers.SetValue(strings.TrimSuffix(b.String(), "\n"))
	e.body.SetValue(req.Body)
	e.auth.SetAuth(req.Auth)

	e.section = SecQuery
	e.Blur()
	return nil
}

// New resets the editor to a blank request.
func (e *Editor) New() tea.Cmd {
	e.activePath = ""
	e.pre = ""
	e.post = ""
	e.query.SetValue("")
	e.headers.SetValue("")
	e.body.SetValue("")
	e.auth.SetAuth(nil)
	e.section = SecQuery
	e.Blur()
	return nil
}

func (e *Editor) ActivePath() string        { return e.activePath }
func (e *Editor) SetActivePath(path string) { e.activePath = path }
