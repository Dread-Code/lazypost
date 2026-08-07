package model

import (
	neturl "net/url"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/cellbuf"

	"postgo/internal/collection"
	"postgo/internal/ui"
)

// layout recomputes pane geometry from the current terminal size. It
// mirrors the arithmetic in View() so panes are sized and rendered from
// the same numbers.
func (m *Model) layout() {
	if m.width < 10 || m.height < 6 {
		return
	}
	sidebarW := m.width / 4
	if sidebarW < 24 {
		sidebarW = 24
	}
	if sidebarW > 40 {
		sidebarW = 40
	}
	rightW := m.width - sidebarW - 1
	contentH := m.height - 3 // title bar + URL bar + status bar

	editorH := contentH * 55 / 100
	respH := contentH - editorH

	// -3 per pane: 2 border rows + 1 title row
	m.urlbar.Resize(m.width)
	m.sidebar.Resize(sidebarW-2, contentH-3)
	m.editor.Resize(rightW-2, editorH-3)
	m.response.Resize(rightW-2, respH-3)
}

// View assembles exactly terminal-height output; taller frames make the
// renderer drop lines (e.g. the title bar).
func (m Model) View() string {
	if m.width < 60 || m.height < 20 {
		return "terminal too small — resize to use postgo"
	}

	sidebarW := m.width / 4
	if sidebarW < 24 {
		sidebarW = 24
	}
	if sidebarW > 40 {
		sidebarW = 40
	}
	rightW := m.width - sidebarW - 1
	contentH := m.height - 3
	editorH := contentH * 55 / 100
	respH := contentH - editorH

	title := lipgloss.JoinHorizontal(lipgloss.Left,
		ui.TitleStyle.Render("postgo"),
		ui.HintStyle.Render("  "+ui.TruncateRunes(m.dir, m.width/2)),
	)
	envLabel := "env: none"
	if name := m.activeEnvName(); name != "" {
		envLabel = "env: " + name
	}
	envStyle := ui.HintStyle
	if m.activeEnvName() != "" {
		envStyle = lipgloss.NewStyle().Foreground(ui.ColorInfo)
	}
	gap := m.width - lipgloss.Width(title) - lipgloss.Width(envLabel)
	if gap < 1 {
		gap = 1
	}
	titleBar := title + strings.Repeat(" ", gap) + envStyle.Render(envLabel)

	bar := m.urlbar.View()
	// pad the bar to full width so the frame stays exactly terminal-sized
	if w := lipgloss.Width(bar); w < m.width {
		bar += strings.Repeat(" ", m.width-w)
	}

	sidebar := renderPane("Collection", m.sidebar.View(), m.focus == pSidebar, sidebarW, contentH, true)

	reqTitle := "Request"
	if p := m.editor.ActivePath(); p != "" {
		reqTitle += " · " + rel(m.dir, p)
	}
	editor := renderPane(reqTitle, m.editor.View(), m.focus == pEditor, rightW, editorH, false)

	respTitle := "Response"
	if s := m.response.StatusLine(); s != "" {
		respTitle += " · " + s
	}
	response := renderPane(respTitle, m.response.View(), m.focus == pResponse, rightW, respH, false)

	right := lipgloss.JoinVertical(lipgloss.Left, editor, response)
	content := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, right)

	status := m.statusBar()
	frame := lipgloss.JoinVertical(lipgloss.Left, titleBar, bar, content, status)
	if m.paletteOpen {
		frame = overlayPalette(frame, m.palette.View(), m.width, m.height)
	}
	if m.namerOpen {
		frame = overlayPalette(frame, m.namer.View(), m.width, m.height)
	}
	return frame
}

// overlayPalette draws the palette box over the frame with an ANSI-aware
// cell buffer, so the frame content beside the box survives. The box hugs
// its content width and drops from the top (like a quick-open), so it never
// cuts the panes in two or sprawls across the pane borders.
func overlayPalette(frame, paletteView string, termW, termH int) string {
	contentW := 0
	for _, l := range strings.Split(paletteView, "\n") {
		if w := lipgloss.Width(l); w > contentW {
			contentW = w
		}
	}
	// lipgloss Width(w) is the *content* width; the two border columns are
	// added on top, so the rendered box is contentW+2 wide.
	box := ui.PaneStyle.Width(contentW).Render(paletteView)
	boxLines := strings.Split(box, "\n")
	boxW := lipgloss.Width(boxLines[0])
	boxH := len(boxLines)
	if boxW+4 >= termW {
		boxW = termW - 4
	}

	pad := (termW - boxW) / 2
	if pad < 0 {
		pad = 0
	}

	buf := cellbuf.NewBuffer(termW, termH)
	cellbuf.SetContent(buf, frame)

	start := (termH - boxH) / 2
	if start < 1 {
		start = 1
	}
	if start+boxH > termH {
		start = termH - boxH
	}
	cellbuf.SetContentRect(buf, box, cellbuf.Rect(pad, start, boxW, boxH))
	return strings.ReplaceAll(cellbuf.Render(buf), "\r\n", "\n")
}

// renderPane draws a fieldset-style pane: the title sits on the top
// border, which splits into two dash runs around it (like an HTML
// fieldset legend). The legend hugs the left when legendLeft is set,
// otherwise it's centered. lipgloss's top border is disabled and rebuilt
// by hand so the title shares the border line.
func renderPane(title, content string, focused bool, w, h int, legendLeft bool) string {
	style := ui.PaneStyle
	if focused {
		style = ui.ActivePaneStyle
	}
	title = ui.TruncateRunes(title, w-4)

	border := style.GetBorderStyle()
	borderStyle := lipgloss.NewStyle()
	if c := style.GetBorderTopForeground(); c != nil {
		borderStyle = borderStyle.Foreground(c)
	}
	titleStyled := lipgloss.NewStyle().Bold(true).Foreground(ui.ColorMuted).Render(" " + title + " ")
	titleW := lipgloss.Width(titleStyled)

	// reserve the label width; split the rest of the top edge into two
	// dash runs (1 col each for the corners). legendLeft puts the label at
	// the front with all the slack on the right.
	avail := (w - 2) - titleW
	if avail < 0 {
		avail = 0
	}
	var left, right int
	if legendLeft {
		left = 1
		right = avail - 1
	} else {
		left = avail / 2
		right = avail - avail/2
	}
	topLine := borderStyle.Render(border.TopLeft+strings.Repeat(border.Top, left)) +
		titleStyled +
		borderStyle.Render(strings.Repeat(border.Top, right)+border.TopRight)

	// body: no top border (the legend line takes its place)
	body := style.Border(border, false, true, true, true).Width(w - 2).Height(h - 2).Render(content)
	return lipgloss.JoinVertical(lipgloss.Left, topLine, body)
}

func (m Model) statusBar() string {
	var help string
	switch m.focus {
	case pSidebar:
		help = "↑↓ navigate · enter load · a add · n new · ctrl+e env · ctrl+l url · ctrl+/ palette · ctrl+r send · q quit"
	case pBar:
		help = "ctrl+t method · enter send · esc back · ctrl+/ palette · ctrl+g export curl · ctrl+r send"
	case pEditor:
		help = "ctrl+n/p field · alt+←→ tab · ctrl+t auth type · ctrl+/ palette · ctrl+s save · ctrl+r send"
	case pResponse:
		help = "←→ or b/h tabs · ↑↓ scroll · ctrl+/ palette · ctrl+g curl · q quit"
	}

	right := ""
	if m.notice != "" {
		if m.noticeError {
			right = ui.ErrorStyle.Render(ui.TruncateRunes(m.notice, m.width/3))
		} else {
			right = ui.NoticeStyle.Render(ui.TruncateRunes(m.notice, m.width/3))
		}
	}
	left := ui.HintStyle.Render(ui.TruncateRunes(help, m.width-lipgloss.Width(right)-1))

	gap := m.width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		gap = 1
	}
	return left + strings.Repeat(" ", gap) + right
}

func rel(dir, path string) string {
	return strings.TrimPrefix(path, dir+"/")
}

// defaultName derives a request name from the last URL path segment when
// the user hasn't provided one (e.g. ".../posts/42" → "42").
func defaultName(rawURL string) string {
	u, err := neturl.Parse(rawURL)
	if err == nil && u.Path != "" && u.Path != "/" {
		segs := strings.Split(strings.Trim(u.Path, "/"), "/")
		last := segs[len(segs)-1]
		if last != "" {
			return collection.Slug(last)
		}
	}
	return "untitled"
}
