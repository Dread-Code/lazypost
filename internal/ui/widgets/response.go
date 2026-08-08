package ui

import (
	"encoding/json"
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"lazypost/internal/httpclient"
	"lazypost/internal/render"
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
}

// NewResponse wires a dot spinner plus two viewports (body, headers);
// only the active tab's viewport is rendered and scrollable.
func NewResponse(width, height int) *Response {
	r := &Response{width: width, height: height}
	r.spinner = spinner.New()
	r.spinner.Spinner = spinner.Dot
	r.spinner.Style = lipgloss.NewStyle().Foreground(ColorPrimary)
	r.body = viewport.New(width, height)
	r.headers = viewport.New(width, height)
	r.resize()
	return r
}

func (r *Response) resize() {
	w := r.width - 4
	h := r.height - 5
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
func highlightJSONColors(kind render.Kind, lit string) string {
	var color lipgloss.AdaptiveColor
	switch kind {
	case render.KindKey:
		color = ColorPrimary
	case render.KindString:
		color = ColorSuccess
	case render.KindNumber:
		color = ColorInfo
	case render.KindLiteral:
		color = ColorWarn
	default: // KindPunctuation
		color = ColorMuted
	}
	return lipgloss.NewStyle().Foreground(color).Render(lit)
}

// highlightedBody prepares the body text for the viewport: color the full
// formatted body first (validity is judged on untruncated input, so long
// lines can't break the tokenizer), then truncate ANSI-aware so escape
// sequences are never clipped. The stored body stays raw either way.
func highlightedBody(res *httpclient.Response, w int) string {
	formatted := res.FormattedBody()
	if len(res.Body) <= maxHighlightBody && json.Valid([]byte(formatted)) {
		return truncateLines(render.HighlightJSON(formatted, highlightJSONColors), w)
	}
	return truncateLines(formatted, w)
}

func (r *Response) SetResponse(res *httpclient.Response) {
	r.state = stDone
	r.res = res
	w := r.width - 6 // border + viewport scrollbar room
	r.body.SetContent(highlightedBody(res, w))
	r.body.GotoTop()
	r.headers.SetContent(truncateLines(res.FormattedHeaders(), w))
	r.headers.GotoTop()
}

func truncateLines(s string, w int) string {
	if w < 4 {
		w = 4
	}
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = truncateRunesAnsi(l, w)
	}
	return strings.Join(lines, "\n")
}

// truncateRunesAnsi shortens s to at most n visible runes, appending an
// ellipsis when truncated. ANSI escape sequences count as zero width and
// are copied whole, so colored content is never clipped mid-sequence.
func truncateRunesAnsi(s string, n int) string {
	if n <= 0 {
		return ""
	}
	if n == 1 {
		return "…"
	}
	width, cut := 0, 0
	need := n - 1
	for i := 0; i < len(s); {
		if s[i] == 0x1b {
			j := i + 1
			if j < len(s) && s[j] == '[' {
				j++ // CSI intro
				for j < len(s) && s[j] >= 0x20 && s[j] <= 0x3f {
					j++ // parameters + intermediates
				}
			}
			if j < len(s) && s[j] >= 0x40 && s[j] <= 0x7e {
				j++ // final byte
			}
			i = j
			continue
		}
		_, size := utf8.DecodeRuneInString(s[i:])
		width++
		if width == need {
			cut = i + size
		}
		i += size
	}
	if width <= n {
		return s
	}
	truncated := s[:cut]
	if strings.Contains(truncated, "\x1b[") {
		// the cut dropped the token's reset; append one so the next
		// line never inherits a stale color
		return truncated + "…\x1b[0m"
	}
	return truncated + "…"
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
		content = HintStyle.Render("press ctrl+r to send the request")
	case stLoading:
		content = r.spinner.View() + " sending..."
	case stError:
		content = ErrorStyle.Render("error: " + r.err.Error())
	case stDone:
		if r.tab == 0 {
			content = r.body.View()
		} else {
			content = r.headers.View()
		}
	}
	return lipgloss.JoinVertical(lipgloss.Left, TabBar(respTabs, r.tab), content)
}

// StatusLine renders the response summary for the pane title.
func (r *Response) StatusLine() string {
	if r.state != stDone || r.res == nil {
		return ""
	}
	style := lipgloss.NewStyle().Foreground(StatusColor(r.res.StatusCode)).Bold(true)
	return style.Render(r.res.Summary())
}

func (r *Response) Loading() bool { return r.state == stLoading }
