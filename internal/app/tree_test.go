package app

import (
	"os"
	"path/filepath"
	"testing"

	"lazypost/internal/collection"
)

func seed(t *testing.T, root string, req *collection.Request) string {
	t.Helper()
	path, err := collection.Save(root, "", req)
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	return path
}

func TestSaveRequestReturnsFreshTree(t *testing.T) {
	root := t.TempDir()
	path, entries, err := SaveRequest(root, "", &collection.Request{
		Name: "thing", Method: "GET", URL: "https://api.test/things",
	})
	if err != nil {
		t.Fatalf("SaveRequest: %v", err)
	}
	if filepath.Base(path) != "thing.yaml" {
		t.Errorf("path = %q, want thing.yaml", path)
	}
	if len(entries) != 1 || entries[0].Name != "thing" {
		t.Errorf("tree = %+v, want the new request", entries)
	}
}

func TestRenameRequestReturnsFreshTree(t *testing.T) {
	root := t.TempDir()
	old := seed(t, root, &collection.Request{Name: "old name", Method: "GET", URL: "https://api.test/x"})

	req, newPath, entries, err := RenameRequest(root, old, "new name")
	if err != nil {
		t.Fatalf("RenameRequest: %v", err)
	}
	if req.Name != "new name" || filepath.Base(newPath) != "new-name.yaml" {
		t.Errorf("renamed = %q at %q", req.Name, newPath)
	}
	if _, err := os.Stat(old); !os.IsNotExist(err) {
		t.Error("old file should be gone")
	}
	if len(entries) != 1 || entries[0].Name != "new name" {
		t.Errorf("tree = %+v, want the renamed request", entries)
	}
}

func TestCreateFolderAndRequest(t *testing.T) {
	root := t.TempDir()

	path, entries, err := CreateFolder(root, root, "v2")
	if err != nil {
		t.Fatalf("CreateFolder: %v", err)
	}
	if filepath.Base(path) != "v2" {
		t.Errorf("folder = %q, want v2", path)
	}
	if fi, err := os.Stat(path); err != nil || !fi.IsDir() {
		t.Fatalf("folder missing: %v", err)
	}
	if len(entries) != 1 || entries[0].Kind != collection.Dir || entries[0].Name != "v2" {
		t.Errorf("tree = %+v, want the new folder", entries)
	}

	req, reqPath, entries, err := CreateRequest(root, path, "list things")
	if err != nil {
		t.Fatalf("CreateRequest: %v", err)
	}
	if req.Name != "list things" || filepath.Base(reqPath) != "list-things.yaml" {
		t.Errorf("request = %q at %q", req.Name, reqPath)
	}
	if _, err := os.Stat(reqPath); err != nil {
		t.Fatalf("request file missing: %v", err)
	}
	if len(entries) != 2 {
		t.Errorf("tree should have folder + request, got %d entries", len(entries))
	}
}

func TestDeleteEntryReturnsFreshTree(t *testing.T) {
	root := t.TempDir()
	path := seed(t, root, &collection.Request{Name: "doomed", Method: "GET", URL: "https://api.test/x"})
	entries, err := collection.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	entries, err = DeleteEntry(root, &entries[0])
	if err != nil {
		t.Fatalf("DeleteEntry: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("request file should be gone")
	}
	if len(entries) != 0 {
		t.Errorf("tree should be empty, got %d entries", len(entries))
	}
}
