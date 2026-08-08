package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"postgo/internal/collection"
)

var authTypes = []string{"none", "basic", "bearer", "apikey"}

type AuthEditor struct {
	authType string
	username textinput.Model
	password textinput.Model
	token    textinput.Model
	keyName  textinput.Model
	keyValue textinput.Model
	keyIn    string // header | query
	field    int    // focused row: text inputs first, apikey adds a toggle row
	focused  bool
	width    int
	height   int
}

// NewAuthEditor wires five text inputs plus a header/query toggle; only
// the ones relevant to the current auth type are rendered and focused.
func NewAuthEditor() AuthEditor {
	newInput := func(placeholder string) textinput.Model {
		ti := textinput.New()
		ti.Placeholder = placeholder
		ti.Prompt = ""
		return ti
	}
	a := AuthEditor{
		authType: "none",
		username: newInput("username"),
		password: newInput("password"),
		token:    newInput("token"),
		keyName:  newInput("X-Api-Key"),
		keyValue: newInput("secret"),
		keyIn:    "header",
	}
	a.password.EchoMode = textinput.EchoPassword
	a.password.EchoCharacter = '•'
	return a
}

func (a *AuthEditor) SetWidth(w int) {
	a.width = w
	fieldW := w - 14
	if fieldW < 10 {
		fieldW = 10
	}
	for _, ti := range []*textinput.Model{&a.username, &a.password, &a.token, &a.keyName, &a.keyValue} {
		ti.Width = fieldW
	}
}

func (a *AuthEditor) SetHeight(_ int) {}

// inputs returns the fields visible for the current auth type; nil for
// "none". The apikey keyIn toggle row is tracked separately by field.
func (a *AuthEditor) inputs() []*textinput.Model {
	switch a.authType {
	case "basic":
		return []*textinput.Model{&a.username, &a.password}
	case "bearer":
		return []*textinput.Model{&a.token}
	case "apikey":
		return []*textinput.Model{&a.keyName, &a.keyValue}
	}
	return nil
}

func (a *AuthEditor) fieldCount() int {
	n := len(a.inputs())
	if a.authType == "apikey" {
		n++ // keyIn toggle row
	}
	return n
}

func (a *AuthEditor) Focus() tea.Cmd {
	a.focused = true
	if a.field >= a.fieldCount() {
		a.field = 0
	}
	if inputs := a.inputs(); a.field < len(inputs) {
		return inputs[a.field].Focus()
	}
	return nil
}

func (a *AuthEditor) Blur() {
	a.focused = false
	for _, ti := range a.inputs() {
		ti.Blur()
	}
}

// CycleType advances the auth type (none → basic → …), resetting the
// cursor to the first field of the new type.
func (a *AuthEditor) CycleType(n int) {
	for i, t := range authTypes {
		if t == a.authType {
			a.authType = authTypes[(i+n+len(authTypes))%len(authTypes)]
			a.field = 0
			if a.focused {
				a.Focus()
			}
			return
		}
	}
	a.authType = authTypes[0]
}

func (a *AuthEditor) Update(msg tea.Msg) tea.Cmd {
	if !a.focused {
		return nil
	}
	if km, ok := msg.(tea.KeyMsg); ok {
		if n := a.fieldCount(); n > 0 {
			switch km.String() {
			case "down":
				a.field = (a.field + 1) % n
				return a.Focus()
			case "up":
				a.field = (a.field - 1 + n) % n
				return a.Focus()
			}
			// field past the last input = apikey keyIn toggle row
			if a.authType == "apikey" && a.field == 2 {
				switch km.String() {
				case " ", "enter", "left", "right", "tab":
					if a.keyIn == "header" {
						a.keyIn = "query"
					} else {
						a.keyIn = "header"
					}
					return nil
				}
			}
		}
	}
	inputs := a.inputs()
	if a.field < len(inputs) {
		var cmd tea.Cmd
		*inputs[a.field], cmd = inputs[a.field].Update(msg)
		return cmd
	}
	return nil
}

func (a *AuthEditor) View() string {
	var types []string
	for _, t := range authTypes {
		if t == a.authType {
			types = append(types, ActiveTabStyle.Render(t))
		} else {
			types = append(types, TabStyle.Render(t))
		}
	}
	rows := []string{
		HintStyle.Render("type ") + strings.Join(types, ""),
	}

	label := lipgloss.NewStyle().Width(10).Foreground(ColorMuted)
	cursor := func(active bool) string {
		if active && a.focused {
			return lipgloss.NewStyle().Foreground(ColorPrimary).Render("▸ ")
		}
		return "  "
	}

	switch a.authType {
	case "none":
		rows = append(rows, HintStyle.Render("no authentication"))
	case "basic":
		rows = append(rows,
			cursor(a.field == 0)+label.Render("username")+" "+a.username.View(),
			cursor(a.field == 1)+label.Render("password")+" "+a.password.View(),
		)
	case "bearer":
		rows = append(rows,
			cursor(a.field == 0)+label.Render("token")+a.token.View(),
		)
	case "apikey":
		in := a.keyIn
		if a.authType == "apikey" && a.field == 2 && a.focused {
			in = lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true).Render(in)
		}
		rows = append(rows,
			cursor(a.field == 0)+label.Render("name")+" "+a.keyName.View(),
			cursor(a.field == 1)+label.Render("value")+a.keyValue.View(),
			cursor(a.field == 2)+label.Render("send in")+in+HintStyle.Render("  (space to toggle)"),
		)
	}
	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

// Auth returns the auth config, or nil when type is none.
func (a *AuthEditor) Auth() *collection.Auth {
	if a.authType == "none" {
		return nil
	}
	auth := &collection.Auth{Type: a.authType}
	switch a.authType {
	case "basic":
		auth.Username = a.username.Value()
		auth.Password = a.password.Value()
	case "bearer":
		auth.Token = a.token.Value()
	case "apikey":
		auth.KeyName = a.keyName.Value()
		auth.KeyValue = a.keyValue.Value()
		auth.KeyIn = a.keyIn
	}
	return auth
}

func (a *AuthEditor) SetAuth(auth *collection.Auth) {
	a.authType = "none"
	a.username.SetValue("")
	a.password.SetValue("")
	a.token.SetValue("")
	a.keyName.SetValue("")
	a.keyValue.SetValue("")
	a.keyIn = "header"
	a.field = 0
	if auth == nil {
		return
	}
	a.authType = auth.Type
	if a.authType == "" {
		a.authType = "none"
	}
	a.username.SetValue(auth.Username)
	a.password.SetValue(auth.Password)
	a.token.SetValue(auth.Token)
	a.keyName.SetValue(auth.KeyName)
	a.keyValue.SetValue(auth.KeyValue)
	if auth.KeyIn != "" {
		a.keyIn = auth.KeyIn
	}
}
