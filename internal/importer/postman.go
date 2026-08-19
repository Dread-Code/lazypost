package importer

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"lazypost/internal/collection"
)

type postmanCollection struct {
	Info struct {
		Name   string `json:"name"`
		Schema string `json:"schema"`
	} `json:"info"`
	Item     []postmanItem     `json:"item"`
	Variable []postmanVariable `json:"variable"`
	Event    []postmanEvent    `json:"event"`
	Auth     *postmanAuth      `json:"auth"`
}

type postmanItem struct {
	Name    string          `json:"name"`
	Item    []postmanItem   `json:"item"`
	Request json.RawMessage `json:"request"`
	Event   []postmanEvent  `json:"event"`
	Auth    *postmanAuth    `json:"auth"`
}

type postmanVariable struct {
	Key      string `json:"key"`
	Value    any    `json:"value"`
	Disabled bool   `json:"disabled"`
}

type postmanEvent struct {
	Listen string `json:"listen"`
	Script struct {
		Exec any `json:"exec"`
	} `json:"script"`
}

type postmanRequest struct {
	URL    json.RawMessage `json:"url"`
	Method string          `json:"method"`
	Header []postmanKV     `json:"header"`
	Body   *postmanBody    `json:"body"`
	Auth   *postmanAuth    `json:"auth"`
}

type postmanKV struct {
	Key         string `json:"key"`
	Value       any    `json:"value"`
	Description string `json:"description"`
	Disabled    bool   `json:"disabled"`
	Type        string `json:"type"`
}

type postmanBody struct {
	Mode       string         `json:"mode"`
	Raw        string         `json:"raw"`
	URLEncoded []postmanKV    `json:"urlencoded"`
	FormData   []postmanKV    `json:"formdata"`
	GraphQL    map[string]any `json:"graphql"`
}

type postmanAuth struct {
	Type   string      `json:"type"`
	Basic  []postmanKV `json:"basic"`
	Bearer []postmanKV `json:"bearer"`
	APIKey []postmanKV `json:"apikey"`
}

type postmanURL struct {
	Raw      string      `json:"raw"`
	Protocol string      `json:"protocol"`
	Host     []string    `json:"host"`
	Path     []string    `json:"path"`
	Query    []postmanKV `json:"query"`
}

func parsePostman(data []byte, source string, environmentFiles []string) (Result, error) {
	var doc postmanCollection
	if err := json.Unmarshal(data, &doc); err != nil {
		return Result{}, fmt.Errorf("parsing Postman collection: %w", err)
	}
	if doc.Info.Name == "" {
		return Result{}, fmt.Errorf("Postman collection has no info.name")
	}
	if !strings.Contains(doc.Info.Schema, "collection/v2.1") {
		return Result{}, fmt.Errorf("unsupported Postman collection schema %q; only v2.1 is supported", doc.Info.Schema)
	}
	workspace := Workspace{Name: doc.Info.Name}
	result := Result{Name: doc.Info.Name, Workspaces: []Workspace{workspace}}
	for _, event := range doc.Event {
		result.Warnings = append(result.Warnings, Warning{Path: "collection", Message: "JavaScript " + event.Listen + " script was not imported"})
	}
	if vars := postmanVariables(doc.Variable); len(vars) > 0 {
		addEnvironment(&result.Workspaces[0], ImportedEnvironment{Name: "base", Variables: vars})
	}
	if err := walkPostmanItems(&result.Workspaces[0], doc.Item, nil, doc.Auth); err != nil {
		return Result{}, fmt.Errorf("%s: %w", source, err)
	}
	result.Warnings = append(result.Warnings, result.Workspaces[0].Warnings...)
	for _, path := range environmentFiles {
		env, err := readEnvironmentFile(path)
		if err != nil {
			return Result{}, err
		}
		addEnvironment(&result.Workspaces[0], env)
	}
	return result, nil
}

func postmanVariables(values []postmanVariable) map[string]string {
	vars := map[string]string{}
	for _, item := range values {
		if item.Disabled || item.Key == "" {
			continue
		}
		vars[item.Key] = normalizeVariables(stringValue(item.Value))
	}
	return vars
}

func walkPostmanItems(workspace *Workspace, items []postmanItem, parent []string, inheritedAuth *postmanAuth) error {
	for _, item := range items {
		name := item.Name
		if name == "" {
			name = "untitled"
		}
		if len(item.Item) > 0 {
			folder := append(append([]string{}, parent...), name)
			workspace.Folders = append(workspace.Folders, folder)
			folderAuth := inheritedAuth
			if item.Auth != nil {
				folderAuth = item.Auth
			}
			if err := walkPostmanItems(workspace, item.Item, folder, folderAuth); err != nil {
				return err
			}
			continue
		}
		if len(item.Request) == 0 || string(item.Request) == "null" {
			continue
		}
		var raw postmanRequest
		if err := json.Unmarshal(item.Request, &raw); err != nil {
			return fmt.Errorf("request %q: %w", name, err)
		}
		if raw.Auth == nil {
			raw.Auth = inheritedAuth
		}
		req, err := mapPostmanRequest(raw, name, append([]string{}, parent...), workspace)
		if err != nil {
			return err
		}
		for _, event := range item.Event {
			workspace.Warnings = append(workspace.Warnings, Warning{Path: strings.Join(append(parent, name), "/"), Message: "JavaScript " + event.Listen + " script was not imported"})
		}
		workspace.Requests = append(workspace.Requests, ImportedRequest{Path: parent, Request: req})
	}
	return nil
}

func mapPostmanRequest(raw postmanRequest, name string, path []string, workspace *Workspace) (collection.Request, error) {
	req := collection.Request{Name: name, Method: raw.Method, URL: postmanURLValue(raw.URL)}
	if req.Method == "" {
		req.Method = "GET"
	}
	for _, header := range raw.Header {
		if header.Disabled {
			workspace.Warnings = append(workspace.Warnings, Warning{Path: strings.Join(append(path, name), "/"), Message: "disabled header skipped: " + header.Key})
			continue
		}
		req.Headers = append(req.Headers, collection.Header{Name: header.Key, Value: normalizeVariables(stringValue(header.Value))})
	}
	if u := postmanURLObject(raw.URL); u != nil {
		for _, query := range u.Query {
			if query.Disabled {
				workspace.Warnings = append(workspace.Warnings, Warning{Path: strings.Join(append(path, name), "/"), Message: "disabled query parameter skipped: " + query.Key})
				continue
			}
			req.Query = append(req.Query, collection.Param{Name: query.Key, Value: normalizeVariables(stringValue(query.Value))})
		}
	}
	if raw.Body != nil {
		switch raw.Body.Mode {
		case "raw":
			req.Body = raw.Body.Raw
		case "urlencoded":
			values := url.Values{}
			for _, item := range raw.Body.URLEncoded {
				if !item.Disabled {
					values.Set(item.Key, stringValue(item.Value))
				}
			}
			req.Body = values.Encode()
		case "formdata":
			workspace.Warnings = append(workspace.Warnings, Warning{Path: strings.Join(append(path, name), "/"), Message: "multipart form-data body was not imported"})
		case "graphql":
			workspace.Warnings = append(workspace.Warnings, Warning{Path: strings.Join(append(path, name), "/"), Message: "GraphQL body was not imported"})
		case "file", "binary":
			workspace.Warnings = append(workspace.Warnings, Warning{Path: strings.Join(append(path, name), "/"), Message: raw.Body.Mode + " body was not imported"})
		}
	}
	if raw.Auth != nil {
		req.Auth = mapPostmanAuth(raw.Auth, workspace, path, name)
	}
	return req, nil
}

func mapPostmanAuth(auth *postmanAuth, workspace *Workspace, path []string, name string) *collection.Auth {
	switch auth.Type {
	case "basic":
		return &collection.Auth{Type: "basic", Username: authValue(auth.Basic, "username"), Password: authValue(auth.Basic, "password")}
	case "bearer":
		return &collection.Auth{Type: "bearer", Token: authValue(auth.Bearer, "token")}
	case "apikey":
		in := authValue(auth.APIKey, "in")
		if in == "" {
			in = "header"
		}
		return &collection.Auth{Type: "apikey", KeyName: authValue(auth.APIKey, "key"), KeyValue: authValue(auth.APIKey, "value"), KeyIn: in}
	case "", "noauth":
		return nil
	default:
		workspace.Warnings = append(workspace.Warnings, Warning{Path: strings.Join(append(path, name), "/"), Message: "unsupported auth scheme omitted: " + auth.Type})
		return nil
	}
}

func authValue(values []postmanKV, key string) string {
	for _, item := range values {
		if item.Key == key {
			return normalizeVariables(stringValue(item.Value))
		}
	}
	return ""
}

func postmanURLObject(raw json.RawMessage) *postmanURL {
	if len(raw) == 0 || raw[0] == '"' {
		return nil
	}
	var value postmanURL
	if json.Unmarshal(raw, &value) != nil {
		return nil
	}
	return &value
}

func postmanURLValue(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var stringURL string
	if json.Unmarshal(raw, &stringURL) == nil {
		return normalizeVariables(stringURL)
	}
	u := postmanURLObject(raw)
	if u == nil {
		return ""
	}
	if u.Raw != "" {
		return normalizeVariables(u.Raw)
	}
	var builder strings.Builder
	if u.Protocol != "" {
		builder.WriteString(u.Protocol)
		builder.WriteString("://")
	}
	builder.WriteString(strings.Join(u.Host, "."))
	if len(u.Path) > 0 {
		builder.WriteString("/")
		builder.WriteString(strings.Join(u.Path, "/"))
	}
	values := url.Values{}
	for _, query := range u.Query {
		if !query.Disabled {
			values.Set(query.Key, stringValue(query.Value))
		}
	}
	if encoded := values.Encode(); encoded != "" {
		builder.WriteByte('?')
		builder.WriteString(encoded)
	}
	return normalizeVariables(builder.String())
}
