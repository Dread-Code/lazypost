package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Dread-Code/lazypost/internal/collection"
	"github.com/Dread-Code/lazypost/internal/importer"
)

func TestImportCollectionStagesAndLoads(t *testing.T) {
	result, err := importer.ParseFile(filepath.Join("..", "importer", "testdata", "postman", "collection.json"), importer.ParseOptions{
		EnvironmentFiles: []string{filepath.Join("..", "importer", "testdata", "postman", "environment.json")},
	})
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	root := filepath.Join(t.TempDir(), "imported")
	summary, err := ImportCollection(result, ImportOptions{Target: root})
	if err != nil {
		t.Fatalf("ImportCollection: %v", err)
	}
	if summary.Requests != 1 || summary.Environments != 2 {
		t.Fatalf("summary = %+v", summary)
	}
	if _, err := os.Stat(filepath.Join(root, collection.ConfigDir, collection.ConfigFile)); err != nil {
		t.Fatalf("config marker: %v", err)
	}
	entries, err := collection.Load(root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(entries) != 3 || entries[2].Req == nil || entries[0].Name != "postman-demo" || entries[1].Name != "users" {
		t.Fatalf("entries = %+v, want workspace, folder, and request", entries)
	}
	envs, names, err := collection.LoadEnvironments(root)
	if err != nil {
		t.Fatalf("LoadEnvironments: %v", err)
	}
	if len(names) != 2 || envs["dev"]["token"] != "secret" {
		t.Fatalf("environments = %#v, names = %v", envs, names)
	}
}

func TestImportDryRunAndStrict(t *testing.T) {
	result := importer.Result{
		Name:       "demo",
		Workspaces: []importer.Workspace{{Name: "demo", Requests: []importer.ImportedRequest{{Request: collection.Request{Name: "one", Method: "GET", URL: "https://example.test"}}}}},
		Warnings:   []importer.Warning{{Path: "one", Message: "unsupported script"}},
	}
	root := filepath.Join(t.TempDir(), "dry-run")
	summary, err := ImportCollection(result, ImportOptions{Target: root, DryRun: true})
	if err != nil || summary.Requests != 1 {
		t.Fatalf("dry run = %+v, err=%v", summary, err)
	}
	if _, err := os.Stat(root); !os.IsNotExist(err) {
		t.Fatalf("dry run created target: %v", err)
	}
	if _, err := ImportCollection(result, ImportOptions{Target: root, Strict: true}); err == nil {
		t.Fatal("strict import with warning succeeded")
	}
}

func TestImportRefusesExistingTargetAndForceReplaces(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "target")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	old := filepath.Join(root, "old.txt")
	if err := os.WriteFile(old, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	result := importer.Result{Workspaces: []importer.Workspace{{Name: "demo", Requests: []importer.ImportedRequest{{Request: collection.Request{Name: "new", Method: "GET", URL: "https://example.test"}}}}}}
	if _, err := ImportCollection(result, ImportOptions{Target: root}); err == nil {
		t.Fatal("existing target import succeeded without force")
	}
	if _, err := ImportCollection(result, ImportOptions{Target: root, Force: true}); err != nil {
		t.Fatalf("forced import: %v", err)
	}
	if _, err := os.Stat(old); !os.IsNotExist(err) {
		t.Fatalf("old target file remains: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "demo", "new.yaml")); err != nil {
		t.Fatalf("new request missing: %v", err)
	}
}

func TestImportCollisionGetsDeterministicSuffix(t *testing.T) {
	result := importer.Result{Workspaces: []importer.Workspace{{Name: "demo", Requests: []importer.ImportedRequest{
		{Request: collection.Request{Name: "Same", Method: "GET", URL: "https://one.test"}},
		{Request: collection.Request{Name: "Same", Method: "GET", URL: "https://two.test"}},
	}}}}
	root := filepath.Join(t.TempDir(), "collision")
	summary, err := ImportCollection(result, ImportOptions{Target: root})
	if err != nil {
		t.Fatalf("ImportCollection: %v", err)
	}
	if len(summary.Warnings) != 1 || !strings.Contains(summary.Warnings[0].Message, "same-2.yaml") {
		t.Fatalf("warnings = %+v", summary.Warnings)
	}
	if _, err := os.Stat(filepath.Join(root, "demo", "same.yaml")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "demo", "same-2.yaml")); err != nil {
		t.Fatal(err)
	}
}

func TestImportPreservesWorkspacesAndNamespacesEnvironments(t *testing.T) {
	result := importer.Result{Workspaces: []importer.Workspace{
		{
			Name:         "Alpha",
			Requests:     []importer.ImportedRequest{{Request: collection.Request{Name: "List", Method: "GET", URL: "https://alpha.test"}}},
			Environments: []importer.ImportedEnvironment{{Name: "dev", Variables: map[string]string{"host": "alpha.test"}}},
		},
		{
			Name:         "Beta",
			Requests:     []importer.ImportedRequest{{Request: collection.Request{Name: "List", Method: "GET", URL: "https://beta.test"}}},
			Environments: []importer.ImportedEnvironment{{Name: "dev", Variables: map[string]string{"host": "beta.test"}}},
		},
	}}
	root := filepath.Join(t.TempDir(), "multi")
	summary, err := ImportCollection(result, ImportOptions{Target: root})
	if err != nil {
		t.Fatalf("ImportCollection: %v", err)
	}
	if summary.Folders != 2 || summary.Requests != 2 || summary.Environments != 2 {
		t.Fatalf("summary = %+v", summary)
	}
	for _, path := range []string{
		filepath.Join(root, "alpha", "list.yaml"),
		filepath.Join(root, "beta", "list.yaml"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("workspace request %s: %v", path, err)
		}
	}
	_, names, err := collection.LoadEnvironments(root)
	if err != nil {
		t.Fatalf("LoadEnvironments: %v", err)
	}
	if len(names) != 2 || names[0] != "alpha-dev" || names[1] != "beta-dev" {
		t.Fatalf("names = %v, want namespaced workspace environments", names)
	}
}

func TestImportAllocatesWorkspaceAndEnvironmentSlugCollisions(t *testing.T) {
	result := importer.Result{Workspaces: []importer.Workspace{
		{
			Name:     "Alpha",
			Requests: []importer.ImportedRequest{{Request: collection.Request{Name: "List", Method: "GET", URL: "https://alpha.test"}}},
			Environments: []importer.ImportedEnvironment{
				{Name: "Dev", Variables: map[string]string{"host": "alpha.test"}},
				{Name: "dev", Variables: map[string]string{"host": "alpha.test/second"}},
			},
		},
		{
			Name:     "alpha",
			Requests: []importer.ImportedRequest{{Request: collection.Request{Name: "List", Method: "GET", URL: "https://alpha-2.test"}}},
		},
	}}
	root := filepath.Join(t.TempDir(), "collision")
	summary, err := ImportCollection(result, ImportOptions{Target: root})
	if err != nil {
		t.Fatalf("ImportCollection: %v", err)
	}
	if len(summary.Warnings) != 2 {
		t.Fatalf("warnings = %+v, want workspace and environment collision warnings", summary.Warnings)
	}
	for _, path := range []string{
		filepath.Join(root, "alpha", "list.yaml"),
		filepath.Join(root, "alpha-2", "list.yaml"),
		filepath.Join(root, "environments", "alpha-dev.yaml"),
		filepath.Join(root, "environments", "alpha-dev-2.yaml"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected allocated import path %s: %v", path, err)
		}
	}
}

func TestImportUnscopedEnvironmentUsesSharedNamespace(t *testing.T) {
	result := importer.Result{
		Workspaces: []importer.Workspace{
			{Name: "Alpha"},
			{Name: "Beta"},
		},
		Environments: []importer.ImportedEnvironment{{Name: "dev", Variables: map[string]string{"host": "shared.test"}}},
	}
	root := filepath.Join(t.TempDir(), "shared-env")
	summary, err := ImportCollection(result, ImportOptions{Target: root})
	if err != nil {
		t.Fatalf("ImportCollection: %v", err)
	}
	if summary.Environments != 1 || len(summary.Warnings) != 1 || !strings.Contains(summary.Warnings[0].Message, "shared") {
		t.Fatalf("summary = %+v, want shared environment warning", summary)
	}
	if _, err := os.Stat(filepath.Join(root, "environments", "shared-dev.yaml")); err != nil {
		t.Fatalf("shared environment missing: %v", err)
	}
}
