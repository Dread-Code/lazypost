package model

import (
	"slices"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Dread-Code/lazypost/internal/collection"
	"github.com/Dread-Code/lazypost/internal/ui/widgets"

	"github.com/Dread-Code/lazypost/internal/ui/themes"
)

// openEnvManager shows the environment manager: a tab bar of environments
// with the active tab's variables listed below. ctrl+e cycles the tab,
// enter activates it, a/r/d edit variables ([[Design - environment
// manager modal]]).
func (m *Model) openEnvManager() (tea.Model, tea.Cmd) {
	if len(m.envNames) == 0 {
		m.setNotice("no environments", true)
		return m, nil
	}
	env := ""
	if m.overlay == ovEnv {
		env = m.envTabName()
	} else if active := m.activeEnvName(); active != "" {
		env = active
	}
	return m.openEnvManagerAt(env)
}

func (m *Model) openEnvManagerAt(name string) (tea.Model, tea.Cmd) {
	if idx := slices.Index(m.envNames, name); idx >= 0 {
		m.palette.envTab = idx
	} else if m.palette.envTab < 0 || m.palette.envTab >= len(m.envNames) {
		m.palette.envTab = 0
	}
	m.overlay = ovEnv
	m.palette.envFiltering = false
	m.loadEnvTab()
	m.palette.prev = m.focus
	return m, nil
}

// envTabName returns the environment for the current tab.
func (m *Model) envTabName() string {
	if m.palette.envTab < 0 || m.palette.envTab >= len(m.envNames) {
		return ""
	}
	return m.envNames[m.palette.envTab]
}

// loadEnvTab populates the palette with the current tab's variables. The
// key renders as the row title, the value hangs off it as "= value" so
// "key = value" reads as one line ([[Design - environment manager
// modal]]).
func (m *Model) loadEnvTab() {
	items := []ui.PaletteItem{}
	if env := m.envTabName(); env != "" {
		keys := make([]string, 0, len(m.envs[env]))
		for k := range m.envs[env] {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			v := m.envs[env][k]
			items = append(items, ui.PaletteItem{Title: k, Detail: "= " + v})
		}
	}
	m.palette.widget.SetItems(items)
	w := m.paletteWidth(items)
	if w < 50 {
		w = 50
	}
	m.palette.widget.Resize(w, m.minPaletteHeight(len(items)+2)) // tab row + filter
	if m.palette.envFiltering {
		m.palette.widget.StartFiltering()
	} else {
		m.palette.widget.OpenBrowsing()
	}
}

// updateEnvManager routes keys while the environment manager is open:
// ctrl+e cycles tabs, enter activates the tab, a/r/d edit variables,
// "/" starts filtering (so letters search instead of acting), esc/q
// closes.
func (m *Model) updateEnvManager(msg tea.Msg) (tea.Model, tea.Cmd) {
	if km, ok := msg.(tea.KeyMsg); ok {
		// While filtering, all letters go to the filter: only esc/q act.
		if m.palette.envFiltering {
			switch {
			case key.Matches(km, keyEsc) || key.Matches(km, keyQuit):
				m.palette.envFiltering = false
				m.palette.widget.ClearFilter()
				return m, nil
			case key.Matches(km, keyUp) || key.Matches(km, keyCtrlP):
				m.palette.widget.CursorUp()
				return m, nil
			case key.Matches(km, keyDown) || key.Matches(km, keyCtrlN):
				m.palette.widget.CursorDown()
				return m, nil
			case key.Matches(km, keyRename):
				return m.editSelectedVariable()
			case key.Matches(km, keyDelete):
				return m.confirmDeleteSelectedVariable()
			}
			cmd, _ := m.palette.widget.Update(msg)
			return m, cmd
		}

		switch {
		case key.Matches(km, keyEsc) || key.Matches(km, keyQuit):
			m.overlay = noOverlay
			return m, m.enter(m.palette.prev)

		case key.Matches(km, keySlash):
			// enter filter mode: letters now search the variables
			m.palette.envFiltering = true
			m.palette.widget.StartFiltering()
			return m, nil

		// bubbles routes every key to the filter input in Filtering state
		// and disables its nav bindings, so move the cursor ourselves.
		case key.Matches(km, keyUp) || key.Matches(km, keyCtrlP):
			m.palette.widget.CursorUp()
			return m, nil
		case key.Matches(km, keyDown) || key.Matches(km, keyCtrlN):
			m.palette.widget.CursorDown()
			return m, nil

		// cycle the environment tab (ctrl+e is free here: the manager is
		// modal, so the global cycle key is intercepted)
		case key.Matches(km, keyCtrlE):
			m.palette.envTab = (m.palette.envTab + 1) % len(m.envNames)
			m.loadEnvTab()
			return m, nil

		case key.Matches(km, keyAdd):
			if env := m.envTabName(); env != "" {
				m.namer.envEdit = env
				m.namer.envKey = ""
				m.namer.envNew = true
				m.overlay = ovNamer
				m.namer.widget.SetLabel("new variable")
				m.namer.widget.SetPlaceholder("key=value")
				m.namer.widget.SetEnvMode(true)
				return m, m.namer.widget.Open()
			}
			return m, nil

		case key.Matches(km, keyRename):
			return m.editSelectedVariable()

		case key.Matches(km, keyDelete):
			return m.confirmDeleteSelectedVariable()

		case key.Matches(km, keyEnter):
			m.setEnv(m.envTabName())
			m.overlay = noOverlay
			return m, m.saveState()
		}
	}
	cmd, _ := m.palette.widget.Update(msg)
	return m, cmd
}

// selectedVarName returns the highlighted variable's key (the part before
// " = ").
func (m *Model) selectedVarName() string {
	it := m.palette.widget.Selected()
	if it == nil {
		return ""
	}
	return it.Title
}

func (m *Model) editSelectedVariable() (tea.Model, tea.Cmd) {
	env := m.envTabName()
	key := m.selectedVarName()
	if env == "" || key == "" {
		return m, nil
	}
	m.namer.envEdit = env
	m.namer.envKey = key
	m.namer.envNew = false
	m.overlay = ovNamer
	m.namer.widget.SetLabel("edit variable")
	m.namer.widget.SetPlaceholder("key=value")
	m.namer.widget.SetEnvMode(true)
	return m, m.namer.widget.OpenPrefilled(key + "=" + m.envs[env][key])
}

func (m *Model) confirmDeleteSelectedVariable() (tea.Model, tea.Cmd) {
	env := m.envTabName()
	key := m.selectedVarName()
	if env == "" || key == "" {
		return m, nil
	}
	m.confirm.widget.Ask("delete variable " + themes.TruncateRunes(key, 30) + "?")
	m.overlay = ovConfirm
	m.confirm.env = env
	m.confirm.key = key
	return m, nil
}

// setEnv activates env by name ("" = none), re-resolving envIdx.
func (m *Model) setEnv(name string) {
	if name == "" {
		m.envIdx = 0
		m.updateEnvBadge()
		return
	}
	if idx := slices.Index(m.envNames, name); idx >= 0 {
		m.envIdx = idx + 1
	} else {
		m.envIdx = 0
	}
	m.updateEnvBadge()
}

// setEnvironmentVar writes key=value into an environment's map, persists
// it, and reopens the env manager (now on that env's tab).
func (m *Model) setEnvironmentVar(env, oldKey, kv string) (tea.Model, tea.Cmd) {
	key, val, ok := strings.Cut(kv, "=")
	if !ok || strings.TrimSpace(key) == "" {
		m.setNotice("expected key=value", true)
		return m, nil
	}
	id, legacy, ok := m.beginMutation()
	if !ok {
		return m, nil
	}
	vars := make(map[string]string, len(m.envs[env])+1)
	for existingKey, existingValue := range m.envs[env] {
		vars[existingKey] = existingValue
	}
	key = strings.TrimSpace(key)
	val = strings.TrimSpace(val)
	root := m.dir
	active := m.activeEnvName()
	return m, func() tea.Msg {
		migrated, err := runLegacyMigration(root, legacy)
		if err != nil {
			return environmentMutationMsg{id: id, env: env, active: active, err: err, migrated: migrated}
		}
		if oldKey != "" && oldKey != key {
			delete(vars, oldKey)
		}
		vars[key] = val
		if err := collection.SaveEnvironment(root, env, vars); err != nil {
			return environmentMutationMsg{id: id, env: env, active: active, err: err, migrated: migrated}
		}
		envs, names, err := collection.LoadEnvironments(root)
		return environmentMutationMsg{id: id, envs: envs, names: names, env: env, active: active, notice: "environment " + env + " updated", err: err, migrated: migrated}
	}
}

// createEnvironment creates an empty environment (from a leading "/" in
// the add-variable namer), persists it, and reopens the env manager on
// the new tab.
func (m *Model) createEnvironment(name string) (tea.Model, tea.Cmd) {
	id, legacy, ok := m.beginMutation()
	if !ok {
		return m, nil
	}
	root := m.dir
	active := m.activeEnvName()
	return m, func() tea.Msg {
		migrated, err := runLegacyMigration(root, legacy)
		if err != nil {
			return environmentMutationMsg{id: id, env: name, active: active, err: err, migrated: migrated}
		}
		if err := collection.CreateEnvironment(root, name, map[string]string{}); err != nil {
			return environmentMutationMsg{id: id, env: name, active: active, err: err, migrated: migrated}
		}
		envs, names, err := collection.LoadEnvironments(root)
		return environmentMutationMsg{id: id, envs: envs, names: names, env: name, active: active, notice: "environment " + name + " created", err: err, migrated: migrated}
	}
}

// deleteVariable removes key from an environment, persists it, and
// reopens the env manager.
func (m *Model) deleteVariable(env, key string) tea.Cmd {
	if m.envs[env] == nil {
		return nil
	}
	vars := make(map[string]string, len(m.envs[env]))
	for existingKey, existingValue := range m.envs[env] {
		vars[existingKey] = existingValue
	}
	delete(vars, key)
	id, legacy, ok := m.beginMutation()
	if !ok {
		return nil
	}
	root := m.dir
	active := m.activeEnvName()
	return func() tea.Msg {
		migrated, err := runLegacyMigration(root, legacy)
		if err != nil {
			return environmentMutationMsg{id: id, env: env, active: active, err: err, migrated: migrated}
		}
		if err := collection.SaveEnvironment(root, env, vars); err != nil {
			return environmentMutationMsg{id: id, env: env, active: active, err: err, migrated: migrated}
		}
		envs, names, err := collection.LoadEnvironments(root)
		return environmentMutationMsg{id: id, envs: envs, names: names, env: env, active: active, notice: "deleted variable " + key, err: err, migrated: migrated}
	}
}

func (m *Model) applyEnvironmentMutation(msg environmentMutationMsg) (tea.Model, tea.Cmd) {
	if msg.id != m.activeMutationID {
		return m, nil
	}
	m.activeMutationID = 0
	m.mutationBusy = false
	if msg.migrated {
		m.legacyMarkers = nil
		m.legacyMigrated = true
	}
	if msg.err != nil {
		m.writeNotice("environment operation failed: "+msg.err.Error(), true)
		return m, nil
	}
	m.envs = msg.envs
	m.envNames = msg.names
	m.setEnv(msg.active)
	m.writeNotice(msg.notice, false)
	return m.openEnvManagerAt(msg.env)
}

// envManagerView renders the environment manager overlay: a tab bar of
// environments above the active tab's variables, with an action hint row
// so the modal's keys are discoverable without ?.
func (m *Model) envManagerView() string {
	tabRow := m.styles.TabBar(m.envNames, m.palette.envTab, max(0, m.width-12), nil)
	hint := m.styles.KeyHint("enter", "activate", "ctrl+e", "tab", "a", "add", "r", "edit", "d", "delete", "/", "filter", "esc", "close")
	return lipgloss.JoinVertical(lipgloss.Left, tabRow, m.palette.widget.View(), hint)
}

// cycleEnv advances the active environment: none → env1 → env2 → … → none.
// The title bar's env badge is the single status display — no notice is
// raised, so there is only ever one env label on screen.
func (m *Model) cycleEnv() {
	if len(m.envNames) == 0 {
		return
	}
	// +1 slot for "none" at index 0
	m.envIdx = (m.envIdx + 1) % (len(m.envNames) + 1)
	m.updateEnvBadge()
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
