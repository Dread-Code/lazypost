package importer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/Dread-Code/lazypost/internal/collection"
)

type insomniaExport struct {
	Type         string             `json:"_type"`
	ExportFormat int                `json:"__export_format"`
	Resources    []insomniaResource `json:"resources"`
}

type insomniaResource struct {
	Type        string         `json:"_type"`
	ID          string         `json:"_id"`
	ParentID    string         `json:"parentId"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Method      string         `json:"method"`
	URL         string         `json:"url"`
	Headers     []insomniaKV   `json:"headers"`
	Parameters  []insomniaKV   `json:"parameters"`
	Body        *insomniaBody  `json:"body"`
	Auth        *insomniaAuth  `json:"authentication"`
	Data        map[string]any `json:"data"`
	PreScript   string         `json:"preRequestScript"`
	PostScript  string         `json:"afterResponseScript"`
}

type insomniaKV struct {
	Name        string `json:"name" yaml:"name"`
	Key         string `json:"key" yaml:"key"`
	Value       any    `json:"value" yaml:"value"`
	Disabled    bool   `json:"disabled" yaml:"disabled"`
	Description string `json:"description" yaml:"description"`
}

type insomniaBody struct {
	MimeType string `json:"mimeType" yaml:"mimeType"`
	Text     string `json:"text" yaml:"text"`
}

type insomniaAuth struct {
	Type     string `json:"type" yaml:"type"`
	Username string `json:"username" yaml:"username"`
	Password string `json:"password" yaml:"password"`
	Token    string `json:"token" yaml:"token"`
	Key      string `json:"key" yaml:"key"`
	Value    string `json:"value" yaml:"value"`
	AddTo    string `json:"addTo" yaml:"addTo"`
}

func parseInsomnia(data []byte, source string, environmentFiles []string) (Result, error) {
	trimmed := strings.TrimSpace(string(data))
	if strings.HasPrefix(trimmed, "{") {
		var doc insomniaExport
		if err := json.Unmarshal(data, &doc); err != nil {
			return Result{}, fmt.Errorf("parsing Insomnia export: %w", err)
		}
		if doc.Type != "export" || doc.ExportFormat != 4 {
			return Result{}, fmt.Errorf("unsupported Insomnia JSON export; expected _type export and __export_format 4")
		}
		return parseInsomniaV4(doc, source, environmentFiles)
	}
	return parseInsomniaV5(data, source, environmentFiles)
}

// parseInsomniaDirectory handles Insomnia's export-folder form: one YAML
// document per workspace/resource. Supported collection workspaces become
// top-level folders; unrelated resource files are warnings, not fatal input
// errors.
func parseInsomniaDirectory(dir string, environmentFiles []string) (Result, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return Result{}, err
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	var collections []Result
	var pendingEnvironments []ImportedEnvironment
	var warnings []Warning
	for _, entry := range entries {
		if entry.IsDir() || (filepath.Ext(entry.Name()) != ".yaml" && filepath.Ext(entry.Name()) != ".yml" && filepath.Ext(entry.Name()) != ".json") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		data, err := readImportFile(path)
		if err != nil {
			return Result{}, err
		}
		var shape struct {
			Type string `yaml:"type"`
		}
		if err := yaml.Unmarshal(data, &shape); err != nil {
			warnings = append(warnings, Warning{Path: entry.Name(), Message: "ignored unreadable Insomnia resource: " + err.Error()})
			continue
		}
		switch {
		case strings.HasPrefix(shape.Type, "collection.insomnia.rest/"):
			result, err := parseInsomniaV5(data, entry.Name(), nil)
			if err != nil {
				return Result{}, err
			}
			collections = append(collections, result)
		case strings.HasPrefix(shape.Type, "environment.insomnia.rest/"):
			env, err := readEnvironmentFile(path)
			if err != nil {
				return Result{}, err
			}
			pendingEnvironments = append(pendingEnvironments, env)
		case strings.HasPrefix(shape.Type, "spec.insomnia.rest/"):
			warnings = append(warnings, Warning{Path: entry.Name(), Message: "OpenAPI-backed Insomnia document skipped; use a separate OpenAPI import"})
		case strings.HasPrefix(shape.Type, "mock.insomnia.rest/"):
			warnings = append(warnings, Warning{Path: entry.Name(), Message: "Insomnia mock-server resource skipped"})
		default:
			warnings = append(warnings, Warning{Path: entry.Name(), Message: "unrecognized Insomnia resource skipped"})
		}
	}
	if len(collections) == 0 {
		return Result{}, fmt.Errorf("directory contains no supported Insomnia collection files")
	}
	result := Result{Name: filepath.Base(dir), Warnings: warnings}
	for _, source := range collections {
		result.Workspaces = append(result.Workspaces, source.Workspaces...)
		result.Warnings = append(result.Warnings, source.Warnings...)
	}
	if len(result.Workspaces) == 0 {
		return Result{}, fmt.Errorf("directory contains no supported Insomnia workspaces")
	}
	result.Environments = append(result.Environments, pendingEnvironments...)
	for _, path := range environmentFiles {
		env, err := readEnvironmentFile(path)
		if err != nil {
			return Result{}, err
		}
		result.Environments = append(result.Environments, env)
	}
	return result, nil
}

func parseInsomniaV4(doc insomniaExport, source string, environmentFiles []string) (Result, error) {
	var result Result
	children := make(map[string][]insomniaResource)
	var workspaces []insomniaResource
	for _, resource := range doc.Resources {
		children[resource.ParentID] = append(children[resource.ParentID], resource)
		if resource.Type == "workspace" {
			workspaces = append(workspaces, resource)
		}
	}
	if len(workspaces) == 0 {
		return Result{}, fmt.Errorf("%s: Insomnia export contains no workspace", source)
	}
	result.Name = workspaces[0].Name
	for _, workspace := range workspaces {
		name := workspace.Name
		if name == "" {
			name = "workspace"
		}
		ws := Workspace{Name: name}
		var walk func(string, []string, map[string]bool) error
		walk = func(parent string, path []string, ancestors map[string]bool) error {
			for _, resource := range children[parent] {
				if resource.ID != "" && ancestors[resource.ID] {
					return fmt.Errorf("%s: cyclic resource graph at %q", source, resource.ID)
				}
				switch resource.Type {
				case "request_group":
					name := resource.Name
					if name == "" {
						name = "untitled"
					}
					folder := append(append([]string{}, path...), name)
					ws.Folders = append(ws.Folders, folder)
					nextAncestors := make(map[string]bool, len(ancestors)+1)
					for id := range ancestors {
						nextAncestors[id] = true
					}
					if resource.ID != "" {
						nextAncestors[resource.ID] = true
					}
					if err := walk(resource.ID, folder, nextAncestors); err != nil {
						return err
					}
				case "request":
					request := mapInsomniaRequest(resource, path, &ws)
					ws.Requests = append(ws.Requests, ImportedRequest{Path: path, Request: request})
				}
			}
			return nil
		}
		ancestors := map[string]bool{}
		if workspace.ID != "" {
			ancestors[workspace.ID] = true
		}
		if err := walk(workspace.ID, nil, ancestors); err != nil {
			return Result{}, err
		}
		result.Workspaces = append(result.Workspaces, ws)
		result.Warnings = append(result.Warnings, ws.Warnings...)
	}
	for _, resource := range doc.Resources {
		if resource.Type != "environment" {
			continue
		}
		for i := range result.Workspaces {
			if resource.ParentID == workspaces[i].ID {
				addEnvironment(&result.Workspaces[i], ImportedEnvironment{Name: resource.Name, Variables: stringMap(resource.Data)})
				break
			}
		}
	}
	for _, path := range environmentFiles {
		env, err := readEnvironmentFile(path)
		if err != nil {
			return Result{}, err
		}
		result.Environments = append(result.Environments, env)
	}
	return result, nil
}

func mapInsomniaRequest(resource insomniaResource, path []string, workspace *Workspace) collection.Request {
	name := resource.Name
	if name == "" {
		name = "untitled"
	}
	req := collection.Request{Name: name, Method: resource.Method, URL: normalizeVariables(resource.URL)}
	if req.Method == "" {
		req.Method = "GET"
	}
	for _, header := range resource.Headers {
		if header.Disabled {
			workspace.Warnings = append(workspace.Warnings, Warning{Path: strings.Join(append(path, name), "/"), Message: "disabled header skipped: " + kvName(header)})
			continue
		}
		req.Headers = append(req.Headers, collection.Header{Name: kvName(header), Value: normalizeVariables(stringValue(header.Value))})
	}
	explicitQuery := make([]collection.Param, 0, len(resource.Parameters))
	explicitQueryKeys := make(map[string]struct{}, len(resource.Parameters))
	for _, parameter := range resource.Parameters {
		paramName := kvName(parameter)
		explicitQueryKeys[paramName] = struct{}{}
		if parameter.Disabled {
			workspace.Warnings = append(workspace.Warnings, Warning{Path: strings.Join(append(path, name), "/"), Message: "disabled query parameter skipped: " + paramName})
			continue
		}
		explicitQuery = append(explicitQuery, collection.Param{Name: paramName, Value: normalizeVariables(stringValue(parameter.Value))})
	}
	req.URL, req.Query = normalizeURLQuery(req.URL, explicitQuery, explicitQueryKeys)
	if resource.Body != nil {
		switch resource.Body.MimeType {
		case "", "application/json", "text/plain", "application/xml", "text/xml", "text/html":
			req.Body = resource.Body.Text
		default:
			workspace.Warnings = append(workspace.Warnings, Warning{Path: strings.Join(append(path, name), "/"), Message: "body MIME type was not imported: " + resource.Body.MimeType})
		}
	}
	if resource.Auth != nil {
		req.Auth = mapInsomniaAuth(resource.Auth, workspace, path, name)
	}
	if resource.PreScript != "" {
		workspace.Warnings = append(workspace.Warnings, Warning{Path: strings.Join(append(path, name), "/"), Message: "JavaScript pre-request script was not imported"})
	}
	if resource.PostScript != "" {
		workspace.Warnings = append(workspace.Warnings, Warning{Path: strings.Join(append(path, name), "/"), Message: "JavaScript after-response script was not imported"})
	}
	return req
}

func mapInsomniaAuth(auth *insomniaAuth, workspace *Workspace, path []string, name string) *collection.Auth {
	switch auth.Type {
	case "basic":
		return &collection.Auth{Type: "basic", Username: normalizeVariables(auth.Username), Password: normalizeVariables(auth.Password)}
	case "bearer":
		return &collection.Auth{Type: "bearer", Token: normalizeVariables(auth.Token)}
	case "apikey":
		in := auth.AddTo
		if in == "" {
			in = "header"
		}
		return &collection.Auth{Type: "apikey", KeyName: auth.Key, KeyValue: normalizeVariables(auth.Value), KeyIn: in}
	case "", "none":
		return nil
	default:
		workspace.Warnings = append(workspace.Warnings, Warning{Path: strings.Join(append(path, name), "/"), Message: "unsupported auth scheme omitted: " + auth.Type})
		return nil
	}
}

func kvName(value insomniaKV) string {
	if value.Name != "" {
		return value.Name
	}
	return value.Key
}

func stringMap(values map[string]any) map[string]string {
	result := map[string]string{}
	for key, value := range values {
		result[key] = normalizeVariables(stringValue(value))
	}
	return result
}

type insomniaV5Document struct {
	Type          string           `yaml:"type"`
	SchemaVersion string           `yaml:"schema_version"`
	Name          string           `yaml:"name"`
	Collection    []insomniaV5Node `yaml:"collection"`
	Environments  *insomniaV5Env   `yaml:"environments"`
}

type insomniaV5Node struct {
	Name       string            `yaml:"name"`
	URL        string            `yaml:"url"`
	Method     string            `yaml:"method"`
	Children   []insomniaV5Node  `yaml:"children"`
	Body       *insomniaBody     `yaml:"body"`
	Headers    []insomniaKV      `yaml:"headers"`
	Parameters []insomniaKV      `yaml:"parameters"`
	Scripts    map[string]string `yaml:"scripts"`
	Auth       *insomniaAuth     `yaml:"authentication"`
}

type insomniaV5Env struct {
	Name string         `yaml:"name"`
	Data map[string]any `yaml:"data"`
}

func parseInsomniaV5(data []byte, source string, environmentFiles []string) (Result, error) {
	var doc insomniaV5Document
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return Result{}, fmt.Errorf("parsing Insomnia v5 YAML: %w", err)
	}
	if strings.HasPrefix(doc.Type, "spec.insomnia.rest/") {
		return Result{}, fmt.Errorf("%s: OpenAPI-backed Insomnia documents are not supported yet", source)
	}
	if !strings.HasPrefix(doc.Type, "collection.insomnia.rest/") {
		return Result{}, fmt.Errorf("%s: unsupported Insomnia YAML type %q", source, doc.Type)
	}
	workspaceName := doc.Name
	if workspaceName == "" {
		workspaceName = "workspace"
	}
	result := Result{Name: doc.Name, Workspaces: []Workspace{{Name: workspaceName}}}
	workspace := &result.Workspaces[0]
	var walk func([]insomniaV5Node, []string)
	walk = func(nodes []insomniaV5Node, path []string) {
		for _, node := range nodes {
			name := node.Name
			if name == "" {
				name = "untitled"
			}
			if len(node.Children) > 0 {
				folder := append(append([]string{}, path...), name)
				workspace.Folders = append(workspace.Folders, folder)
				walk(node.Children, folder)
				continue
			}
			request := mapInsomniaV5Request(node, path, workspace)
			workspace.Requests = append(workspace.Requests, ImportedRequest{Path: path, Request: request})
		}
	}
	walk(doc.Collection, nil)
	if doc.Environments != nil && len(doc.Environments.Data) > 0 {
		addEnvironment(workspace, ImportedEnvironment{Name: doc.Environments.Name, Variables: stringMap(doc.Environments.Data)})
	}
	for _, path := range environmentFiles {
		env, err := readEnvironmentFile(path)
		if err != nil {
			return Result{}, err
		}
		addEnvironment(workspace, env)
	}
	result.Warnings = append(result.Warnings, workspace.Warnings...)
	return result, nil
}

func mapInsomniaV5Request(node insomniaV5Node, path []string, workspace *Workspace) collection.Request {
	request := mapInsomniaRequest(insomniaResource{
		Name:       node.Name,
		URL:        node.URL,
		Method:     node.Method,
		Headers:    node.Headers,
		Parameters: node.Parameters,
		Body:       node.Body,
		Auth:       node.Auth,
	}, path, workspace)
	for scriptName, script := range node.Scripts {
		if script != "" {
			workspace.Warnings = append(workspace.Warnings, Warning{Path: strings.Join(append(path, request.Name), "/"), Message: "JavaScript " + scriptName + " script was not imported"})
		}
	}
	return request
}
