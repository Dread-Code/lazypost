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

	w.waitFor(t, "quotes by author", 3*time.Second)
	tm.Send(tea.KeyMsg{Type: tea.KeyDown})
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	w.waitFor(t, "{{host}}/api/quotes/author", 3*time.Second)

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlN})
	w.waitFor(t, "Accept: application/json", 3*time.Second)

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
