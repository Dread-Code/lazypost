package ui

import (
	"fmt"
	"strings"

	"github.com/Dread-Code/lazypost/lib/codeeditor"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Dread-Code/lazypost/internal/collection"
	"github.com/Dread-Code/lazypost/internal/render"

	"github.com/Dread-Code/lazypost/internal/ui/themes"
)

type Section int

const (
	SecQuery Section = iota
	SecHeaders
	SecBody
	SecAuth
	SecScripts
)

var sectionTabs = []string{"Query", "Headers", "Body", "Auth", "Scripts"}

type Editor struct {
	query   *codeeditor.Editor
	headers *codeeditor.Editor
	body    *codeeditor.Editor
	auth    AuthEditor
	pre     *codeeditor.Editor
	post    *codeeditor.Editor
	section Section
	// field selects which script editor (0=pre, 1=post) has focus
	field   int
	focused bool

	activePath  string
	title       string // display name from the loaded request; "" = none
	width       int
	height      int
	pendingYank string
	styles      *themes.Styles
}

// NewEditor builds the four-section editor. Method and URL are not
// here — they live in the URLBar ([[ADR-0010]]).
func NewEditor(width, height int) *Editor {
	styles := themes.NewStyles(themes.DefaultTheme)
	return NewEditorWithStyles(width, height, styles)
}

func NewEditorWithStyles(width, height int, styles themes.Styles) *Editor {
	e := &Editor{width: width, height: height, styles: &styles}

	e.query = codeeditor.New(0, 0, "tag: news\n# one Name: Value per line", nil)

	e.headers = codeeditor.New(0, 0, "Content-Type: application/json\nAuthorization: Bearer ...", nil)

	e.body = codeeditor.New(0, 0, `{"hello": "world"}`, jsonHighlighterWithStyles(e.styles))

	e.pre = codeeditor.New(0, 0, "-- runs before the request", luaHighlighterWithStyles(e.styles))
	e.post = codeeditor.New(0, 0, "-- runs after the response", luaHighlighterWithStyles(e.styles))

	for _, ed := range []*codeeditor.Editor{e.query, e.headers, e.body, e.pre, e.post} {
		ed.SetStyleProvider(func() codeeditor.Style { return editorStyles(*e.styles) })
		ed.SetYank(func(s string) { e.pendingYank = s })
	}

	e.auth = NewAuthEditor()
	e.auth.SetStyles(styles)
	e.resize()
	return e
}

func (e *Editor) resize() {
	inner := e.width - 2 // pane content width
	if inner < 10 {
		inner = 10
	}
	e.auth.SetWidth(inner)

	contentH := e.height - 3 // tab row + divider + mode footer
	if contentH < 1 {
		contentH = 1
	}
	// the Scripts tab shows one hook at a time below a toggle row
	scriptH := contentH - 1
	if scriptH < 1 {
		scriptH = 1
	}
	e.query.Resize(inner, contentH)
	e.headers.Resize(inner, contentH)
	e.body.Resize(inner, contentH)
	e.pre.Resize(inner, scriptH)
	e.post.Resize(inner, scriptH)
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
	e.FormatBody()
	e.focused = false
	e.query.Blur()
	e.headers.Blur()
	e.body.Blur()
	e.pre.Blur()
	e.post.Blur()
	e.auth.Blur()
}

// FormatBody pretty-prints the body like the response pane, mirroring the
// response's FormattedBody ([[ADR-0016 Body JSON auto-format on save and
// blur]] · [[ADR-0017]]). Valid JSON formats directly; bodies that are
// invalid only because of raw {{placeholders}} in value positions (e.g.
// `"userId": {{user_id}}`) are formatted around the placeholders by
// render.FormatJSON. Genuinely non-JSON bodies (half-typed, plain text)
// are left exactly as typed; idempotent for already-formatted bodies.
func (e *Editor) FormatBody() {
	e.body.SetValue(render.FormatJSON(e.body.Value()))
}

// SetStyles replaces the rendering snapshot and invalidates cached syntax
// output after a runtime theme change.
func (e *Editor) SetStyles(styles themes.Styles) {
	if e.styles == nil {
		e.styles = &styles
	} else {
		*e.styles = styles
	}
	for _, ed := range []*codeeditor.Editor{e.query, e.headers, e.body, e.pre, e.post} {
		ed.SetStyleProvider(func() codeeditor.Style { return editorStyles(*e.styles) })
		ed.InvalidateHighlight()
	}
	e.auth.SetStyles(styles)
}

// RefreshTheme is retained as a compatibility helper for standalone widget
// callers; the root model uses SetStyles with its local snapshot.
func (e *Editor) RefreshTheme() {
	e.SetStyles(themes.NewStyles(themes.DefaultTheme))
}

// TakeYank returns and clears text produced by a code-editor yank. The root
// model turns it into an asynchronous clipboard command.
func (e *Editor) TakeYank() string {
	yank := e.pendingYank
	e.pendingYank = ""
	return yank
}

func (e *Editor) focusSection() tea.Cmd {
	e.query.Blur()
	e.headers.Blur()
	e.body.Blur()
	e.pre.Blur()
	e.post.Blur()
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
	case SecScripts:
		if e.field == 1 {
			return e.post.Focus()
		}
		return e.pre.Focus()
	}
	return nil
}

func (e *Editor) Update(msg tea.Msg) (*Editor, tea.Cmd) {
	if !e.focused {
		return e, nil
	}
	if km, ok := msg.(tea.KeyMsg); ok {
		switch {
		// ctrl+n/p and alt+arrows cycle the tabs (sections == tabs since
		// URL left the editor in ADR-0010)
		case key.Matches(km, keySectionNext) || key.Matches(km, keyAltRight):
			next := Section((int(e.section) + 1) % len(sectionTabs))
			if e.section == SecBody && next != SecBody {
				e.FormatBody()
			}
			e.section = next
			return e, e.focusSection()
		case key.Matches(km, keySectionPrev) || key.Matches(km, keyAltLeft):
			next := Section((int(e.section) - 1 + len(sectionTabs)) % len(sectionTabs))
			if e.section == SecBody && next != SecBody {
				e.FormatBody()
			}
			e.section = next
			return e, e.focusSection()
		case key.Matches(km, keyCtrlT):
			switch e.section {
			case SecAuth:
				e.auth.CycleType(1)
				return e, nil
			case SecScripts:
				// ctrl+t toggles which hook (pre/post) is edited, like the
				// auth type row
				e.field = (e.field + 1) % 2
				return e, e.focusSection()
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
	case SecScripts:
		if e.field == 1 {
			e.post, cmd = e.post.Update(msg)
		} else {
			e.pre, cmd = e.pre.Update(msg)
		}
	case SecAuth:
		cmd = e.auth.Update(msg)
	}
	return e, cmd
}

func (e *Editor) View() string {
	styles := *e.styles
	tabRow := styles.TabBar(sectionTabs, int(e.section), max(0, e.width-2), &styles.EditorAccent)
	divider := styles.Rule(max(0, e.width-2))

	var content string
	switch e.section {
	case SecQuery:
		content = e.query.View()
	case SecHeaders:
		content = e.headers.View()
	case SecBody:
		content = e.body.View()
	case SecScripts:
		content = e.scriptsView()
	case SecAuth:
		content = e.auth.View()
	}

	return lipgloss.JoinVertical(lipgloss.Left, tabRow, divider, padToHeight(content, e.contentH()), e.footer())
}

// contentH is the height the section content must fill: the pane minus
// the tab row, divider, and mode footer.
func (e *Editor) contentH() int {
	h := e.height - 3
	if h < 1 {
		h = 1
	}
	return h
}

// padToHeight appends empty rows so content fills exactly h rows — the
// section widgets must own their height so the mode footer sits at the
// pane bottom instead of hugging the last content line.
func padToHeight(content string, h int) string {
	rows := strings.Count(content, "\n") + 1
	if rows >= h {
		return content
	}
	return content + strings.Repeat("\n", h-rows)
}

// footer is the mode indicator row at the bottom of the editor: the
// active code field's mode, each with its own theme color (NORMAL in
// primary, INSERT in success, VISUAL in warn), empty for the Auth
// section which has no code field.
func (e *Editor) footer() string {
	label := e.ModeLabel()
	if label == "" {
		return ""
	}
	var color lipgloss.AdaptiveColor
	switch e.Mode() {
	case codeeditor.ModeInsert:
		color = e.styles.ColorSuccess
	case codeeditor.ModeNormal:
		color = e.styles.ColorPrimary
	case codeeditor.ModeVisualChar, codeeditor.ModeVisualLine:
		color = e.styles.ColorWarn
	}
	return lipgloss.NewStyle().Foreground(color).Render(label)
}

// scriptsView renders a pre/post toggle row (like the auth type row) with
// only the focused script's editor below it.
func (e *Editor) scriptsView() string {
	toggleRow := e.styles.HintStyle.Render("hook ") + e.styles.TabBar([]string{"pre", "post"}, e.field, max(0, e.width-2), &e.styles.EditorAccent)

	var content string
	if e.field == 1 {
		content = e.post.View()
	} else {
		content = e.pre.View()
	}
	return lipgloss.JoinVertical(lipgloss.Left, toggleRow, content)
}

// Request builds a collection.Request from the editor state. Method and
// URL come from the URLBar; the root model fills them in.
func (e *Editor) Request() *collection.Request {
	req, _ := e.RequestWithError()
	return req
}

// RequestWithError builds a request and rejects malformed header/query lines
// instead of silently dropping user input.
func (e *Editor) RequestWithError() (*collection.Request, error) {
	query, err := parseParams(e.query.Value())
	if err != nil {
		return nil, err
	}
	headers, err := parseHeaders(e.headers.Value())
	if err != nil {
		return nil, err
	}
	return &collection.Request{
		Name:    e.name(),
		Query:   query,
		Headers: headers,
		Auth:    e.auth.Auth(),
		Body:    e.body.Value(),
		Pre:     e.pre.Value(),
		Post:    e.post.Value(),
	}, nil
}

func (e *Editor) name() string {
	if e.title != "" {
		return e.title
	}
	if e.activePath != "" {
		base := e.activePath
		if i := strings.LastIndex(base, "/"); i >= 0 {
			base = base[i+1:]
		}
		return strings.TrimSuffix(base, ".yaml")
	}
	return ""
}

func parseHeaders(s string) ([]collection.Header, error) {
	var out []collection.Header
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		i := strings.Index(line, ":")
		if i <= 0 {
			return nil, fmt.Errorf("invalid header line %q: expected name: value", line)
		}
		name := strings.TrimSpace(line[:i])
		if name == "" {
			return nil, fmt.Errorf("invalid header line %q: name is required", line)
		}
		out = append(out, collection.Header{
			Name:  name,
			Value: strings.TrimSpace(line[i+1:]),
		})
	}
	return out, nil
}

// parseParams parses query params from one "Name: Value" per line,
// mirroring the headers format.
func parseParams(s string) ([]collection.Param, error) {
	var out []collection.Param
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		i := strings.Index(line, ":")
		if i <= 0 {
			return nil, fmt.Errorf("invalid query line %q: expected name: value", line)
		}
		name := strings.TrimSpace(line[:i])
		if name == "" {
			return nil, fmt.Errorf("invalid query line %q: name is required", line)
		}
		out = append(out, collection.Param{
			Name:  name,
			Value: strings.TrimSpace(line[i+1:]),
		})
	}
	return out, nil
}

// SetRequest loads req into the editor. path may be empty for unsaved
// requests. Focus is decided by the caller; widgets are blurred here.
func (e *Editor) SetRequest(req *collection.Request, path string) tea.Cmd {
	e.activePath = path
	e.title = req.Name
	e.pre.SetValue(req.Pre)
	e.post.SetValue(req.Post)

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

	// the section stays where the user left it when switching requests
	e.Blur()
	return nil
}

// New resets the editor to a blank request.
func (e *Editor) New() tea.Cmd {
	e.activePath = ""
	e.title = ""
	e.pre.SetValue("")
	e.post.SetValue("")
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

// Mode returns the editing mode of the code-editor field in the active
// section; the Auth section has no mode.
func (e *Editor) Mode() codeeditor.Mode {
	switch e.section {
	case SecQuery:
		return e.query.Mode()
	case SecHeaders:
		return e.headers.Mode()
	case SecBody:
		return e.body.Mode()
	case SecScripts:
		if e.field == 1 {
			return e.post.Mode()
		}
		return e.pre.Mode()
	}
	return codeeditor.ModeInsert
}

// InVimMode reports whether the active code field is in normal or
// visual mode (insert is plain editing). The model uses it to let q
// quit the app without swallowing the printable q in insert mode.
func (e *Editor) InVimMode() bool { return e.Mode() != codeeditor.ModeInsert }

// ModeLabel is the footer text for the active field's mode; empty for
// the Auth section, which has no mode.
func (e *Editor) ModeLabel() string {
	switch e.section {
	case SecQuery, SecHeaders, SecBody, SecScripts:
	default:
		return ""
	}
	switch e.Mode() {
	case codeeditor.ModeInsert:
		return "—INSERT—"
	case codeeditor.ModeNormal:
		return "—NORMAL—"
	case codeeditor.ModeVisualChar, codeeditor.ModeVisualLine:
		return "—VISUAL—"
	}
	return ""
}

// Section returns the active editor tab, for session persistence.
func (e *Editor) Section() int { return int(e.section) }

// SetSection restores the active editor tab; out-of-range values are
// ignored.
func (e *Editor) SetSection(n int) {
	if n >= 0 && n < len(sectionTabs) {
		e.section = Section(n)
	}
}
