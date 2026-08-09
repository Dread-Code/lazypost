package model

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/cellbuf"

	"lazypost/internal/ui/themes"
)

// geometry derives pane sizes from the terminal size. Both layout() and
// View() use it so panes are sized and rendered from the same numbers
// and the two can't drift.
type geometry struct {
	sidebarW int
	rightW   int
	contentH int
	editorH  int
	respH    int
}

func (m Model) geometry() geometry {
	sidebarW := m.width / 4
	if sidebarW < 24 {
		sidebarW = 24
	}
	if sidebarW > 40 {
		sidebarW = 40
	}
	rightW := m.width - sidebarW - 1
	contentH := m.height - 2 // URL bar + status bar
	editorH := contentH * 55 / 100
	return geometry{
		sidebarW: sidebarW,
		rightW:   rightW,
		contentH: contentH,
		editorH:  editorH,
		respH:    contentH - editorH,
	}
}

// layout recomputes pane geometry from the current terminal size.
func (m *Model) layout() {
	if m.width < 10 || m.height < 6 {
		return
	}
	g := m.geometry()
	// -3 per pane: 2 border rows + 1 title row
	m.urlbar.Resize(m.width)
	m.sidebar.Resize(g.sidebarW-2, g.contentH-3)
	m.editor.Resize(g.rightW-2, g.editorH-3)
	m.response.Resize(g.rightW-2, g.respH-3)
}

// View assembles exactly terminal-height output; taller frames make the
// renderer drop lines (e.g. the title bar).
func (m Model) View() string {
	if m.width < 60 || m.height < 20 {
		return "terminal too small — resize the window"
	}

	g := m.geometry()

	// the URL bar carries the env badge at its right end; the app name is
	// absent from the UI, the file root and version live at the status
	// bar's right
	bar := m.urlbar.View()
	// pad the bar to full width so the frame stays exactly terminal-sized
	if w := lipgloss.Width(bar); w < m.width {
		bar += strings.Repeat(" ", m.width-w)
	}

	sidebar := renderPane("Collection", m.sidebar.View(), &themes.SidebarAccent, m.focus == pSidebar, g.sidebarW, g.contentH, true)

	reqTitle := "Request"
	if p := m.editor.ActivePath(); p != "" {
		reqTitle += " · " + rel(m.dir, p)
	}
	editor := renderPane(reqTitle, m.editor.View(), &themes.EditorAccent, m.focus == pEditor, g.rightW, g.editorH, false)

	respTitle := "Response"
	if s := m.response.StatusLine(); s != "" {
		respTitle += " · " + s
	}
	response := renderPane(respTitle, m.response.View(), &themes.ResponseAccent, m.focus == pResponse, g.rightW, g.respH, false)

	right := lipgloss.JoinVertical(lipgloss.Left, editor, response)
	content := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, right)

	status := m.statusBar()
	frame := lipgloss.JoinVertical(lipgloss.Left, bar, content, status)
	switch m.overlay {
	case ovPalette:
		title := "Command palette"
		if m.palette.theme {
			title = "Switch theme"
		}
		content := m.palette.widget.View()
		if m.palette.theme {
			content += "\n" + themes.KeyHint("↑↓", "preview", "enter", "apply", "esc", "cancel")
		} else {
			content += "\n" + themes.KeyHint("↑↓", "navigate", "enter", "run", "esc", "close")
		}
		frame = overlayPalette(frame, title, content, m.width, m.height)
	case ovEnv:
		frame = overlayPalette(frame, "Environments", m.envManagerView(), m.width, m.height)
	case ovNamer:
		frame = overlayPalette(frame, m.namer.widget.Label(), m.namer.widget.View(), m.width, m.height)
	case ovConfirm:
		frame = overlayPalette(frame, m.confirm.widget.Label(), m.confirm.widget.View(), m.width, m.height)
	case ovHistory:
		content := m.historyWidget.View() + "\n" + themes.KeyHint("enter", "restore", "ctrl+r", "resend", "esc", "close")
		frame = overlayPalette(frame, "Request history", content, m.width, m.height)
	case ovHelp:
		frame = overlayPalette(frame, "Keybindings", helpContent(m.width-8), m.width, m.height)
	}
	return frame
}

// dimFrame wraps a rendered frame in the faint attribute so an open modal
// pops against it. Inner color codes only set attributes (never reset), so
// the dim survives until the trailing reset.
func dimFrame(frame string) string {
	return "\x1b[2m" + frame + "\x1b[0m"
}

// overlayPalette draws the modal box over the dimmed frame with an
// ANSI-aware cell buffer, so the frame content beside the box survives
// as a faded backdrop. The box hugs its content width and drops from the
// top (like a quick-open), so it never cuts the panes in two or sprawls
// across the pane borders.
func overlayPalette(frame, title, paletteView string, termW, termH int) string {
	box := "\x1b[0m" + renderModal(title, paletteView)
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
	cellbuf.SetContent(buf, dimFrame(frame))

	start := (termH - boxH) / 2
	if start < 1 {
		start = 1
	}
	if start+boxH > termH {
		// a modal taller than the terminal pins to the top so its head
		// (title + first sections) stays visible
		if boxH > termH {
			start = 0
		} else {
			start = termH - boxH
		}
	}
	cellbuf.SetContentRect(buf, box, cellbuf.Rect(pad, start, boxW, boxH))
	return strings.ReplaceAll(cellbuf.Render(buf), "\r\n", "\n")
}

// legendAlign is where a fieldset legend sits on the top border.
type legendAlign int

const (
	alignLeft legendAlign = iota
	alignCenter
	alignRight
)

// legendLine builds a fieldset-style top border: the title sits on the
// border, split into two dash runs (1 col each for the corners) at the
// chosen alignment. titleStyle styles the legend text (muted for resting
// panes, primary for the focused pane and modals).
func legendLine(border lipgloss.Border, borderStyle, titleStyle lipgloss.Style, title string, w int, align legendAlign) string {
	titleStyled := titleStyle.Render(" " + title + " ")
	titleW := lipgloss.Width(titleStyled)
	avail := (w - 2) - titleW
	if avail < 0 {
		avail = 0
	}
	var left, right int
	switch align {
	case alignLeft:
		left = 1
		right = avail - 1
	case alignRight:
		left = avail - 1
		right = 1
	default: // alignCenter
		left = avail / 2
		right = avail - avail/2
	}
	if left < 0 {
		left = 0
	}
	if right < 0 {
		right = 0
	}
	return borderStyle.Render(border.TopLeft+strings.Repeat(border.Top, left)) +
		titleStyled +
		borderStyle.Render(strings.Repeat(border.Top, right)+border.TopRight)
}

// renderModal draws a modal box whose title sits right-aligned on the top
// border, matching the pane legend style ([[Design - command palette]] and
// friends). The box hugs the wider of the content or the title; content
// gets one column of breathing room inside the border.
func renderModal(title, content string) string {
	contentW := 0
	for _, l := range strings.Split(content, "\n") {
		if w := lipgloss.Width(l); w > contentW {
			contentW = w
		}
	}
	// modals sit on top of the frame, so they always wear the focused look
	style := themes.ModalStyle
	border := style.GetBorderStyle()
	borderStyle := lipgloss.NewStyle()
	if c := style.GetBorderTopForeground(); c != nil {
		borderStyle = borderStyle.Foreground(c)
	}
	titleW := lipgloss.Width(lipgloss.NewStyle().Bold(true).Foreground(themes.ColorMuted).Render(" " + title + " "))
	// room for the title (with its own padding) plus a dash on each side
	boxW := max(contentW+4, titleW+4)
	topLine := legendLine(border, borderStyle, themes.ActiveLegendTitleStyle, title, boxW, alignRight)
	body := style.Border(border, false, true, true, true).
		Padding(0, 1).
		Width(boxW - 2).
		Render(content)
	return lipgloss.JoinVertical(lipgloss.Left, topLine, body)
}

// renderPane draws a fieldset-style pane: the title sits on the top
// border, which splits into two dash runs around it (like an HTML
// fieldset legend). accent is the pane's section hue: the legend title
// always wears it (section identity), and the border wears it while the
// pane is focused (muted when resting). The active tab inside the pane
// also always wears the accent. The legend hugs the left when legendLeft
// is set, otherwise it's centered. lipgloss's top border is disabled and
// rebuilt by hand so the title shares the border line.
func renderPane(title, content string, accent *themes.PaneAccent, focused bool, w, h int, legendLeft bool) string {
	style := themes.PaneStyle
	legendTitle := accent.Legend
	if focused {
		style = accent.Active
	}
	title = themes.TruncateRunes(title, w-4)

	border := style.GetBorderStyle()
	borderStyle := lipgloss.NewStyle()
	if c := style.GetBorderTopForeground(); c != nil {
		borderStyle = borderStyle.Foreground(c)
	}
	align := alignCenter
	if legendLeft {
		align = alignLeft
	}
	topLine := legendLine(border, borderStyle, legendTitle, title, w, align)

	// body: no top border (the legend line takes its place)
	body := style.Border(border, false, true, true, true).Width(w - 2).Height(h - 2).Render(content)
	return lipgloss.JoinVertical(lipgloss.Left, topLine, body)
}

func (m Model) statusBar() string {
	var help string
	switch m.focus {
	case pSidebar:
		help = themes.KeyHint("↑↓ ctrl+n/p", "nav loads", "enter", "url", "a", "add", "d", "del", "r", "rename", "?", "help")
	case pBar:
		help = themes.KeyHint("ctrl+t", "method", "enter", "send", "esc", "back")
	case pEditor:
		help = themes.KeyHint("ctrl+n/p", "field", "alt+←→", "tab", "ctrl+t", "auth type", "ctrl+s", "save", "ctrl+r", "send")
	case pResponse:
		help = themes.KeyHint("←→ or b/h", "tabs", "↑↓", "scroll", "?", "help", "q", "quit")
	}

	// right side: the file root and version sit at the far right,
	// transient notices to their left, so the identity stays put while
	// notices come and go
	right := themes.HintStyle.Render(themes.TruncateRunes(m.collectionTitle(), m.width/4))
	if m.version != "" {
		right += "  " + themes.VersionStyle.Render(m.version)
	}
	if m.notice != "" {
		text := themes.TruncateRunes(m.notice, m.width/3)
		if m.noticeError {
			text = "✖ " + text
			right = themes.ErrorStyle.Render(text) + "  " + right
		} else {
			text = "✓ " + text
			right = themes.NoticeStyle.Render(text) + "  " + right
		}
	}
	limit := m.width - lipgloss.Width(right) - 1
	if limit < 1 {
		limit = 1
	}
	left := ansi.Truncate(help, limit, "…")

	gap := m.width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		gap = 1
	}
	return left + strings.Repeat(" ", gap) + right
}

// collectionTitle is what the title bar shows for the collection: the
// .lazypost marker name when one exists, else the root path
// ([[Design - collection marker file]]).
func (m *Model) collectionTitle() string {
	if m.collectionName != "" {
		return m.collectionName
	}
	return m.dir
}

func rel(dir, path string) string {
	return strings.TrimPrefix(path, dir+"/")
}
