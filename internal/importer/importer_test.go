package importer

import (
	"path/filepath"
	"strings"
	"testing"
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
	if workspace.Requests[0].Path[0] != "Users" || req.Method != "GET" || req.URL != "{{base_url}}/users?limit=10" {
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

func TestDetectRejectsOpenAPIInsomniaDocument(t *testing.T) {
	_, err := Detect([]byte("type: spec.insomnia.rest/5.0\nschema_version: '5.1'\n"), "")
	if err == nil || !strings.Contains(err.Error(), "OpenAPI-backed") {
		t.Fatalf("Detect error = %v, want deferred OpenAPI error", err)
	}
}
