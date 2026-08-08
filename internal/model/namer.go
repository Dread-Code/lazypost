package model

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// updateNamer routes a key while the namer is open: enter creates (or
// renames) the request or folder, esc cancels. Everything else feeds the
// text input.
func (m *Model) updateNamer(msg tea.Msg) (tea.Model, tea.Cmd) {
	if km, ok := msg.(tea.KeyMsg); ok {
		switch {
		case key.Matches(km, keyEsc):
			m.overlay = noOverlay
			m.namer.rename = false
			m.namer.envEdit = ""
			m.namer.envNew = false
			m.namer.widget.SetEnvMode(false)
			return m, nil

		case key.Matches(km, keyEnter):
			name := m.namer.widget.Value()
			if name == "" {
				m.setNotice("name is required", true)
				return m, nil
			}

			// environment variable edit (key=value); a leading "/" in add
			// mode creates a new environment instead
			if m.namer.envEdit != "" {
				m.overlay = noOverlay
				env := m.namer.envEdit
				m.namer.envEdit = ""
				if m.namer.envNew && m.namer.widget.IsFolder() {
					m.namer.envNew = false
					m.namer.widget.SetEnvMode(false)
					return m.createEnvironment(name)
				}
				m.namer.envNew = false
				m.namer.widget.SetEnvMode(false)
				return m.setEnvironmentVar(env, name)
			}

			// renaming is only valid for requests, so a leading / (folder
			// mode) is not allowed
			if m.namer.rename {
				if m.namer.widget.IsFolder() {
					m.setNotice("rename cannot create a folder", true)
					return m, nil
				}
				m.overlay = noOverlay
				m.namer.rename = false
				return m.renameRequest(m.namer.old, name)
			}
			m.overlay = noOverlay
			if m.namer.widget.IsFolder() {
				return m.createFolderIn(m.namer.dir, name)
			}
			return m.createRequestIn(m.namer.dir, name)
		}
	}
	cmd := m.namer.widget.Update(msg)
	return m, cmd
}
