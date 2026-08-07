package model

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"

	"postgo/internal/collection"
	"postgo/internal/session"
)

func TestSendRequestEndToEnd(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message":"hello-from-e2e"}`))
	}))
	defer srv.Close()

	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	root := t.TempDir()
	req := collection.Request{
		Name:   "e2e",
		Method: "GET",
		URL:    srv.URL + "/ping",
	}
	if _, err := collection.Save(root, filepath.Join(root, "e2e.yaml"), &req); err != nil {
		t.Fatalf("seed request: %v", err)
	}

	entries, err := collection.Load(root)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	envs, names, _ := collection.LoadEnvironments(root)
	_ = os.Unsetenv("http_proxy")

	tm := teatest.NewTestModel(t, New(root, entries, envs, names, session.State{}), teatest.WithInitialTermSize(120, 40))
	w := &watcher{r: tm.Output()}

	w.waitFor(t, "e2e", 3*time.Second)

	// the sidebar now leads with the collection root; move to the request
	tm.Send(tea.KeyMsg{Type: tea.KeyDown})
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	w.waitFor(t, srv.URL+"/ping", 3*time.Second)

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlR})
	w.waitFor(t, "200 OK", 5*time.Second)
	w.waitFor(t, "hello-from-e2e", 5*time.Second)

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

func TestPreHookAddsHeader(t *testing.T) {
	got := make(chan string, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got <- r.Header.Get("X-Hook")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	root := t.TempDir()
	req := collection.Request{
		Name:   "hooked",
		Method: "GET",
		URL:    srv.URL + "/ping",
		Pre:    `req.headers["X-Hook"] = "from-lua"`,
	}
	if _, err := collection.Save(root, filepath.Join(root, "hooked.yaml"), &req); err != nil {
		t.Fatal(err)
	}
	entries, _ := collection.Load(root)

	tm := teatest.NewTestModel(t, New(root, entries, nil, nil, session.State{}), teatest.WithInitialTermSize(120, 40))
	w := &watcher{r: tm.Output()}
	w.waitFor(t, "hooked", 3*time.Second)

	tm.Send(tea.KeyMsg{Type: tea.KeyDown})  // onto the request
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter}) // load it
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlR}) // send
	select {
	case v := <-got:
		if v != "from-lua" {
			t.Errorf("expected X-Hook from-lua, got %q", v)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for request")
	}

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}
