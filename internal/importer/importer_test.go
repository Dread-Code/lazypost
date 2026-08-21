package importer

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Dread-Code/lazypost/internal/collection"
	"github.com/Dread-Code/lazypost/internal/httpclient"
)

func TestPostmanCollectionAndEnvironment(t *testing.T) {
	result, err := ParseFile(filepath.Join("testdata", "postman", "collection.json"), ParseOptions{
		EnvironmentFiles: []string{filepath.Join("testdata", "postman", "environment.json")},
	})
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	if result.Name != "Postman Demo" || len(result.Workspaces) != 1 {
		t.Fatalf("result = %+v, want one workspace", result)
	}
	workspace := result.Workspaces[0]
	if len(workspace.Requests) != 1 || len(workspace.Folders) != 1 {
		t.Fatalf("workspace = %+v, want one folder and request", workspace)
	}
	req := workspace.Requests[0].Request
	if workspace.Requests[0].Path[0] != "Users" || req.Method != "GET" || req.URL != "{{base_url}}/users" {
		t.Errorf("request = %+v", req)
	}
	if len(req.Headers) != 1 || req.Headers[0].Name != "Accept" || len(req.Query) != 1 || req.Auth == nil || req.Auth.Token != "{{token}}" {
		t.Errorf("mapped request fields = %+v", req)
	}
	if len(workspace.Environments) != 2 || workspace.Environments[0].Name != "base" || workspace.Environments[1].Name != "dev" {
		t.Errorf("environments = %+v", workspace.Environments)
	}
	if workspace.Environments[1].Variables["disabled"] != "" || workspace.Environments[1].Variables["token"] != "secret" {
		t.Errorf("environment variables = %+v", workspace.Environments[1].Variables)
	}
	if len(result.Warnings) < 3 {
		t.Errorf("warnings = %+v, want disabled fields and script warnings", result.Warnings)
	}
}

func TestInsomniaV4(t *testing.T) {
	result, err := ParseFile(filepath.Join("testdata", "insomnia", "v4.json"), ParseOptions{})
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	if result.Name != "Insomnia v4 Demo" || len(result.Workspaces) != 2 || len(result.Workspaces[0].Requests) != 1 || len(result.Workspaces[0].Environments) != 1 || len(result.Workspaces[1].Requests) != 1 || len(result.Workspaces[1].Environments) != 1 {
		t.Fatalf("result = %+v", result)
	}
	if got := result.Workspaces[0].Requests[0].Request.URL; got != "{{base_url}}/users" {
		t.Errorf("URL = %q, want normalized Insomnia variable", got)
	}
	if result.Workspaces[0].Requests[0].Path[0] != "Users" || result.Workspaces[0].Environments[0].Variables["base_url"] != "https://api.example.test" {
		t.Errorf("result = %+v", result)
	}
	if result.Workspaces[1].Name != "Admin" || len(result.Workspaces[1].Requests[0].Path) != 0 {
		t.Errorf("second workspace = %+v", result.Workspaces[1])
	}
}

func TestInsomniaV5(t *testing.T) {
	result, err := ParseFile(filepath.Join("testdata", "insomnia", "v5.yaml"), ParseOptions{})
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	if result.Name != "Insomnia v5 Demo" || len(result.Workspaces) != 1 || len(result.Workspaces[0].Requests) != 1 || len(result.Workspaces[0].Environments) != 1 {
		t.Fatalf("result = %+v", result)
	}
	if result.Workspaces[0].Requests[0].Request.URL != "{{base_url}}/users" || result.Workspaces[0].Environments[0].Name != "Base" {
		t.Errorf("result = %+v", result)
	}
	var scriptWarning bool
	for _, warning := range result.Warnings {
		if strings.Contains(warning.Message, "afterResponse") {
			scriptWarning = true
		}
	}
	if !scriptWarning {
		t.Errorf("warnings = %+v, want script warning", result.Warnings)
	}
}

func TestInsomniaDirectoryCombinesWorkspacesAndSkipsMocks(t *testing.T) {
	result, err := ParseFile(filepath.Join("testdata", "insomnia", "directory"), ParseOptions{})
	if err != nil {
		t.Fatalf("ParseFile directory: %v", err)
	}
	if len(result.Workspaces) != 2 || len(result.Workspaces[0].Requests) != 1 || len(result.Workspaces[1].Requests) != 1 {
		t.Fatalf("result = %+v, want two workspaces and requests", result)
	}
	if result.Workspaces[0].Name != "Alpha" || result.Workspaces[1].Name != "Beta" {
		t.Errorf("workspace names = %+v", result.Workspaces)
	}
	var mockWarning bool
	for _, warning := range result.Warnings {
		if strings.Contains(warning.Message, "mock-server") {
			mockWarning = true
		}
	}
	if !mockWarning {
		t.Errorf("warnings = %+v, want skipped mock warning", result.Warnings)
	}
}

func TestInsomniaDirectoryKeepsUnscopedEnvironmentsSeparate(t *testing.T) {
	dir := t.TempDir()
	for name, workspace := range map[string]string{
		"alpha.yaml": "type: collection.insomnia.rest/5.0\nschema_version: '5.1'\nname: Alpha\n",
		"beta.yaml":  "type: collection.insomnia.rest/5.0\nschema_version: '5.1'\nname: Beta\n",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(workspace), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	env := "type: environment.insomnia.rest/5.0\nschema_version: '5.1'\nname: Shared\ndata:\n  host: https://shared.test\n"
	if err := os.WriteFile(filepath.Join(dir, "shared.yaml"), []byte(env), 0o600); err != nil {
		t.Fatal(err)
	}

	result, err := ParseFile(dir, ParseOptions{})
	if err != nil {
		t.Fatalf("ParseFile directory: %v", err)
	}
	if len(result.Workspaces) != 2 || len(result.Environments) != 1 || result.Environments[0].Name != "Shared" {
		t.Fatalf("workspaces=%+v global environments=%+v", result.Workspaces, result.Environments)
	}
}

func TestDetectRejectsOpenAPIInsomniaDocument(t *testing.T) {
	_, err := Detect([]byte("type: spec.insomnia.rest/5.0\nschema_version: '5.1'\n"), "")
	if err == nil || !strings.Contains(err.Error(), "OpenAPI-backed") {
		t.Fatalf("Detect error = %v, want deferred OpenAPI error", err)
	}
}

func TestImportRejectsCyclicInsomniaResources(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cycle.json")
	data := `{"_type":"export","__export_format":4,"resources":[{"_type":"workspace","_id":"w","name":"Demo"},{"_type":"request_group","_id":"g","parentId":"w","name":"A"},{"_type":"request_group","_id":"g","parentId":"g","name":"B"}]}`
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ParseFile(path, ParseOptions{}); err == nil || !strings.Contains(err.Error(), "cyclic resource graph") {
		t.Fatalf("ParseFile cycle error = %v, want cyclic resource graph", err)
	}
}

func TestImportRejectsOversizedFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "large.json")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.Truncate(path, maxImportFileSize+1); err != nil {
		t.Fatal(err)
	}
	if _, err := ParseFile(path, ParseOptions{}); err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("ParseFile oversized error = %v, want size limit error", err)
	}
}

func TestImportedQueriesExecuteExactlyOnce(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query()["limit"]; len(got) != 1 || got[0] != "10" {
			t.Errorf("limit query = %v, want exactly one value", got)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	for _, source := range []string{
		filepath.Join("testdata", "postman", "collection.json"),
		filepath.Join("testdata", "insomnia", "v4.json"),
	} {
		result, err := ParseFile(source, ParseOptions{})
		if err != nil {
			t.Fatalf("ParseFile(%s): %v", source, err)
		}
		req := result.Workspaces[0].Requests[0].Request
		req.URL = srv.URL + "/users"
		if _, err := httpclient.Exec(req, nil); err != nil {
			t.Fatalf("Exec(%s): %v", source, err)
		}
	}
}

func TestNormalizeURLQueryPreservesFragmentAndRepeatedValues(t *testing.T) {
	base, params := normalizeURLQuery(
		"https://api.test/items?keep=yes&limit=10#fragment",
		[]collection.Param{
			{Name: "limit", Value: "10"},
			{Name: "tag", Value: "one"},
			{Name: "tag", Value: "two"},
		},
		map[string]struct{}{"limit": {}},
	)
	if base != "https://api.test/items#fragment" {
		t.Errorf("base URL = %q", base)
	}
	if len(params) != 4 || params[0].Name != "keep" || params[1].Name != "limit" || params[2].Value != "one" || params[3].Value != "two" {
		t.Errorf("normalized params = %+v", params)
	}
}
