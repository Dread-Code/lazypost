package model

import (
	"io"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"

	"postgo/internal/collection"
)

func loadSample(t *testing.T) Model {
	t.Helper()
	entries, err := collection.Load("../../sample-collections")
	if err != nil {
		t.Fatalf("load sample collections: %v", err)
	}
	envs, names, err := collection.LoadEnvironments("../../sample-collections")
	if err != nil {
		t.Fatalf("load environments: %v", err)
	}
	return New("../../sample-collections", entries, envs, names)
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
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	w.waitFor(t, "{{host}}/api/quotes/author", 3*time.Second)

	// tab into the editor (Headers section), then ctrl+n to Body;
	// the empty body renders its placeholder
	tm.Send(tea.KeyMsg{Type: tea.KeyTab})
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlN})
	w.waitFor(t, `{"hello": "world"}`, 3*time.Second)

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

func TestBarEnterSendsAndEscReturns(t *testing.T) {
	tm := teatest.NewTestModel(t, loadSample(t), teatest.WithInitialTermSize(120, 40))
	w := &watcher{r: tm.Output()}

	w.waitFor(t, "quotes by author", 3*time.Second)
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
