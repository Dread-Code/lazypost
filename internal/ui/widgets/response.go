package ui

import (
	"encoding/json"
	"strings"

	"github.com/Dread-Code/lazypost/lib/codeeditor"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Dread-Code/lazypost/internal/httpclient"
	"github.com/Dread-Code/lazypost/internal/render"

	"github.com/Dread-Code/lazypost/internal/ui/themes"
)

type respState int

const (
	stIdle respState = iota
	stLoading
	stDone
	stError
)

var respTabs = []string{"Body", "Headers"}

type Response struct {
	state   respState
	spinner spinner.Model
	res     *httpclient.Response
	err     error
	tab     int
	body    viewport.Model
	headers viewport.Model
	focused bool
	width   int
	height  int
	styles  themes.Styles
}

// NewResponse wires a dot spinner plus two viewports (body, headers);
// only the active tab's viewport is rendered and scrollable.
func NewResponse(width, height int) *Response {
	styles := themes.NewStyles(themes.DefaultTheme)
	r := &Response{width: width, height: height, styles: styles}
	r.spinner = spinner.New()
	r.spinner.Spinner = spinner.Dot
	r.spinner.Style = lipgloss.NewStyle().Foreground(styles.ColorPrimary)
	r.body = viewport.New(width, height)
	r.headers = viewport.New(width, height)
	r.resize()
	return r
}

// RefreshTheme rebuilds cached response content with the current theme while
// preserving viewport offsets.
func (r *Response) RefreshTheme() {
	r.SetStyles(themes.NewStyles(themes.DefaultTheme))
}

// SetStyles replaces the rendering snapshot and refreshes cached response
// content without resetting viewport offsets.
func (r *Response) SetStyles(styles themes.Styles) {
	r.styles = styles
	r.spinner.Style = lipgloss.NewStyle().Foreground(styles.ColorPrimary)
	r.refreshContent()
}

func (r *Response) resize() {
	w := r.width - 4
	h := r.height - 6 // tab row + divider + borders
	if w < 4 {
		w = 4
	}
	if h < 1 {
		h = 1
	}
	r.body.Width = w
	r.body.Height = h
	r.headers.Width = w
	r.headers.Height = h
}

func (r *Response) Resize(width, height int) {
	r.width, r.height = width, height
	r.resize()
	r.refreshContent()
}

func (r *Response) Focus() { r.focused = true }
func (r *Response) Blur()  { r.focused = false }

func (r *Response) StartLoading() tea.Cmd {
	r.state = stLoading
	r.res = nil
	r.err = nil
	return r.spinner.Tick
}

// maxHighlightBody skips coloring above this raw body size; the pretty
// JSON is still shown, just uncolored.
const maxHighlightBody = 512 << 10 // 512 KiB

// highlightJSONColors maps JSON token kinds onto the active theme. Styles
// are built from the package color vars, so a runtime theme switch is
// picked up on the next response.
func highlightJSONColors(styles themes.Styles, kind render.Kind, lit string) string {
	var color lipgloss.AdaptiveColor
	switch kind {
	case render.KindKey:
		color = styles.ColorPrimary
	case render.KindString:
		color = styles.ColorSuccess
	case render.KindNumber:
		color = styles.ColorInfo
	case render.KindLiteral:
		color = styles.ColorWarn
	default: // KindPunctuation
		color = styles.ColorMuted
	}
	return lipgloss.NewStyle().Foreground(color).Render(lit)
}

// highlightedBody prepares the body text for the viewport: color the full
// formatted body first (validity is judged on untruncated input, so long
// lines can't break the tokenizer), then truncate ANSI-aware so escape
// sequences are never clipped. The stored body stays raw either way.
func highlightedBody(res *httpclient.Response, w int) string {
	return highlightedBodyWithStyles(res, w, themes.NewStyles(themes.DefaultTheme))
}

func highlightedBodyWithStyles(res *httpclient.Response, w int, styles themes.Styles) string {
	formatted := res.FormattedBody()
	if len(res.Body) <= maxHighlightBody && json.Valid([]byte(formatted)) {
		return truncateLines(render.HighlightJSON(formatted, func(kind render.Kind, lit string) string {
			return highlightJSONColors(styles, kind, lit)
		}), w)
	}
	return truncateLines(formatted, w)
}

func (r *Response) SetResponse(res *httpclient.Response) {
	r.state = stDone
	r.res = res
	r.refreshContent()
	r.body.GotoTop()
	r.headers.GotoTop()
}

func (r *Response) refreshContent() {
	if r.res == nil {
		return
	}
	w := r.width - 6 // border + viewport scrollbar room
	r.body.SetContent(highlightedBodyWithStyles(r.res, w, r.styles))
	r.headers.SetContent(truncateLines(r.res.FormattedHeaders(), w))
}

func truncateLines(s string, w int) string {
	if w < 4 {
		w = 4
	}
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = codeeditor.TruncateRunesAnsi(l, w)
	}
	return strings.Join(lines, "\n")
}

func (r *Response) SetError(err error) {
	r.state = stError
	r.err = err
}

func (r *Response) Update(msg tea.Msg) (*Response, tea.Cmd) {
	if km, ok := msg.(tea.KeyMsg); ok && r.focused {
		switch km.String() {
		// letters chosen because the viewports own the arrow keys
		case "left", "b":
			r.tab = 0
			return r, nil
		case "right", "h":
			r.tab = 1
			return r, nil
		}
	}
	// spinner ticks must keep ticking even while the pane is unfocused
	switch r.state {
	case stLoading:
		var cmd tea.Cmd
		r.spinner, cmd = r.spinner.Update(msg)
		return r, cmd
	case stDone:
		if r.focused && r.tab == 0 {
			var cmd tea.Cmd
			r.body, cmd = r.body.Update(msg)
			return r, cmd
		}
		if r.focused && r.tab == 1 {
			var cmd tea.Cmd
			r.headers, cmd = r.headers.Update(msg)
			return r, cmd
		}
	}
	return r, nil
}

func (r *Response) View() string {
	var content string
	switch r.state {
	case stIdle:
		content = r.center(r.styles.HintStyle.Render("press ") + r.styles.KeyHint("ctrl+r", "to send the request"))
	case stLoading:
		content = r.center(r.spinner.View() + " sending...")
	case stError:
		content = r.center(r.styles.ErrorStyle.Render("error: " + r.err.Error()))
	case stDone:
		if r.tab == 0 {
			content = r.body.View()
		} else {
			content = r.headers.View()
		}
	}
	divider := r.styles.Rule(max(0, r.width-4))
	return lipgloss.JoinVertical(lipgloss.Left, r.styles.TabBar(respTabs, r.tab, max(0, r.width-4), &r.styles.ResponseAccent), divider, content)
}

// center places the idle/loading/error placeholder in the middle of the
// pane so the empty response reads as a state, not a missing feature.
func (r *Response) center(content string) string {
	w := r.width - 4
	h := r.height - 6 // tab row + divider + borders
	if w < 4 {
		w = 4
	}
	if h < 1 {
		h = 1
	}
	return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, content)
}

// StatusLine renders the response summary for the pane title. The
// executed URL lives in the Headers tab, not here ([[ADR-0014 Response
// shows the executed URL]]).
func (r *Response) StatusLine() string {
	if r.state != stDone || r.res == nil {
		return ""
	}
	style := lipgloss.NewStyle().Foreground(r.styles.StatusColor(r.res.StatusCode)).Bold(true)
	return style.Render(r.res.Summary())
}

func (r *Response) Loading() bool { return r.state == stLoading }
