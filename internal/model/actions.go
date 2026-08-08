package model

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"postgo/internal/clipboard"
	"postgo/internal/collection"
	"postgo/internal/curl"
	"postgo/internal/httpclient"
	"postgo/internal/render"
	"postgo/internal/script"
	"postgo/internal/session"
	"postgo/internal/ui"
)

// Action is one selectable command: the palette lists all of them, and
// globalActions are also bound to their shortcut keys. It is the single
// source of truth for global keybindings ([[Design - command palette]]).
type Action struct {
	Title    string
	Shortcut string
	Keys     []string
	Run      func(m *Model) (tea.Model, tea.Cmd)
}

// globalActions are the key-bound commands handled by the root model
// before any pane sees a key.
var globalActions = []Action{
	{Title: "Send request", Shortcut: "ctrl+r", Keys: []string{"ctrl+r"}, Run: func(m *Model) (tea.Model, tea.Cmd) { return m.send() }},
	{Title: "Save request", Shortcut: "ctrl+s", Keys: []string{"ctrl+s"}, Run: func(m *Model) (tea.Model, tea.Cmd) { return m.save() }},
	{Title: "Cycle environment", Shortcut: "ctrl+e", Keys: []string{"ctrl+e"}, Run: func(m *Model) (tea.Model, tea.Cmd) { m.cycleEnv(); return m, m.saveState() }},
	{Title: "Focus URL bar", Shortcut: "ctrl+l", Keys: []string{"ctrl+l"}, Run: func(m *Model) (tea.Model, tea.Cmd) { return m, m.enter(pBar) }},
	{Title: "Copy as curl", Shortcut: "ctrl+g", Keys: []string{"ctrl+g"}, Run: func(m *Model) (tea.Model, tea.Cmd) { return m.exportCurl() }},
	{Title: "Quit", Shortcut: "ctrl+c", Keys: []string{"ctrl+c"}, Run: func(m *Model) (tea.Model, tea.Cmd) { return m, m.quit() }},
}

// paletteActions returns every command the palette offers: the global
// actions plus navigation commands that are only reachable from it.
func (m *Model) paletteActions() []Action {
	return append(globalActions,
		Action{Title: "New request", Run: func(m *Model) (tea.Model, tea.Cmd) {
			m.urlbar.New()
			return m, tea.Batch(m.editor.New(), m.enter(pBar))
		}},
		Action{Title: "Focus editor", Run: func(m *Model) (tea.Model, tea.Cmd) { return m, m.enter(pEditor) }},
		Action{Title: "Focus response", Run: func(m *Model) (tea.Model, tea.Cmd) { return m, m.enter(pResponse) }},
		Action{Title: "Clear chain store", Run: func(m *Model) (tea.Model, tea.Cmd) {
			m.store = map[string]string{}
			m.setNotice("chain store cleared", false)
			return m, nil
		}},
		Action{Title: "Switch theme", Run: func(m *Model) (tea.Model, tea.Cmd) {
			return m.openThemePicker()
		}},
		Action{Title: "Environments", Run: func(m *Model) (tea.Model, tea.Cmd) {
			return m.openEnvManager()
		}},
	)
}

// openThemePicker reopens the palette as a theme list; enter applies the
// selected theme and persists it in session state.
func (m *Model) openThemePicker() (tea.Model, tea.Cmd) {
	names := ui.ThemeNames()
	items := make([]ui.PaletteItem, len(names))
	for i, n := range names {
		items[i] = ui.PaletteItem{Title: n}
	}
	m.palette.SetItems(items)
	w := m.paletteWidth(items)
	if w < 28 {
		w = 28
	}
	m.palette.Resize(w, m.minPaletteHeight(len(items)))
	m.palette.Open()
	m.palettePrev = m.focus
	m.paletteTheme = true
	m.paletteOpen = true
	return m, nil
}

// applySelectedTheme applies the theme highlighted in the picker and
// persists it in session state.
func (m *Model) applySelectedTheme() (tea.Model, tea.Cmd) {
	it := m.palette.Selected()
	if it == nil {
		return m, nil
	}
	m.paletteOpen = false
	m.paletteTheme = false
	name := it.Title
	ui.ThemeByName(name).Apply()
	m.state.Theme = name
	m.setNotice("theme: "+name, false)
	return m, m.saveState()
}

// openEnvManager shows the environment manager: a tab bar of environments
// with the active tab's variables listed below. ctrl+e cycles the tab,
// enter activates it, a/r/d edit variables ([[Design - environment
// manager modal]]).
func (m *Model) openEnvManager() (tea.Model, tea.Cmd) {
	if len(m.envNames) == 0 {
		m.setNotice("no environments", true)
		return m, nil
	}
	if !m.envManagerOpen {
		// open on the active environment if set, else the first
		m.envTab = m.envIdx - 1
		if m.envTab < 0 {
			m.envTab = 0
		}
	}
	m.envManagerOpen = true
	m.envFiltering = false
	m.loadEnvTab()
	m.palettePrev = m.focus
	return m, nil
}

// envTabName returns the environment for the current tab.
func (m *Model) envTabName() string {
	if m.envTab < 0 || m.envTab >= len(m.envNames) {
		return ""
	}
	return m.envNames[m.envTab]
}

// loadEnvTab populates the palette with the current tab's variables.
func (m *Model) loadEnvTab() {
	items := []ui.PaletteItem{}
	if env := m.envTabName(); env != "" {
		for k, v := range m.envs[env] {
			items = append(items, ui.PaletteItem{Title: k + " = " + v})
		}
	}
	m.palette.SetItems(items)
	w := m.paletteWidth(items)
	if w < 50 {
		w = 50
	}
	m.palette.Resize(w, m.minPaletteHeight(len(items)+2)) // tab row + filter
	if m.envFiltering {
		m.palette.StartFiltering()
	} else {
		m.palette.OpenBrowsing()
	}
}

// updateEnvManager routes keys while the environment manager is open:
// ctrl+e cycles tabs, enter activates the tab, a/r/d edit variables,
// "/" starts filtering (so letters search instead of acting), esc/q
// closes.
func (m *Model) updateEnvManager(msg tea.Msg) (tea.Model, tea.Cmd) {
	if km, ok := msg.(tea.KeyMsg); ok {
		// While filtering, all letters go to the filter: only esc/q act.
		if m.envFiltering {
			switch km.String() {
			case "esc", "q":
				m.envFiltering = false
				m.palette.ClearFilter()
				return m, nil
			}
			cmd, _ := m.palette.Update(msg)
			return m, cmd
		}

		switch {
		case km.String() == "esc" || km.String() == "q":
			m.envManagerOpen = false
			return m, m.enter(m.palettePrev)

		case km.String() == "/":
			// enter filter mode: letters now search the variables
			m.envFiltering = true
			m.palette.StartFiltering()
			return m, nil

		// bubbles routes every key to the filter input in Filtering state
		// and disables its nav bindings, so move the cursor ourselves.
		case km.String() == "up" || km.String() == "ctrl+p":
			m.palette.CursorUp()
			return m, nil
		case km.String() == "down" || km.String() == "ctrl+n":
			m.palette.CursorDown()
			return m, nil

		// cycle the environment tab (ctrl+e is free here: the manager is
		// modal, so the global cycle key is intercepted)
		case km.String() == "ctrl+e":
			m.envTab = (m.envTab + 1) % len(m.envNames)
			m.loadEnvTab()
			return m, nil

		case km.String() == "a":
			if env := m.envTabName(); env != "" {
				m.namerEnvEditVar = env
				m.namerOpen = true
				return m, m.namer.Open()
			}
			return m, nil

		case km.String() == "r":
			if env := m.envTabName(); env != "" {
				if key := m.selectedVarName(); key != "" {
					m.namerEnvEditVar = env
					m.namerOpen = true
					return m, m.namer.OpenPrefilled(key + "=" + m.envs[env][key])
				}
			}
			return m, nil

		case km.String() == "d":
			if env := m.envTabName(); env != "" {
				if key := m.selectedVarName(); key != "" {
					m.confirm.Ask("delete variable " + ui.TruncateRunes(key, 30) + "?")
					m.confirmOpen = true
					m.confirmVarEnv = env
					m.confirmVarKey = key
				}
			}
			return m, nil

		case key.Matches(km, m.keyEnter):
			m.setEnv(m.envTabName())
			m.envManagerOpen = false
			return m, m.saveState()
		}
	}
	cmd, _ := m.palette.Update(msg)
	return m, cmd
}

// selectedVarName returns the highlighted variable's key (the part before
// " = ").
func (m *Model) selectedVarName() string {
	it := m.palette.Selected()
	if it == nil {
		return ""
	}
	if k, _, ok := strings.Cut(it.Title, " = "); ok {
		return k
	}
	return ""
}

// setEnv activates env by name ("" = none), re-resolving envIdx.
func (m *Model) setEnv(name string) {
	if name == "" {
		m.envIdx = 0
		return
	}
	if idx := indexOf(name, m.envNames); idx >= 0 {
		m.envIdx = idx + 1
	} else {
		m.envIdx = 0
	}
}

// reloadEnvs re-reads environments from disk and re-resolves envIdx so a
// rename/delete of the active env falls back to none.
func (m *Model) reloadEnvs() {
	envs, names, err := collection.LoadEnvironments(m.dir)
	if err != nil {
		m.setNotice("reload environments: "+err.Error(), true)
		return
	}
	active := m.activeEnvName()
	m.envs = envs
	m.envNames = names
	m.setEnv(active)
}

// setEnvironmentVar writes key=value into an environment's map, persists
// it, and reopens the env manager (now on that env's tab).
func (m *Model) setEnvironmentVar(env, kv string) (tea.Model, tea.Cmd) {
	key, val, ok := strings.Cut(kv, "=")
	if !ok || strings.TrimSpace(key) == "" {
		m.setNotice("expected key=value", true)
		return m, nil
	}
	vars := m.envs[env]
	if vars == nil {
		vars = map[string]string{}
	}
	vars[strings.TrimSpace(key)] = strings.TrimSpace(val)
	if err := collection.SaveEnvironment(m.dir, env, vars); err != nil {
		m.setNotice("edit environment: "+err.Error(), true)
		return m, nil
	}
	m.reloadEnvs()
	m.setNotice("environment "+env+" updated", false)
	m.envTab = indexOf(env, m.envNames)
	return m.openEnvManager()
}

// deleteVariable removes key from an environment, persists it, and
// reopens the env manager.
func (m *Model) deleteVariable(env, key string) tea.Cmd {
	vars := m.envs[env]
	if vars == nil {
		return nil
	}
	delete(vars, key)
	if err := collection.SaveEnvironment(m.dir, env, vars); err != nil {
		m.setNotice("edit environment: "+err.Error(), true)
		return nil
	}
	m.reloadEnvs()
	m.setNotice("deleted variable "+key, false)
	m.envTab = indexOf(env, m.envNames)
	_, _ = m.openEnvManager() // mutates m in place via pointer receiver
	return nil
}

// envManagerView renders the environment manager overlay: a tab bar of
// environments above the active tab's variables.
func (m *Model) envManagerView() string {
	tabRow := ui.TabBar(m.envNames, m.envTab)
	return lipgloss.JoinVertical(lipgloss.Left, tabRow, m.palette.View())
}

// matches reports whether km hits any of the action's shortcut keys.
func (a Action) matches(km tea.KeyMsg) bool {
	if len(a.Keys) == 0 {
		return false
	}
	b := key.NewBinding(key.WithKeys(a.Keys...))
	return key.Matches(km, b)
}

// openPalette shows the command palette over the current frame.
func (m *Model) openPalette() (tea.Model, tea.Cmd) {
	actions := m.paletteActions()
	items := make([]ui.PaletteItem, len(actions))
	for i, a := range actions {
		items[i] = ui.PaletteItem{Title: a.Title, Shortcut: a.Shortcut}
	}
	m.palette.SetItems(items)
	m.palette.Resize(m.paletteWidth(items), m.minPaletteHeight(len(items)))
	m.palette.Open()
	m.palettePrev = m.focus
	m.paletteTheme = false
	m.paletteOpen = true
	return m, nil
}

// updatePalette routes a key while the palette is open: enter runs the
// selected action, esc/q close it. Non-key messages (e.g. the list's
// FilterMatchesMsg) are passed through so async filtering works.
func (m *Model) updatePalette(msg tea.Msg) (tea.Model, tea.Cmd) {
	if km, ok := msg.(tea.KeyMsg); ok {
		switch {
		case km.String() == "esc" || km.String() == "q":
			m.paletteOpen = false
			m.paletteTheme = false
			return m, m.enter(m.palettePrev)

		// bubbles routes every key to the filter input in Filtering state
		// and disables its nav bindings, so move the cursor ourselves. j/k
		// stay free for the filter query.
		case km.String() == "up" || km.String() == "ctrl+p":
			m.palette.CursorUp()
			return m, nil
		case km.String() == "down" || km.String() == "ctrl+n":
			m.palette.CursorDown()
			return m, nil

		case key.Matches(km, m.keyEnter):
			if m.paletteTheme {
				return m.applySelectedTheme()
			}
			actions := m.paletteActions()
			if it := m.palette.Selected(); it != nil {
				for _, a := range actions {
					if a.Title == it.Title {
						m.paletteOpen = false
						return a.Run(m)
					}
				}
			}
			return m, nil
		}
	}
	cmd, _ := m.palette.Update(msg)
	return m, cmd
}

// paletteWidth sizes the palette to its widest item (+ shortcut), capped so
// the dialog stays tight and centered instead of sprawling across the pane
// borders ([[Design - command palette]]).
func (m *Model) paletteWidth(items []ui.PaletteItem) int {
	w := m.width - 8
	longest := 0
	for _, it := range items {
		l := lipgloss.Width(it.Title) + lipgloss.Width(it.Shortcut)
		if l > longest {
			longest = l
		}
	}
	if longest+8 < w {
		w = longest + 8
	}
	if w < 20 {
		w = 20
	}
	return w
}

// minPaletteHeight sizes the palette dialog to its items (+ the filter
// row + border), capped so it never swallows the terminal.
func (m *Model) minPaletteHeight(n int) int {
	h := n + 2 // +2 for the filter row + border
	if h < 4 {
		return 4
	}
	if h > 10 {
		return 10
	}
	return h
}

// updateNamer routes a key while the namer is open: enter creates (or
// renames) the request or folder, esc cancels. Everything else feeds the
// text input.
func (m *Model) updateNamer(msg tea.Msg) (tea.Model, tea.Cmd) {
	if km, ok := msg.(tea.KeyMsg); ok {
		switch {
		case km.String() == "esc":
			m.namerOpen = false
			m.namerRename = false
			m.namerEnvEditVar = ""
			return m, nil

		case key.Matches(km, m.keyEnter):
			name := m.namer.Value()
			if name == "" {
				m.setNotice("name is required", true)
				return m, nil
			}

			// environment variable edit (key=value)
			if m.namerEnvEditVar != "" {
				m.namerOpen = false
				env := m.namerEnvEditVar
				m.namerEnvEditVar = ""
				return m.setEnvironmentVar(env, name)
			}

			// renaming is only valid for requests, so a leading / (folder
			// mode) is not allowed
			if m.namerRename {
				if m.namer.IsFolder() {
					m.setNotice("rename cannot create a folder", true)
					return m, nil
				}
				m.namerOpen = false
				m.namerRename = false
				return m.renameRequest(m.namerOld, name)
			}
			m.namerOpen = false
			if m.namer.IsFolder() {
				return m.createFolderIn(m.namerDir, name)
			}
			return m.createRequestIn(m.namerDir, name)
		}
	}
	cmd := m.namer.Update(msg)
	return m, cmd
}

// openDeleteConfirm shows the confirm modal for deleting e (a request or
// a folder). The actual delete only runs if the user confirms.
func (m *Model) openDeleteConfirm(e *collection.Entry) tea.Cmd {
	kind := "request"
	if e.Kind == collection.Dir {
		kind = "folder"
	}
	label := "delete " + kind + " " + ui.TruncateRunes(e.Name, 30) + "?"
	m.confirm.Ask(label)
	m.confirmOpen = true
	m.confirmTarget = e
	return nil
}

// updateConfirm routes keys while the confirm modal is open: y/enter runs
// the pending delete, n/esc cancels.
func (m *Model) updateConfirm(msg tea.Msg) (tea.Model, tea.Cmd) {
	if km, ok := msg.(tea.KeyMsg); ok {
		switch km.String() {
		case "y", "enter":
			if m.confirmTarget != nil {
				target := m.confirmTarget
				m.confirmTarget = nil
				m.confirmOpen = false
				return m, m.doDelete(target)
			}
			if m.confirmVarKey != "" {
				env, key := m.confirmVarEnv, m.confirmVarKey
				m.confirmVarEnv, m.confirmVarKey = "", ""
				m.confirmOpen = false
				return m, m.deleteVariable(env, key)
			}
		case "n", "esc", "q":
			m.confirmOpen = false
			m.confirmTarget = nil
			m.confirmVarEnv, m.confirmVarKey = "", ""
		}
	}
	return m, nil
}

// doDelete removes the highlighted entry (file or folder) and reloads the
// tree. If it was the active request the editor is reset.
func (m *Model) doDelete(e *collection.Entry) tea.Cmd {
	if err := collection.Delete(m.dir, e.Path); err != nil {
		m.setNotice("delete failed: "+err.Error(), true)
		return nil
	}
	if e.Path == m.editor.ActivePath() {
		m.urlbar.New()
		m.editor.New()
	}
	entries, err := collection.Load(m.dir)
	if err == nil {
		m.sidebar.SetEntries(entries)
	}
	m.setNotice("deleted "+rel(m.dir, e.Path), false)
	return m.saveState()
}

// renameRequest rewrites the request at oldPath under its new slug path
// and removes the old file.
func (m *Model) renameRequest(oldPath, name string) (tea.Model, tea.Cmd) {
	req, err := collection.LoadFile(oldPath)
	if err != nil {
		m.setNotice("rename failed: "+err.Error(), true)
		return m, nil
	}
	req.Name = name
	newPath := filepath.Join(filepath.Dir(oldPath), collection.Slug(name)+".yaml")
	if _, err := collection.Save(m.dir, newPath, req); err != nil {
		m.setNotice("rename failed: "+err.Error(), true)
		return m, nil
	}
	if err := os.Remove(oldPath); err != nil {
		m.setNotice("rename failed: "+err.Error(), true)
		return m, nil
	}
	entries, err := collection.Load(m.dir)
	if err == nil {
		m.sidebar.SetEntries(entries)
	}
	if oldPath == m.editor.ActivePath() {
		m.editor.SetActivePath(newPath)
		m.urlbar.SetRequest(req.Method, req.URL)
		m.editor.SetRequest(req, newPath)
	}
	m.setNotice("renamed "+rel(m.dir, oldPath)+" → "+rel(m.dir, newPath), false)
	return m, m.saveState()
}

// createFolderIn makes a new directory under dir and reloads the tree.
func (m *Model) createFolderIn(dir, name string) (tea.Model, tea.Cmd) {
	path := filepath.Join(dir, collection.Slug(name))
	if err := os.MkdirAll(path, 0o755); err != nil {
		m.setNotice("create failed: "+err.Error(), true)
		return m, nil
	}
	entries, err := collection.Load(m.dir)
	if err == nil {
		m.sidebar.SetEntries(entries)
	}
	m.setNotice("created "+rel(m.dir, path), false)
	return m, nil
}

// createRequestIn writes a blank named request under dir, reloads the
// tree, and loads it into the editor so the user can fill it in.
func (m *Model) createRequestIn(dir, name string) (tea.Model, tea.Cmd) {
	req := &collection.Request{Name: name, Method: "GET"}
	path := filepath.Join(dir, collection.Slug(name)+".yaml")
	if _, err := collection.Save(m.dir, path, req); err != nil {
		m.setNotice("create failed: "+err.Error(), true)
		return m, nil
	}
	entries, err := collection.Load(m.dir)
	if err == nil {
		m.sidebar.SetEntries(entries)
	}
	m.urlbar.SetRequest(req.Method, req.URL)
	m.setNotice("created "+rel(m.dir, path), false)
	return m, tea.Batch(m.editor.SetRequest(req, path), m.enter(pBar))
}

// send composes the request (URL/method from the bar, the rest from the
// editor), runs the pre-hook, then the HTTP call off the render loop, and
// finally the post-hook. Results come back as responseMsg or errMsg; both
// carry any store writes made by the hooks.
func (m *Model) send() (tea.Model, tea.Cmd) {
	req := m.editor.Request()
	req.Method = m.urlbar.Method()
	req.URL = m.urlbar.URL()
	if req.URL == "" {
		m.setNotice("URL is required", true)
		return m, nil
	}
	m.setNotice("", false)
	vars := m.activeVars()
	store := cloneVars(m.store)

	if req.Pre != "" {
		extra, err := script.Pre(req.Pre, req, vars, store)
		if err != nil {
			m.setNotice(err.Error(), true)
			return m, nil
		}
		vars = mergeVars(vars, extra)
	}

	// interpolation precedence: env → store → pre-returned vars
	vars = mergeVars(vars, store)

	preReq := *req // snapshot for the post-hook (post sees the request as sent)
	cmd := func() tea.Msg {
		res, err := httpclient.Exec(preReq, vars)
		if err != nil {
			return errMsg{err: err, store: store}
		}
		if preReq.Post != "" {
			if fail, err := script.Post(preReq.Post, &preReq, vars, store,
				res.Status, res.StatusCode, res.Headers, string(res.Body)); err != nil {
				return errMsg{err: err, store: store}
			} else if fail != "" {
				return errMsg{err: fmt.Errorf("post hook: %s", fail), store: store}
			}
		}
		return responseMsg{res: res, store: store}
	}
	return m, tea.Batch(m.response.StartLoading(), cmd)
}

// cloneVars returns a shallow copy of vars (nil-safe).
func cloneVars(vars map[string]string) map[string]string {
	out := make(map[string]string, len(vars))
	for k, v := range vars {
		out[k] = v
	}
	return out
}

// mergeVars layers extra over base (extra wins).
func mergeVars(base, extra map[string]string) map[string]string {
	out := make(map[string]string, len(base)+len(extra))
	for k, v := range base {
		out[k] = v
	}
	for k, v := range extra {
		out[k] = v
	}
	return out
}

// save persists the composed request to disk, then reloads the sidebar
// so the new/changed file appears in the tree.
func (m *Model) save() (tea.Model, tea.Cmd) {
	req := m.editor.Request()
	req.Method = m.urlbar.Method()
	req.URL = m.urlbar.URL()
	if req.URL == "" {
		m.setNotice("nothing to save: URL is empty", true)
		return m, nil
	}
	if req.Name == "" {
		req.Name = defaultName(req.URL)
	}
	path, err := collection.Save(m.dir, m.editor.ActivePath(), req)
	if err != nil {
		m.setNotice("save failed: "+err.Error(), true)
		return m, nil
	}
	m.editor.SetActivePath(path)
	entries, err := collection.Load(m.dir)
	if err == nil {
		m.sidebar.SetEntries(entries)
	}
	m.setNotice("saved "+rel(m.dir, path), false)
	return m, nil
}

// cycleEnv advances the active environment: none → env1 → env2 → … → none.
func (m *Model) cycleEnv() {
	if len(m.envNames) == 0 {
		return
	}
	// +1 slot for "none" at index 0
	m.envIdx = (m.envIdx + 1) % (len(m.envNames) + 1)
	if name := m.activeEnvName(); name != "" {
		m.setNotice("environment: "+name, false)
	} else {
		m.setNotice("environment: none", false)
	}
}

// quit persists state synchronously (the program is about to exit, so an
// async save could be cut off) then quits.
func (m *Model) quit() tea.Cmd {
	_ = session.Save(m.dir, m.snapshot())
	return tea.Quit
}

// snapshot captures the persisted UI state (env, active request, collapsed
// dirs) without writing it.
func (m *Model) snapshot() session.State {
	st := m.state
	st.Env = m.activeEnvName()
	if e := m.sidebar.Selected(); e != nil {
		if rel, err := filepath.Rel(m.dir, e.Path); err == nil {
			st.ActivePath = rel
		}
	}
	st.Collapsed = m.sidebar.CollapsedPaths(m.dir)
	return st
}

// saveState snapshots the persisted UI state (env, active request,
// collapsed dirs) to disk off the render loop.
func (m *Model) saveState() tea.Cmd {
	st := m.snapshot()
	return func() tea.Msg {
		if err := session.Save(m.dir, st); err != nil {
			return errMsg{err: err}
		}
		return nil
	}
}

// importCurl parses a pasted curl command into the bar and editor,
// replacing whatever was there ([[Design - curl import export]]).
func (m *Model) importCurl(text string) (tea.Model, tea.Cmd) {
	req, err := curl.Parse(text)
	if err != nil {
		m.setNotice("curl import failed: "+err.Error(), true)
		return m, nil
	}
	m.urlbar.SetRequest(req.Method, req.URL)
	m.editor.SetRequest(req, "")
	m.setNotice("imported curl request", false)
	return m, nil
}

// exportCurl writes the current request as a curl one-liner to the
// clipboard, interpolated with the active environment. Uses the platform
// tool (pbcopy etc.); falls back to OSC52, which Terminal.app ignores but
// iTerm2/Ghostty/kitty/wezterm honor.
func (m *Model) exportCurl() (tea.Model, tea.Cmd) {
	req := m.editor.Request()
	req.Method = m.urlbar.Method()
	req.URL = m.urlbar.URL()
	if req.URL == "" {
		m.setNotice("nothing to export: URL is empty", true)
		return m, nil
	}
	line := curlExportLine(*req, m.activeVars())
	m.setNotice("curl copied to clipboard", false)
	if err := clipboard.Write(line); err != nil {
		seq := "\x1b]52;c;" + base64.StdEncoding.EncodeToString([]byte(line)) + "\a"
		return m, tea.Printf("%s", seq)
	}
	return m, nil
}

// curlExportLine renders req as a curl command with {{vars}} interpolated
// from vars (unknown placeholders pass through, per ADR-0006).
func curlExportLine(req collection.Request, vars map[string]string) string {
	return curl.Format(render.Request(req, vars))
}

func (m *Model) setNotice(s string, isError bool) {
	m.notice = s
	m.noticeError = isError
}

func (m *Model) activeEnvName() string {
	if m.envIdx == 0 || len(m.envNames) == 0 {
		return ""
	}
	return m.envNames[m.envIdx-1]
}

func (m *Model) activeVars() map[string]string {
	if name := m.activeEnvName(); name != "" {
		return m.envs[name]
	}
	// nil vars makes interpolation a pass-through
	return nil
}
