package model

import (
	"slices"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"lazypost/internal/collection"
	"lazypost/internal/ui/widgets"

	"lazypost/internal/ui/themes"
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
	if m.overlay != ovEnv {
		// open on the active environment if set, else the first
		m.palette.envTab = m.envIdx - 1
		if m.palette.envTab < 0 {
			m.palette.envTab = 0
		}
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
		for k, v := range m.envs[env] {
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
				m.namer.envNew = true
				m.overlay = ovNamer
				m.namer.widget.SetLabel("new variable")
				m.namer.widget.SetPlaceholder("key=value")
				m.namer.widget.SetEnvMode(true)
				return m, m.namer.widget.Open()
			}
			return m, nil

		case key.Matches(km, keyRename):
			if env := m.envTabName(); env != "" {
				if key := m.selectedVarName(); key != "" {
					m.namer.envEdit = env
					m.namer.envNew = false
					m.overlay = ovNamer
					m.namer.widget.SetLabel("edit variable")
					m.namer.widget.SetPlaceholder("key=value")
					m.namer.widget.SetEnvMode(true)
					return m, m.namer.widget.OpenPrefilled(key + "=" + m.envs[env][key])
				}
			}
			return m, nil

		case key.Matches(km, keyDelete):
			if env := m.envTabName(); env != "" {
				if key := m.selectedVarName(); key != "" {
					m.confirm.widget.Ask("delete variable " + themes.TruncateRunes(key, 30) + "?")
					m.overlay = ovConfirm
					m.confirm.env = env
					m.confirm.key = key
				}
			}
			return m, nil

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
	if k, _, ok := strings.Cut(it.Title, " = "); ok {
		return k
	}
	return ""
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
	if !m.prepareWrite() {
		return m, nil
	}
	vars := m.envs[env]
	if vars == nil {
		vars = map[string]string{}
	}
	vars[strings.TrimSpace(key)] = strings.TrimSpace(val)
	if err := collection.SaveEnvironment(m.dir, env, vars); err != nil {
		m.writeNotice("edit environment: "+err.Error(), true)
		return m, nil
	}
	m.reloadEnvs()
	m.writeNotice("environment "+env+" updated", false)
	m.palette.envTab = slices.Index(m.envNames, env)
	return m.openEnvManager()
}

// createEnvironment creates an empty environment (from a leading "/" in
// the add-variable namer), persists it, and reopens the env manager on
// the new tab.
func (m *Model) createEnvironment(name string) (tea.Model, tea.Cmd) {
	if !m.prepareWrite() {
		return m, nil
	}
	if err := collection.SaveEnvironment(m.dir, name, map[string]string{}); err != nil {
		m.writeNotice("create environment: "+err.Error(), true)
		return m, nil
	}
	m.reloadEnvs()
	m.writeNotice("environment "+name+" created", false)
	m.palette.envTab = slices.Index(m.envNames, name)
	return m.openEnvManager()
}

// deleteVariable removes key from an environment, persists it, and
// reopens the env manager.
func (m *Model) deleteVariable(env, key string) tea.Cmd {
	vars := m.envs[env]
	if vars == nil {
		return nil
	}
	if !m.prepareWrite() {
		return nil
	}
	delete(vars, key)
	if err := collection.SaveEnvironment(m.dir, env, vars); err != nil {
		m.writeNotice("edit environment: "+err.Error(), true)
		return nil
	}
	m.reloadEnvs()
	m.writeNotice("deleted variable "+key, false)
	m.palette.envTab = slices.Index(m.envNames, env)
	_, _ = m.openEnvManager() // mutates m in place via pointer receiver
	return nil
}

// envManagerView renders the environment manager overlay: a tab bar of
// environments above the active tab's variables, with an action hint row
// so the modal's keys are discoverable without ?.
func (m *Model) envManagerView() string {
	tabRow := themes.TabBar(m.envNames, m.palette.envTab, max(0, m.width-12), nil)
	hint := themes.KeyHint("enter", "activate", "ctrl+e", "tab", "a", "add", "r", "edit", "d", "delete", "/", "filter", "esc", "close")
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
