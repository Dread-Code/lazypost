package model

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"

	"postgo/internal/collection"
	"postgo/internal/session"
)

func loadSample(t *testing.T) Model {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	entries, err := collection.Load("../../sample-collections")
	if err != nil {
		t.Fatalf("load sample collections: %v", err)
	}
	envs, names, err := collection.LoadEnvironments("../../sample-collections")
	if err != nil {
		t.Fatalf("load environments: %v", err)
	}
	return New("../../sample-collections", entries, envs, names, session.State{})
}

// watcher accumulates program output across waitFor calls, since the
// bubbletea renderer only re-emits changed lines.
type watcher struct {
	r   io.Reader
	buf strings.Builder
}

func (w *watcher) waitFor(t *testing.T, substr string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	b := make([]byte, 4096)
	for time.Now().Before(deadline) {
		n, err := w.r.Read(b)
		w.buf.Write(b[:n])
		if strings.Contains(w.buf.String(), substr) {
			return
		}
		if err != nil && err != io.EOF {
			t.Fatalf("read output: %v", err)
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timeout waiting for %q in output", substr)
}

func TestAppBootsAndQuits(t *testing.T) {
	tm := teatest.NewTestModel(t, loadSample(t), teatest.WithInitialTermSize(120, 40))
	w := &watcher{r: tm.Output()}

	w.waitFor(t, "postgo", 3*time.Second)
	w.waitFor(t, "Collection", 3*time.Second)
	w.waitFor(t, "Request", 3*time.Second)
	w.waitFor(t, "Response", 3*time.Second)

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

func TestLoadRequestIntoEditor(t *testing.T) {
	tm := teatest.NewTestModel(t, loadSample(t), teatest.WithInitialTermSize(120, 40))
	w := &watcher{r: tm.Output()}

	w.waitFor(t, "quotes by author", 3*time.Second)

	tm.Send(tea.KeyMsg{Type: tea.KeyDown})
	tm.Send(tea.KeyMsg{Type: tea.KeyDown})
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	w.waitFor(t, "{{host}}/api/quotes/author", 3*time.Second)

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

func TestSectionNavigation(t *testing.T) {
	tm := teatest.NewTestModel(t, loadSample(t), teatest.WithInitialTermSize(120, 40))
	w := &watcher{r: tm.Output()}

	// enter loads the request and focuses the URL bar
	w.waitFor(t, "quotes by author", 3*time.Second)
	tm.Send(tea.KeyMsg{Type: tea.KeyDown})
	tm.Send(tea.KeyMsg{Type: tea.KeyDown})
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	w.waitFor(t, "{{host}}/api/quotes/author", 3*time.Second)

	// tab into the editor (Query section by default), then ctrl+n twice
	// to Body; the empty body renders its placeholder
	tm.Send(tea.KeyMsg{Type: tea.KeyTab})
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlN})
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlN})
	w.waitFor(t, `{"hello": "world"}`, 3*time.Second)

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

func TestQueryTab(t *testing.T) {
	tm := teatest.NewTestModel(t, loadSample(t), teatest.WithInitialTermSize(120, 40))
	w := &watcher{r: tm.Output()}

	w.waitFor(t, "quotes by author", 3*time.Second)
	tm.Send(tea.KeyMsg{Type: tea.KeyDown})
	tm.Send(tea.KeyMsg{Type: tea.KeyDown})
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	w.waitFor(t, "{{host}}/api/quotes/author", 3*time.Second)

	// tab into the editor lands on the first tab (Query); the empty
	// query textarea shows its placeholder
	tm.Send(tea.KeyMsg{Type: tea.KeyTab})
	w.waitFor(t, "tag: news", 3*time.Second)

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

func TestScriptsTabEditsHooks(t *testing.T) {
	tm := teatest.NewTestModel(t, loadSample(t), teatest.WithInitialTermSize(120, 40))
	w := &watcher{r: tm.Output()}

	w.waitFor(t, "quotes by author", 3*time.Second)
	tm.Send(tea.KeyMsg{Type: tea.KeyDown})
	tm.Send(tea.KeyMsg{Type: tea.KeyDown})
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	w.waitFor(t, "{{host}}/api/quotes/author", 3*time.Second)

	// tab into the editor (Query), then ctrl+n four times to Scripts
	tm.Send(tea.KeyMsg{Type: tea.KeyTab})
	for i := 0; i < 4; i++ {
		tm.Send(tea.KeyMsg{Type: tea.KeyCtrlN})
	}
	w.waitFor(t, "Scripts", 3*time.Second)
	w.waitFor(t, "pre", 3*time.Second)

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

func TestBarEnterSendsAndEscReturns(t *testing.T) {
	tm := teatest.NewTestModel(t, loadSample(t), teatest.WithInitialTermSize(120, 40))
	w := &watcher{r: tm.Output()}

	w.waitFor(t, "quotes by author", 3*time.Second)
	tm.Send(tea.KeyMsg{Type: tea.KeyDown})
	tm.Send(tea.KeyMsg{Type: tea.KeyDown})
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	w.waitFor(t, "{{host}}/api/quotes/author", 3*time.Second)

	// enter in the bar sends; no env active, so the send fails with the
	// unresolved-placeholder error
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	w.waitFor(t, "unresolved placeholder", 3*time.Second)

	// esc returns focus to the pane before the bar (sidebar)
	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})
	w.waitFor(t, "enter load", 3*time.Second)

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

func TestPasteCurlImportAndExport(t *testing.T) {
	tm := teatest.NewTestModel(t, loadSample(t), teatest.WithInitialTermSize(120, 40))
	w := &watcher{r: tm.Output()}

	w.waitFor(t, "Collection", 3*time.Second)

	// jump to the URL bar, paste a curl command — it is imported
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlL})
	tm.Send(tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune(`curl -X POST -H 'X-Token: abc' -d '{"a":1}' https://api.test/things`),
		Paste: true,
	})
	w.waitFor(t, "https://api.test/things", 3*time.Second)
	w.waitFor(t, "imported curl request", 3*time.Second)

	// ctrl+g exports the current request as curl to the clipboard
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlG})
	w.waitFor(t, "curl copied to clipboard", 3*time.Second)

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

func TestCurlExportLineInterpolates(t *testing.T) {
	req := collection.Request{
		Method: "GET",
		URL:    "{{host}}/api/random",
		Headers: []collection.Header{
			{Name: "Accept", Value: "application/json"},
		},
	}
	got := curlExportLine(req, map[string]string{"host": "https://zenquotes.io"})
	for _, want := range []string{"https://zenquotes.io/api/random", "-H 'Accept: application/json'"} {
		if !strings.Contains(got, want) {
			t.Errorf("curlExportLine = %q, want contains %q", got, want)
		}
	}
	if strings.Contains(got, "{{host}}") {
		t.Errorf("curlExportLine should interpolate host, got %q", got)
	}

	// unknown placeholders still pass through
	got = curlExportLine(req, nil)
	if !strings.Contains(got, "{{host}}") {
		t.Errorf("no vars should keep {{host}}, got %q", got)
	}
}

func TestCommandPalette(t *testing.T) {
	tm := teatest.NewTestModel(t, loadSample(t), teatest.WithInitialTermSize(120, 40))
	w := &watcher{r: tm.Output()}

	w.waitFor(t, "Collection", 3*time.Second)

	// ctrl+/ arrives as ctrl+_ (0x1F); opens the palette
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlUnderscore})
	w.waitFor(t, "Send request", 3*time.Second)

	// navigate with arrows (the Filtering state otherwise routes every key
	// to the filter input)
	tm.Send(tea.KeyMsg{Type: tea.KeyDown})
	tm.Send(tea.KeyMsg{Type: tea.KeyDown})
	w.waitFor(t, "▸ Cycle environment", 3*time.Second)

	// filter narrows the list
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("curl")})
	w.waitFor(t, "Copy as curl", 3*time.Second)

	// clear filter, pick "Focus response", enter runs it
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlU})
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("response")})
	// filtering is async (FilterMatchesMsg); give it a beat before enter
	time.Sleep(250 * time.Millisecond)
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	w.waitFor(t, "scroll", 3*time.Second) // response pane status-bar help

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

func TestTabCyclesBodyPanesOnly(t *testing.T) {
	tm := teatest.NewTestModel(t, loadSample(t), teatest.WithInitialTermSize(120, 40))
	w := &watcher{r: tm.Output()}

	w.waitFor(t, "Collection", 3*time.Second)

	// initial focus is the sidebar; tab moves to the editor (skipping the bar)
	tm.Send(tea.KeyMsg{Type: tea.KeyTab})
	w.waitFor(t, "ctrl+n/p field", 3*time.Second)

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

func TestSidebarEnterTogglesDir(t *testing.T) {
	tm := teatest.NewTestModel(t, loadSample(t), teatest.WithInitialTermSize(120, 40))
	w := &watcher{r: tm.Output()}

	w.waitFor(t, "Collection", 3*time.Second)

	// cursor starts on the "authors" directory; enter collapses it instead
	// of loading a request, so focus stays in the sidebar (the status bar
	// help for the sidebar still shows)
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	w.waitFor(t, "enter load", 3*time.Second)

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

func TestAddRequestInFolder(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := os.MkdirAll(filepath.Join(root, "authors"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "quotes"), 0o755); err != nil {
		t.Fatal(err)
	}
	entries, err := collection.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	tm := teatest.NewTestModel(t, New(root, entries, nil, nil, session.State{}), teatest.WithInitialTermSize(120, 40))
	w := &watcher{r: tm.Output()}

	w.waitFor(t, "authors", 3*time.Second)

	// cursor starts on the collection root; move onto the authors folder
	tm.Send(tea.KeyMsg{Type: tea.KeyDown})
	// press a to open the namer
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	w.waitFor(t, "new request name", 3*time.Second)

	// type the name and confirm
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("create post")})
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	w.waitFor(t, "created ", 3*time.Second)

	// the file exists with the request name and focus moved to the URL bar
	path := filepath.Join(root, "authors", "create-post.yaml")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected created request file: %v", err)
	}
	req, err := collection.LoadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if req.Name != "create post" {
		t.Errorf("expected name 'create post', got %q", req.Name)
	}

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

func TestAddRequestOnRequestInFolder(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := os.MkdirAll(filepath.Join(root, "authors"), 0o755); err != nil {
		t.Fatal(err)
	}
	seed := &collection.Request{Name: "list authors", Method: "GET", URL: "https://api.test/authors"}
	if _, err := collection.Save(root, filepath.Join(root, "authors", "list.yaml"), seed); err != nil {
		t.Fatal(err)
	}
	entries, err := collection.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	tm := teatest.NewTestModel(t, New(root, entries, nil, nil, session.State{}), teatest.WithInitialTermSize(120, 40))
	w := &watcher{r: tm.Output()}

	w.waitFor(t, "list authors", 3*time.Second)

	// tree: collection root, authors/, list authors. Cursor on the root;
	// move onto the request inside authors
	tm.Send(tea.KeyMsg{Type: tea.KeyDown})
	tm.Send(tea.KeyMsg{Type: tea.KeyDown})
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	w.waitFor(t, "new request name", 3*time.Second)

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("create post")})
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	w.waitFor(t, "created ", 3*time.Second)

	// the new request lands in the same folder as the highlighted request
	path := filepath.Join(root, "authors", "create-post.yaml")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected created request file in authors/: %v", err)
	}

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

func TestAddFolderWithSlash(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := os.MkdirAll(filepath.Join(root, "authors"), 0o755); err != nil {
		t.Fatal(err)
	}
	entries, err := collection.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	tm := teatest.NewTestModel(t, New(root, entries, nil, nil, session.State{}), teatest.WithInitialTermSize(120, 40))
	w := &watcher{r: tm.Output()}

	w.waitFor(t, "authors", 3*time.Second)

	// cursor on the collection root; open the namer and type a leading /
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	w.waitFor(t, "new request name", 3*time.Second)
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/v2")})
	w.waitFor(t, "new collection name", 3*time.Second)

	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	w.waitFor(t, "created ", 3*time.Second)

	// the folder exists at the collection root
	dir := filepath.Join(root, "v2")
	if fi, err := os.Stat(dir); err != nil || !fi.IsDir() {
		t.Fatalf("expected created folder %s: %v", dir, err)
	}

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

func TestRestoreSessionState(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	entries, err := collection.Load("../../sample-collections")
	if err != nil {
		t.Fatal(err)
	}
	envs, names, err := collection.LoadEnvironments("../../sample-collections")
	if err != nil {
		t.Fatal(err)
	}
	st := session.State{
		Env:        "prod",
		ActivePath: "quotes/random.yaml",
		Collapsed:  []string{"authors"},
	}
	tm := teatest.NewTestModel(t, New("../../sample-collections", entries, envs, names, st), teatest.WithInitialTermSize(120, 40))
	w := &watcher{r: tm.Output()}

	// environment restored
	w.waitFor(t, "env: prod", 3*time.Second)
	// collapsed folder: authors subtree hidden
	w.waitFor(t, "authors", 3*time.Second)
	if strings.Contains(w.buf.String(), "quotes by author") {
		t.Errorf("collapsed authors should hide its requests")
	}
	// active request restored into the editor
	w.waitFor(t, "{{host}}/api/random", 3*time.Second)

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

func TestSessionSnapshot(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	m := loadSample(t)
	m.cycleEnv() // none -> dev
	m.cycleEnv() // dev -> prod
	m.sidebar.ToggleCollapsed()
	st := m.snapshot()
	if st.Env != "prod" {
		t.Errorf("expected env prod, got %q", st.Env)
	}
	if len(st.Collapsed) != 3 {
		t.Errorf("expected all top-level dirs collapsed, got %v", st.Collapsed)
	}
}

func TestDeleteRequest(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	seed := &collection.Request{Name: "list authors", Method: "GET", URL: "https://api.test/authors"}
	if _, err := collection.Save(root, filepath.Join(root, "list.yaml"), seed); err != nil {
		t.Fatal(err)
	}
	entries, err := collection.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	tm := teatest.NewTestModel(t, New(root, entries, nil, nil, session.State{}), teatest.WithInitialTermSize(120, 40))
	w := &watcher{r: tm.Output()}

	w.waitFor(t, "list authors", 3*time.Second)

	// tree: collection root, list authors. Move onto the request, press d,
	// confirm with y.
	tm.Send(tea.KeyMsg{Type: tea.KeyDown})
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	w.waitFor(t, "delete request", 3*time.Second)
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	w.waitFor(t, "deleted ", 3*time.Second)

	if _, err := os.Stat(filepath.Join(root, "list.yaml")); !os.IsNotExist(err) {
		t.Errorf("expected request file deleted: %v", err)
	}

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

func TestDeleteCancelKeepsRequest(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	seed := &collection.Request{Name: "list authors", Method: "GET", URL: "https://api.test/authors"}
	if _, err := collection.Save(root, filepath.Join(root, "list.yaml"), seed); err != nil {
		t.Fatal(err)
	}
	entries, err := collection.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	tm := teatest.NewTestModel(t, New(root, entries, nil, nil, session.State{}), teatest.WithInitialTermSize(120, 40))
	w := &watcher{r: tm.Output()}

	w.waitFor(t, "list authors", 3*time.Second)

	tm.Send(tea.KeyMsg{Type: tea.KeyDown})
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	w.waitFor(t, "delete request", 3*time.Second)
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	w.waitFor(t, "y yes · n no", 3*time.Second) // confirm modal closed

	if _, err := os.Stat(filepath.Join(root, "list.yaml")); err != nil {
		t.Errorf("expected request file kept after cancel: %v", err)
	}

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

func TestDeleteFolder(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := os.MkdirAll(filepath.Join(root, "authors"), 0o755); err != nil {
		t.Fatal(err)
	}
	seed := &collection.Request{Name: "list authors", Method: "GET", URL: "https://api.test/authors"}
	if _, err := collection.Save(root, filepath.Join(root, "authors", "list.yaml"), seed); err != nil {
		t.Fatal(err)
	}
	entries, err := collection.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	tm := teatest.NewTestModel(t, New(root, entries, nil, nil, session.State{}), teatest.WithInitialTermSize(120, 40))
	w := &watcher{r: tm.Output()}

	w.waitFor(t, "authors", 3*time.Second)

	// cursor starts on the collection root; move onto the authors folder,
	// press d, confirm with y
	tm.Send(tea.KeyMsg{Type: tea.KeyDown})
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	w.waitFor(t, "delete folder", 3*time.Second)
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	w.waitFor(t, "deleted ", 3*time.Second)

	if fi, err := os.Stat(filepath.Join(root, "authors")); !os.IsNotExist(err) {
		t.Errorf("expected authors folder gone, got %v", fi)
	}

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

func TestRenameRequest(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	seed := &collection.Request{Name: "list authors", Method: "GET", URL: "https://api.test/authors"}
	if _, err := collection.Save(root, filepath.Join(root, "list.yaml"), seed); err != nil {
		t.Fatal(err)
	}
	entries, err := collection.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	tm := teatest.NewTestModel(t, New(root, entries, nil, nil, session.State{}), teatest.WithInitialTermSize(120, 40))
	w := &watcher{r: tm.Output()}

	w.waitFor(t, "list authors", 3*time.Second)

	// move onto the request, press r, confirm the pre-filled name
	tm.Send(tea.KeyMsg{Type: tea.KeyDown})
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	w.waitFor(t, "new request name", 3*time.Second)
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlU}) // clear pre-filled value
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("renamed thing")})
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	w.waitFor(t, "renamed ", 3*time.Second)

	if _, err := os.Stat(filepath.Join(root, "renamed-thing.yaml")); err != nil {
		t.Fatalf("expected renamed file: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "list.yaml")); !os.IsNotExist(err) {
		t.Errorf("expected old file gone: %v", err)
	}
	req, err := collection.LoadFile(filepath.Join(root, "renamed-thing.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if req.Name != "renamed thing" {
		t.Errorf("expected name 'renamed thing', got %q", req.Name)
	}

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

func TestPaletteClearsChainStore(t *testing.T) {
	tm := teatest.NewTestModel(t, loadSample(t), teatest.WithInitialTermSize(120, 40))
	w := &watcher{r: tm.Output()}

	w.waitFor(t, "Collection", 3*time.Second)

	// open the palette, filter to "clear", run it
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlUnderscore})
	w.waitFor(t, "Send request", 3*time.Second)
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("clear")})
	w.waitFor(t, "Clear chain store", 3*time.Second)
	time.Sleep(250 * time.Millisecond)
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	w.waitFor(t, "chain store cleared", 3*time.Second)

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

func TestSwitchTheme(t *testing.T) {
	tm := teatest.NewTestModel(t, loadSample(t), teatest.WithInitialTermSize(120, 40))
	w := &watcher{r: tm.Output()}

	w.waitFor(t, "Collection", 3*time.Second)

	// open the palette, filter to "theme", run Switch theme
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlUnderscore})
	w.waitFor(t, "Send request", 3*time.Second)
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("theme")})
	w.waitFor(t, "Switch theme", 3*time.Second)
	time.Sleep(250 * time.Millisecond)
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	w.waitFor(t, "solarized", 3*time.Second) // theme picker lists presets

	// select solarized (third item; nav down twice then enter)
	tm.Send(tea.KeyMsg{Type: tea.KeyDown})
	tm.Send(tea.KeyMsg{Type: tea.KeyDown})
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	w.waitFor(t, "theme: solarized", 3*time.Second)

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

func TestEnvManagerTabsAndActivate(t *testing.T) {
	tm := teatest.NewTestModel(t, loadSample(t), teatest.WithInitialTermSize(120, 40))
	w := &watcher{r: tm.Output()}

	w.waitFor(t, "env: none", 3*time.Second)

	// open the env manager; no "none" tab, so it starts on dev
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlUnderscore})
	w.waitFor(t, "Send request", 3*time.Second)
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("environ")})
	w.waitFor(t, "Environments", 3*time.Second)
	time.Sleep(250 * time.Millisecond)
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	w.waitFor(t, "dev", 3*time.Second) // tab bar shows dev (no none)
	w.waitFor(t, "host = ", 3*time.Second)

	// enter activates dev and closes
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	w.waitFor(t, "env: dev", 3*time.Second)

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

func TestEnvManagerAddVariable(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := os.MkdirAll(filepath.Join(root, "environments"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := collection.SaveEnvironment(root, "dev", map[string]string{"host": "https://dev.test"}); err != nil {
		t.Fatal(err)
	}
	seed := &collection.Request{Name: "thing", Method: "GET", URL: "https://api.test/things"}
	if _, err := collection.Save(root, filepath.Join(root, "thing.yaml"), seed); err != nil {
		t.Fatal(err)
	}
	entries, _ := collection.Load(root)
	envs, names, _ := collection.LoadEnvironments(root)
	tm := teatest.NewTestModel(t, New(root, entries, envs, names, session.State{}), teatest.WithInitialTermSize(120, 40))
	w := &watcher{r: tm.Output()}

	w.waitFor(t, "thing", 3*time.Second)

	// open the env manager; starts on dev (first env, no "none" tab)
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlUnderscore})
	w.waitFor(t, "Send request", 3*time.Second)
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("environ")})
	w.waitFor(t, "Environments", 3*time.Second)
	time.Sleep(250 * time.Millisecond)
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	w.waitFor(t, "host = ", 3*time.Second)

	// add a variable to the dev tab via key=value namer; the modal must
	// say "new variable" (not the request-naming label) with a key=value
	// placeholder
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	w.waitFor(t, "new variable", 3*time.Second)
	w.waitFor(t, "key=value", 3*time.Second)
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("timeout=10")})
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	w.waitFor(t, "timeout = 10", 3*time.Second)

	// verify persisted
	data, err := os.ReadFile(filepath.Join(root, "environments", "dev.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "timeout") {
		t.Errorf("timeout not persisted:\n%s", data)
	}

	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

func TestEnvManagerAddEnvironment(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := os.MkdirAll(filepath.Join(root, "environments"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := collection.SaveEnvironment(root, "dev", map[string]string{"host": "https://dev.test"}); err != nil {
		t.Fatal(err)
	}
	seed := &collection.Request{Name: "thing", Method: "GET", URL: "https://api.test/things"}
	if _, err := collection.Save(root, filepath.Join(root, "thing.yaml"), seed); err != nil {
		t.Fatal(err)
	}
	entries, _ := collection.Load(root)
	envs, names, _ := collection.LoadEnvironments(root)
	tm := teatest.NewTestModel(t, New(root, entries, envs, names, session.State{}), teatest.WithInitialTermSize(120, 40))
	w := &watcher{r: tm.Output()}

	w.waitFor(t, "thing", 3*time.Second)

	// open the env manager
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlUnderscore})
	w.waitFor(t, "Send request", 3*time.Second)
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("environ")})
	w.waitFor(t, "Environments", 3*time.Second)
	time.Sleep(250 * time.Millisecond)
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	w.waitFor(t, "host = ", 3*time.Second)

	// a leading / in the add-variable namer means a new environment
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	w.waitFor(t, "new variable", 3*time.Second)
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/staging")})
	w.waitFor(t, "new environment name", 3*time.Second)
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	// the notice only appears once the env is created (the "/staging"
	// text is already in the output buffer from the namer view, so it
	// can't be used to wait)
	w.waitFor(t, "environment staging created", 3*time.Second)

	// persisted as an empty environment file
	data, err := os.ReadFile(filepath.Join(root, "environments", "staging.yaml"))
	if err != nil {
		t.Fatalf("staging env not persisted: %v", err)
	}
	if strings.Contains(string(data), "dev.test") {
		t.Errorf("staging env should start empty:\n%s", data)
	}

	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

func TestEnvManagerFilter(t *testing.T) {
	tm := teatest.NewTestModel(t, loadSample(t), teatest.WithInitialTermSize(120, 40))
	w := &watcher{r: tm.Output()}

	w.waitFor(t, "env: none", 3*time.Second)

	// open the env manager (starts on dev, listing its variables)
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlUnderscore})
	w.waitFor(t, "Send request", 3*time.Second)
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("environ")})
	w.waitFor(t, "Environments", 3*time.Second)
	time.Sleep(250 * time.Millisecond)
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	w.waitFor(t, "host = ", 3*time.Second)

	// "/" enters filter mode: typing "a" must NOT open the add-namer
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	time.Sleep(250 * time.Millisecond)
	if strings.Contains(w.buf.String(), "new request name") {
		t.Error("typing 'a' in filter mode should not open the add-namer")
	}

	// esc exits filter mode, esc closes the manager
	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})
	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

func TestCycleEnvironment(t *testing.T) {
	tm := teatest.NewTestModel(t, loadSample(t), teatest.WithInitialTermSize(120, 40))
	w := &watcher{r: tm.Output()}

	w.waitFor(t, "env: none", 3*time.Second)

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlE})
	w.waitFor(t, "env: dev", 3*time.Second)

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlE})
	w.waitFor(t, "env: prod", 3*time.Second)

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

func TestNewRequest(t *testing.T) {
	tm := teatest.NewTestModel(t, loadSample(t), teatest.WithInitialTermSize(120, 40))
	w := &watcher{r: tm.Output()}

	w.waitFor(t, "Collection", 3*time.Second)

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	w.waitFor(t, "ctrl+t method", 3*time.Second)

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}
