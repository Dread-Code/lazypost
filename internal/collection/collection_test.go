package collection

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSaveLoadRoundTrip(t *testing.T) {
	root := t.TempDir()

	in := &Request{
		Name:   "create thing",
		Method: "POST",
		URL:    "https://api.test/things",
		Headers: []Header{
			{Name: "Content-Type", Value: "application/json"},
		},
		Auth: &Auth{Type: "bearer", Token: "tok"},
		Body: `{"a": 1}`,
	}

	path, err := Save(root, "", in)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if filepath.Base(path) != "create-thing.yaml" {
		t.Errorf("slug path = %q", path)
	}

	out, err := LoadFile(path)
	if err != nil {
		t.Fatalf("LoadFile: %v", err)
	}
	if out.Name != in.Name || out.Method != in.Method || out.URL != in.URL || out.Body != in.Body {
		t.Errorf("round trip mismatch: %+v", out)
	}
	if out.Auth == nil || out.Auth.Token != "tok" {
		t.Errorf("auth mismatch: %+v", out.Auth)
	}
	if len(out.Headers) != 1 || out.Headers[0].Name != "Content-Type" {
		t.Errorf("headers mismatch: %+v", out.Headers)
	}
}

func TestSaveRequiresName(t *testing.T) {
	if _, err := Save(t.TempDir(), "", &Request{URL: "https://x.test"}); err == nil {
		t.Error("expected error for missing name")
	}
}

func TestLoadTree(t *testing.T) {
	root := t.TempDir()
	mustWrite := func(rel, content string) {
		t.Helper()
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	mustWrite("posts/get.yaml", "name: get posts\nmethod: GET\nurl: https://x.test/posts\n")
	mustWrite("posts/sub/deep.yaml", "method: DELETE\nurl: https://x.test/posts/1\n")
	mustWrite("users/get.yaml", "url: https://x.test/users\n")
	mustWrite("README.md", "not yaml")
	mustWrite("environments/dev.yaml", "variables:\n  host: https://x.test\n")

	entries, err := Load(root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	// walk order: posts/, posts/get.yaml, posts/sub/, posts/sub/deep.yaml,
	// users/, users/get.yaml — environments/ skipped, README.md ignored.
	if len(entries) != 6 {
		t.Fatalf("expected 6 entries, got %d", len(entries))
	}
	if entries[0].Kind != Dir || entries[0].Name != "posts" || entries[0].Depth != 0 {
		t.Errorf("entries[0] = %+v", entries[0])
	}
	if entries[1].Kind != Req || entries[1].Name != "get posts" || entries[1].Depth != 1 {
		t.Errorf("entries[1] = %+v", entries[1])
	}
	if entries[2].Kind != Dir || entries[2].Name != "sub" || entries[2].Depth != 1 {
		t.Errorf("entries[2] = %+v", entries[2])
	}
	if entries[3].Kind != Req || entries[3].Name != "deep" || entries[3].Req.Method != "DELETE" || entries[3].Depth != 2 {
		t.Errorf("entries[3] = %+v", entries[3])
	}
	if entries[5].Kind != Req || entries[5].Name != "get" || entries[5].Req.Method != "GET" {
		t.Errorf("entries[5] = %+v (default method should be GET, name from filename)", entries[5])
	}
}

func TestLoadSkipsHiddenEntries(t *testing.T) {
	root := t.TempDir()
	mustWrite := func(rel, content string) {
		t.Helper()
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	mustWrite("app.yaml", "name: app\nmethod: GET\nurl: https://x.test\n")
	mustWrite(".git/config", "git stuff")
	mustWrite(".git/hooks/x", "")
	mustWrite("node_modules/pkg/index.yaml", "name: from node_modules\nmethod: GET\nurl: https://x.test/n\n")
	mustWrite(".hidden.yaml", "name: hidden\nmethod: GET\nurl: https://x.test/h\n")

	entries, err := Load(root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	// .git and .hidden.yaml are skipped; node_modules is a real (non-dot)
	// directory and stays
	if len(entries) != 4 {
		t.Fatalf("expected 4 entries, got %d: %+v", len(entries), entries)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name, ".") || strings.Contains(e.Path, ".git") {
			t.Errorf("hidden entry leaked through: %+v", e)
		}
	}
	if entries[0].Name != "app" {
		t.Errorf("entries[0] = %+v", entries[0])
	}
}

func TestSaveEnvironment(t *testing.T) {
	root := t.TempDir()
	if err := SaveEnvironment(root, "staging", map[string]string{"host": "https://staging.test"}); err != nil {
		t.Fatal(err)
	}
	envs, names, err := LoadEnvironments(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != 1 || names[0] != "staging" {
		t.Errorf("names = %v", names)
	}
	if envs["staging"]["host"] != "https://staging.test" {
		t.Errorf("vars = %v", envs["staging"])
	}
}

func TestLoadEnvironments(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "environments")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(dir, "dev.yaml"), []byte("variables:\n  host: https://dev.test\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "prod.yaml"), []byte("variables:\n  host: https://prod.test\n"), 0o644)

	envs, names, err := LoadEnvironments(root)
	if err != nil {
		t.Fatalf("LoadEnvironments: %v", err)
	}
	if len(names) != 2 || names[0] != "dev" || names[1] != "prod" {
		t.Errorf("names = %v", names)
	}
	if envs["dev"]["host"] != "https://dev.test" {
		t.Errorf("dev vars = %v", envs["dev"])
	}

	envs, names, err = LoadEnvironments(t.TempDir())
	if err != nil || len(names) != 0 || len(envs) != 0 {
		t.Errorf("missing dir should be empty, got %v %v %v", envs, names, err)
	}
}

func TestRename(t *testing.T) {
	root := t.TempDir()
	old := filepath.Join(root, "old-name.yaml")
	if _, err := Save(root, old, &Request{Name: "old name", Method: "GET", URL: "https://api.test/x"}); err != nil {
		t.Fatal(err)
	}

	req, newPath, err := Rename(root, old, "new name")
	if err != nil {
		t.Fatalf("Rename: %v", err)
	}
	if req.Name != "new name" {
		t.Errorf("renamed request name = %q", req.Name)
	}
	if filepath.Base(newPath) != "new-name.yaml" {
		t.Errorf("new path = %q, want new-name.yaml", newPath)
	}
	if _, err := os.Stat(old); !os.IsNotExist(err) {
		t.Error("old file should be removed")
	}
	out, err := LoadFile(newPath)
	if err != nil {
		t.Fatal(err)
	}
	if out.Name != "new name" {
		t.Errorf("on-disk name = %q, want new name", out.Name)
	}
}

func TestDelete(t *testing.T) {
	root := t.TempDir()
	if _, err := Save(root, filepath.Join(root, "a.yaml"), &Request{Name: "a", Method: "GET"}); err != nil {
		t.Fatal(err)
	}
	if err := Delete(root, filepath.Join(root, "a.yaml")); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "a.yaml")); !os.IsNotExist(err) {
		t.Error("file should be gone")
	}

	// protected paths are refused
	if err := Delete(root, root); err == nil {
		t.Error("expected error deleting the collection root")
	}
	os.MkdirAll(filepath.Join(root, "environments"), 0o755)
	if err := Delete(root, filepath.Join(root, "environments")); err == nil {
		t.Error("expected error deleting environments/")
	}
}

func TestSlug(t *testing.T) {
	cases := map[string]string{
		"Create Post!":  "create-post",
		"  get_user  ":  "get-user",
		"Weird--Chars$": "weird-chars",
	}
	for in, want := range cases {
		if got := Slug(in); got != want {
			t.Errorf("Slug(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestDefaultName(t *testing.T) {
	cases := map[string]string{
		"https://api.test/posts/42":       "42",
		"https://api.test/posts/My Post":  "my-post",
		"https://api.test/":               "untitled",
		"https://api.test":                "untitled",
		"https://api.test/posts/":         "posts",
		"a bare relative path with space": "a-bare-relative-path-with-space",
	}
	for in, want := range cases {
		if got := DefaultName(in); got != want {
			t.Errorf("DefaultName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestMarkerRoundTrip(t *testing.T) {
	dir := t.TempDir()
	if err := WriteMarker(dir, "My API collection"); err != nil {
		t.Fatalf("WriteMarker: %v", err)
	}
	m, err := LoadMarker(dir)
	if err != nil {
		t.Fatalf("LoadMarker: %v", err)
	}
	if m == nil || m.Name != "My API collection" || m.Root != "" {
		t.Fatalf("LoadMarker = %+v, want name %q with empty root", m, "My API collection")
	}
}

func TestMarkerAbsent(t *testing.T) {
	dir := t.TempDir()
	m, err := LoadMarker(dir)
	if err != nil {
		t.Fatalf("LoadMarker: %v", err)
	}
	if m != nil {
		t.Fatalf("LoadMarker = %+v, want nil for a markerless dir", m)
	}
}

func TestMarkerMalformed(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, MarkerFile), []byte("{{not yaml"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadMarker(dir); err == nil {
		t.Fatal("LoadMarker on malformed marker: want error, got nil")
	}
}

func TestMarkerNotInTree(t *testing.T) {
	dir := t.TempDir()
	if err := WriteMarker(dir, "n"); err != nil {
		t.Fatal(err)
	}
	entries, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	for _, e := range entries {
		if e.Name == MarkerFile {
			t.Fatalf("Load included the marker %q as a tree entry", MarkerFile)
		}
	}
}
