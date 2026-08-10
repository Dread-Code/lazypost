package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Dread-Code/codeeditor"

	"lazypost/internal/collection"
)

func TestEditorCarriesHooks(t *testing.T) {
	e := NewEditor(60, 15)
	req := &collection.Request{
		Name:   "thing",
		Method: "GET",
		URL:    "https://api.test/things",
		Pre:    "req.headers['X-Ts'] = os.time()",
		Post:   "return response.status_code == 200",
	}
	e.SetRequest(req, "/col/thing.yaml")
	got := e.Request()
	if got.Pre != req.Pre || got.Post != req.Post {
		t.Errorf("hooks not carried: pre=%q post=%q", got.Pre, got.Post)
	}
	if got.Name != "thing" {
		t.Errorf("name lost: %q", got.Name)
	}

	e.New()
	if e.Request().Pre != "" || e.Request().Post != "" {
		t.Error("New should clear hooks")
	}
}

// Regression: loading another request must keep the section the user is
// on (e.g. Headers), not reset to Query.
func TestSetRequestKeepsSection(t *testing.T) {
	e := NewEditor(60, 20)
	e.section = SecHeaders
	e.SetRequest(&collection.Request{Name: "a", URL: "https://api.test/a"}, "/col/a.yaml")
	if e.section != SecHeaders {
		t.Errorf("section reset to %d, want SecHeaders", e.section)
	}
	// a genuinely new request still starts on Query
	e.New()
	if e.section != SecQuery {
		t.Errorf("New should start on Query, got %d", e.section)
	}
}

func TestSectionAccessors(t *testing.T) {
	e := NewEditor(60, 20)
	if e.Section() != int(SecQuery) {
		t.Errorf("initial section = %d", e.Section())
	}
	e.SetSection(int(SecScripts))
	if e.Section() != int(SecScripts) {
		t.Errorf("SetSection(SecScripts) = %d", e.Section())
	}
	e.SetSection(99)
	if e.Section() != int(SecScripts) {
		t.Errorf("out-of-range SetSection changed section to %d", e.Section())
	}
	e.SetSection(-1)
	if e.Section() != int(SecScripts) {
		t.Errorf("negative SetSection changed section to %d", e.Section())
	}
}

// Regression: the display name of a loaded request must survive a save —
// the name field is preserved instead of being re-derived from the path
// slug (loading 01-create.yaml and saving used to rewrite
// `name: create post` to `01-create`).
func TestSetRequestPreservesName(t *testing.T) {
	e := NewEditor(60, 20)
	e.SetRequest(&collection.Request{Name: "create post", URL: "https://api.test/posts"}, "/col/01-create.yaml")
	if got := e.Request().Name; got != "create post" {
		t.Errorf("name = %q, want the loaded display name", got)
	}
	// requests without a name still fall back to the slug
	e.SetRequest(&collection.Request{URL: "https://api.test/x"}, "/col/other.yaml")
	if got := e.Request().Name; got != "other" {
		t.Errorf("fallback name = %q, want slug 'other'", got)
	}
	// New clears it
	e.New()
	if got := e.Request().Name; got != "" {
		t.Errorf("New should clear the name, got %q", got)
	}
}

// The editor exposes the active field's vim mode for the footer row.
func TestEditorModeLabel(t *testing.T) {
	e := NewEditor(60, 20)
	e.Focus()
	// focusing the editor lands on the Query code field: NORMAL on entry
	if got := e.ModeLabel(); got != "—NORMAL—" {
		t.Errorf("initial mode label = %q, want —NORMAL— (query)", got)
	}
	// the Query field is a code editor too: type 'i' to insert
	e.query.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("i")})
	if got := e.ModeLabel(); got != "—INSERT—" {
		t.Errorf("query mode label in insert = %q, want —INSERT—", got)
	}
	// switch to the Headers tab: also a code field, NORMAL on entry
	for e.section != SecHeaders {
		e.Update(tea.KeyMsg{Type: tea.KeyCtrlN})
	}
	if got := e.ModeLabel(); got != "—NORMAL—" {
		t.Errorf("headers mode label = %q, want —NORMAL— on entry", got)
	}
	// switch to the Body tab: entering the code field resets to NORMAL
	for e.section != SecBody {
		e.Update(tea.KeyMsg{Type: tea.KeyCtrlN})
	}
	if got := e.ModeLabel(); got != "—NORMAL—" {
		t.Errorf("body mode label = %q, want —NORMAL— on entry", got)
	}
	// entering insert mode shows the insert footer; esc returns to normal
	e.body.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("i")})
	if got := e.ModeLabel(); got != "—INSERT—" {
		t.Errorf("body mode label in insert = %q, want —INSERT—", got)
	}
	e.body.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if got := e.ModeLabel(); got != "—NORMAL—" {
		t.Errorf("body mode label after esc = %q, want —NORMAL—", got)
	}
	// the Scripts tab's pre hook also lands in NORMAL on entry
	for e.section != SecScripts {
		e.Update(tea.KeyMsg{Type: tea.KeyCtrlN})
	}
	if got := e.ModeLabel(); got != "—NORMAL—" {
		t.Errorf("scripts tab mode label = %q, want —NORMAL—", got)
	}
	// the footer row is rendered for code sections, empty on Auth
	if !strings.Contains(e.View(), "—NORMAL—") {
		t.Errorf("footer should render the mode, got:\n%s", e.View())
	}
	for e.section != SecAuth {
		e.Update(tea.KeyMsg{Type: tea.KeyCtrlN})
	}
	if got := e.ModeLabel(); got != "" {
		t.Errorf("auth tab mode label = %q, want empty (no code field)", got)
	}
}

func TestEditorScriptsTab(t *testing.T) {
	e := NewEditor(60, 20)
	// navigate to the Scripts tab (index 4) via ctrl+n
	e.Focus()
	for i := 0; i < 4; i++ {
		e.Update(tea.KeyMsg{Type: tea.KeyCtrlN})
	}
	if e.section != SecScripts {
		t.Fatalf("expected Scripts section, got %d", e.section)
	}
	if !strings.Contains(e.View(), "Scripts") {
		t.Errorf("Scripts tab not rendered:\n%s", e.View())
	}
	if !strings.Contains(e.View(), "pre") || !strings.Contains(e.View(), "post") {
		t.Errorf("pre/post labels not rendered:\n%s", e.View())
	}

	// focus starts on pre; ctrl+t moves to post, again back to pre
	e.Update(tea.KeyMsg{Type: tea.KeyCtrlT})
	if e.field != 1 {
		t.Errorf("expected field post after ctrl+t, got %d", e.field)
	}
	e.Update(tea.KeyMsg{Type: tea.KeyCtrlT})
	if e.field != 0 {
		t.Errorf("expected field pre after second ctrl+t, got %d", e.field)
	}

	// typing goes to the focused field (enter insert mode first: the
	// code fields land in NORMAL on focus)
	e.pre.SetMode(codeeditor.ModeInsert)
	e.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("return true")})
	if got := e.Request().Pre; got != "return true" {
		t.Errorf("expected pre set by typing, got %q", got)
	}

	// arrows move the cursor inside the textarea, not the hook toggle
	e.Update(tea.KeyMsg{Type: tea.KeyUp})
	if e.field != 0 {
		t.Errorf("up arrow should not change the field, got %d", e.field)
	}
}

// Regression ([[ADR-0016 Body JSON auto-format on save and blur]]): a
// valid JSON body is pretty-printed with 2-space indent the moment editing
// stops (Blur), so pane switches and saves store it formatted.
func TestBlurFormatsValidJSONBody(t *testing.T) {
	e := NewEditor(60, 20)
	req := &collection.Request{Name: "a", URL: "https://api.test/a", Body: `{"a":1,"b":[true,null,"x"]}`}
	e.SetRequest(req, "/col/a.yaml")
	want := "{\n  \"a\": 1,\n  \"b\": [\n    true,\n    null,\n    \"x\"\n  ]\n}"
	if got := e.body.Value(); got != want {
		t.Errorf("body after blur:\n got %q\nwant %q", got, want)
	}
	// idempotent: a second blur leaves the formatted body alone
	e.Blur()
	if got := e.body.Value(); got != want {
		t.Errorf("second blur reformatted:\n got %q\nwant %q", got, want)
	}
}

// Regression: bodies that are not valid JSON — including half-typed
// bodies and plain text — must pass through untouched. Raw
// {{placeholders}} in value positions (e.g. `"userId": {{user_id}}`) make
// the body invalid JSON but still format around the placeholders
// ([[ADR-0017]]).
func TestBlurLeavesInvalidJSONUntouched(t *testing.T) {
	cases := []string{
		`{"a": 1`,       // unterminated
		`{"userId": 1,`, // trailing comma
		`plain text`,
	}
	for _, body := range cases {
		e := NewEditor(60, 20)
		req := &collection.Request{Name: "a", URL: "https://api.test/a", Body: body}
		e.SetRequest(req, "/col/a.yaml")
		if got := e.body.Value(); got != body {
			t.Errorf("body %q changed to %q on blur", body, got)
		}
	}
}

// Regression ([[ADR-0017]]): a raw placeholder in a value position makes
// the body invalid JSON, but it must still pretty-print like the response
// pane — the placeholder survives verbatim in the formatted output.
func TestBlurFormatsBodyWithValuePlaceholder(t *testing.T) {
	e := NewEditor(60, 20)
	body := `{"title": "{{title}}", "body": "A post created from lazypost", "userId": {{user_id}}}`
	e.body.SetValue(body)
	e.Blur()
	want := "{\n  \"title\": \"{{title}}\",\n  \"body\": \"A post created from lazypost\",\n  \"userId\": {{user_id}}\n}"
	if got := e.body.Value(); got != want {
		t.Errorf("body after blur:\n got %q\nwant %q", got, want)
	}
	// idempotent: formatting again yields the same output
	e.Blur()
	if got := e.body.Value(); got != want {
		t.Errorf("second blur changed the formatted body:\n got %q\nwant %q", got, want)
	}
}

// Regression: a placeholder inside a string is valid JSON, so the body
// still pretty-prints while keeping the raw placeholder ([[ADR-0016]]).
func TestBlurFormatsBodyWithStringPlaceholder(t *testing.T) {
	e := NewEditor(60, 20)
	req := &collection.Request{Name: "a", URL: "https://api.test/a", Body: `{"a":"{{title}}"}`}
	e.SetRequest(req, "/col/a.yaml")
	want := "{\n  \"a\": \"{{title}}\"\n}"
	if got := e.body.Value(); got != want {
		t.Errorf("body after blur:\n got %q\nwant %q", got, want)
	}
}
