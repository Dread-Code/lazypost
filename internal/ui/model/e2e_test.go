package model

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"

	"lazypost/internal/collection"
	"lazypost/internal/session"
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

func TestChainStoreAcrossSends(t *testing.T) {
	var mu sync.Mutex
	var headers []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		headers = append(headers, r.Header.Get("X-Chain"))
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	root := t.TempDir()
	req := collection.Request{
		Name:   "chain",
		Method: "GET",
		URL:    srv.URL + "/ping",
		// pre reads the store; post writes it
		Pre:  `if store.get("token") ~= nil then req.headers["X-Chain"] = store.get("token") end`,
		Post: `store.set("token", "chained-value") return true`,
	}
	if _, err := collection.Save(root, filepath.Join(root, "chain.yaml"), &req); err != nil {
		t.Fatal(err)
	}
	entries, _ := collection.Load(root)

	tm := teatest.NewTestModel(t, New(root, entries, nil, nil, session.State{}), teatest.WithInitialTermSize(120, 40))
	w := &watcher{r: tm.Output()}
	w.waitFor(t, "chain", 3*time.Second)

	tm.Send(tea.KeyMsg{Type: tea.KeyDown})  // onto the request
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter}) // load it
	// send, wait for it to reach the server, then send again so the second
	// sees the store populated by the first's post hook
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlR})
	waitForN(t, &mu, &headers, 1, 5*time.Second)
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlR})
	waitForN(t, &mu, &headers, 2, 5*time.Second)

	mu.Lock()
	defer mu.Unlock()
	if len(headers) != 2 {
		t.Fatalf("expected 2 requests, got %d", len(headers))
	}
	if headers[0] != "" {
		t.Errorf("first request should have no X-Chain, got %q", headers[0])
	}
	if headers[1] != "chained-value" {
		t.Errorf("second request should carry the chained value, got %q", headers[1])
	}

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

func waitForN(t *testing.T, mu *sync.Mutex, headers *[]string, n int, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		mu.Lock()
		c := len(*headers)
		mu.Unlock()
		if c >= n {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("timeout waiting for %d requests", n)
}

func TestRequestHistory(t *testing.T) {
	var mu sync.Mutex
	goodHits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		goodHits++
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	root := t.TempDir()
	good := collection.Request{Name: "good", Method: "GET", URL: srv.URL + "/good"}
	if _, err := collection.Save(root, filepath.Join(root, "good.yaml"), &good); err != nil {
		t.Fatal(err)
	}
	// a request that fails (connection refused)
	bad := collection.Request{Name: "bad", Method: "GET", URL: "http://127.0.0.1:1/bad"}
	if _, err := collection.Save(root, filepath.Join(root, "bad.yaml"), &bad); err != nil {
		t.Fatal(err)
	}
	entries, _ := collection.Load(root)

	tm := teatest.NewTestModel(t, New(root, entries, nil, nil, session.State{}), teatest.WithInitialTermSize(120, 40))
	w := &watcher{r: tm.Output()}

	// bad sorts first; send it to get an error entry
	tm.Send(tea.KeyMsg{Type: tea.KeyDown})  // -> bad
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlR}) // send, fails
	w.waitFor(t, "connection refused", 5*time.Second)

	// then good, which hits the server
	tm.Send(tea.KeyMsg{Type: tea.KeyDown})  // -> good
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlR}) // send
	waitHits(t, &mu, &goodHits, 1)

	// open history: newest first, so good is on top
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlH})
	w.waitFor(t, "Request history", 3*time.Second)
	w.waitFor(t, "good", 3*time.Second)

	// enter loads the top entry (good) into the editor + URL bar
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	w.waitFor(t, srv.URL+"/good", 3*time.Second)

	// resend with ctrl+r from the bar
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlR})
	waitHits(t, &mu, &goodHits, 2)

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

func waitHits(t *testing.T, mu *sync.Mutex, hits *int, want int) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		n := *hits
		mu.Unlock()
		if n >= want {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	mu.Lock()
	defer mu.Unlock()
	t.Fatalf("timeout: hits = %d, want %d", *hits, want)
}
