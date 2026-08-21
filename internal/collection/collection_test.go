package collection

import (
	"errors"
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

func TestLoadEnvironmentsRejectsYamlExtensionCollision(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, environmentsDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"dev.yaml", "dev.yml"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("variables:\n  host: https://example.test\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if _, _, err := LoadEnvironments(root); err == nil || !strings.Contains(err.Error(), "duplicate environment name") {
		t.Fatalf("LoadEnvironments error = %v, want duplicate-name error", err)
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

func TestRenameWithSameSlugDoesNotDeleteRequest(t *testing.T) {
	root := t.TempDir()
	old := filepath.Join(root, "old-name.yaml")
	if _, err := Save(root, old, &Request{Name: "old name", Method: "GET", URL: "https://api.test/x"}); err != nil {
		t.Fatal(err)
	}

	req, newPath, err := Rename(root, old, "Old Name")
	if err != nil {
		t.Fatalf("Rename: %v", err)
	}
	if newPath != old || req.Name != "Old Name" {
		t.Fatalf("rename = %q, %q; want same path and updated name", newPath, req.Name)
	}
	got, err := LoadFile(old)
	if err != nil {
		t.Fatalf("LoadFile after same-slug rename: %v", err)
	}
	if got.Name != "Old Name" {
		t.Errorf("name = %q, want Old Name", got.Name)
	}
}

func TestRelativeRootPathsRemainWithinRoot(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "collection")
	if err := os.Mkdir(root, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(parent)

	path, err := Save("collection", "", &Request{Name: "old", Method: "GET"})
	if err != nil {
		t.Fatal(err)
	}
	if path != filepath.Join("collection", "old.yaml") {
		t.Fatalf("path = %q, want relative collection path", path)
	}
	_, newPath, err := Rename("collection", path, "new")
	if err != nil {
		t.Fatal(err)
	}
	if newPath != filepath.Join("collection", "new.yaml") {
		t.Fatalf("new path = %q, want relative collection path", newPath)
	}
}

func TestCreateOperationsDoNotReplaceExistingPaths(t *testing.T) {
	root := t.TempDir()
	if _, err := CreateFolder(root, root, "v1"); err != nil {
		t.Fatal(err)
	}
	if _, err := CreateFolder(root, root, "v1"); !errors.Is(err, ErrConflict) {
		t.Fatalf("second folder create error = %v, want ErrConflict", err)
	}

	if _, _, err := CreateRequest(root, root, "list users"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := CreateRequest(root, root, "list users"); !errors.Is(err, ErrConflict) {
		t.Fatalf("second request create error = %v, want ErrConflict", err)
	}
	if err := SaveEnvironment(root, "dev", map[string]string{"token": "old"}); err != nil {
		t.Fatal(err)
	}
	if err := CreateEnvironment(root, "dev", map[string]string{"token": "new"}); !errors.Is(err, ErrConflict) {
		t.Fatalf("second environment create error = %v, want ErrConflict", err)
	}
}

func TestCollectionRejectsPathsOutsideRoot(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "collection")
	if err := os.Mkdir(root, 0o755); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(parent, "outside.yaml")
	req := &Request{Name: "outside", Method: "GET", URL: "https://api.test"}

	if _, err := Save(root, outside, req); !errors.Is(err, ErrOutsideRoot) {
		t.Fatalf("Save outside root error = %v, want ErrOutsideRoot", err)
	}
	if err := Delete(root, outside); !errors.Is(err, ErrOutsideRoot) {
		t.Fatalf("Delete outside root error = %v, want ErrOutsideRoot", err)
	}
}

func TestCollectionRejectsSymlinkPaths(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(t.TempDir(), "target.yaml")
	if err := os.WriteFile(target, []byte("name: target\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "link.yaml")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	req := &Request{Name: "link", Method: "GET", URL: "https://api.test"}
	if _, err := Save(root, link, req); !errors.Is(err, ErrSymlink) {
		t.Fatalf("Save through symlink error = %v, want ErrSymlink", err)
	}
	if err := Delete(root, link); !errors.Is(err, ErrSymlink) {
		t.Fatalf("Delete symlink error = %v, want ErrSymlink", err)
	}
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "name: target\n" {
		t.Fatalf("symlink target changed: %q", data)
	}
}

func TestNewCollectionFilesUseRestrictivePermissions(t *testing.T) {
	root := t.TempDir()
	path, err := Save(root, "", &Request{Name: "secret", Method: "GET"})
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Errorf("request mode = %o, want 600", got)
	}
	if err := SaveEnvironment(root, "dev", map[string]string{"token": "secret"}); err != nil {
		t.Fatal(err)
	}
	info, err = os.Stat(filepath.Join(root, "environments", "dev.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Errorf("environment mode = %o, want 600", got)
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
	if err := WriteMarker(dir); err != nil {
		t.Fatalf("WriteMarker: %v", err)
	}
	m, err := LoadMarker(dir)
	if err != nil {
		t.Fatalf("LoadMarker: %v", err)
	}
	if m == nil || m.Legacy || m.Version != 1 || m.Name != "" || m.Root != "" {
		t.Fatalf("LoadMarker = %+v, want new versioned marker without legacy fields", m)
	}
	if _, err := os.Stat(filepath.Join(dir, ConfigDir, ConfigFile)); err != nil {
		t.Fatalf("new config marker missing: %v", err)
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

func TestConfigDirectoryWithoutFileIsNotMarker(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ConfigDir), 0o755); err != nil {
		t.Fatal(err)
	}
	m, err := LoadMarker(dir)
	if err != nil {
		t.Fatalf("LoadMarker: %v", err)
	}
	if m != nil {
		t.Fatalf("LoadMarker = %+v, want nil for incomplete config marker", m)
	}
}

func TestMarkerMalformed(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ConfigDir), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ConfigDir, ConfigFile), []byte("{{not yaml"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadMarker(dir); err == nil {
		t.Fatal("LoadMarker on malformed marker: want error, got nil")
	}
}

func TestLegacyMarkerFallbackAndMigration(t *testing.T) {
	dir := t.TempDir()
	legacyPath := filepath.Join(dir, MarkerFile)
	legacy := []byte("name: Old API\nroot: ../main\n")
	if err := os.WriteFile(legacyPath, legacy, 0o644); err != nil {
		t.Fatal(err)
	}
	m, err := LoadMarker(dir)
	if err != nil {
		t.Fatalf("LoadMarker: %v", err)
	}
	if m == nil || !m.Legacy || m.Name != "Old API" || m.Root != "../main" || m.LegacyPath != legacyPath {
		t.Fatalf("LoadMarker = %+v, want legacy marker details", m)
	}
	if err := MigrateLegacy(dir, m.LegacyPath); err != nil {
		t.Fatalf("MigrateLegacy: %v", err)
	}
	if _, err := os.Stat(legacyPath); !os.IsNotExist(err) {
		t.Fatalf("legacy marker still exists: %v", err)
	}
	got, err := LoadMarker(dir)
	if err != nil {
		t.Fatalf("LoadMarker after migration: %v", err)
	}
	if got == nil || got.Legacy || got.Version != 1 || got.Name != "" || got.Root != "" {
		t.Fatalf("migrated marker = %+v, want new marker without legacy fields", got)
	}
}

func TestMarkerNotInTree(t *testing.T) {
	dir := t.TempDir()
	if err := WriteMarker(dir); err != nil {
		t.Fatal(err)
	}
	entries, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	for _, e := range entries {
		if e.Name == ConfigDir {
			t.Fatalf("Load included the config directory as a tree entry")
		}
	}
}

func TestNestedConfigIsInTree(t *testing.T) {
	dir := t.TempDir()
	nested := filepath.Join(dir, "folder", ConfigDir)
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	entries, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	var found bool
	for _, e := range entries {
		if e.Path == nested && e.Kind == Dir {
			found = true
		}
	}
	if !found {
		t.Fatal("Load omitted nested config directory")
	}
}

func TestNestedEnvironmentsDirectoryIsInTree(t *testing.T) {
	dir := t.TempDir()
	nested := filepath.Join(dir, "folder", environmentsDir)
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	entries, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	var found bool
	for _, e := range entries {
		if e.Path == nested && e.Kind == Dir {
			found = true
		}
	}
	if !found {
		t.Fatal("Load omitted nested environments directory")
	}
}
