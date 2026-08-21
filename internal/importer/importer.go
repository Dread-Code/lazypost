// Package importer turns Postman and Insomnia exports into lazypost's
// collection model. It only parses and normalizes data; filesystem writes
// belong to internal/app.
package importer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"

	"lazypost/internal/collection"
)

type Format string

const (
	FormatPostman  Format = "postman"
	FormatInsomnia Format = "insomnia"
)

// ParseOptions controls format detection and optional environment files.
type ParseOptions struct {
	Format           string
	EnvironmentFiles []string
}

const maxImportFileSize = 32 << 20 // 32 MiB per source/resource file

type Result struct {
	Name       string
	Workspaces []Workspace
	// Environments are source-level environments without an unambiguous
	// workspace owner, such as Insomnia directory exports.
	Environments []ImportedEnvironment
	Warnings     []Warning
}

// Workspace is the source-level boundary around folders, requests, and
// environments. Postman collections become one synthetic workspace named
// after info.name; Insomnia workspaces retain their native names.
type Workspace struct {
	Name         string
	Folders      [][]string
	Requests     []ImportedRequest
	Environments []ImportedEnvironment
	Warnings     []Warning
}

type ImportedRequest struct {
	Path    []string
	Request collection.Request
}

type ImportedEnvironment struct {
	Name      string
	Variables map[string]string
}

type Warning struct {
	Path    string
	Message string
}

func (w Warning) String() string {
	if w.Path == "" {
		return w.Message
	}
	return w.Path + ": " + w.Message
}

// ParseFile detects or uses the requested format and returns a complete
// in-memory result. No output files are touched here.
func ParseFile(path string, opts ParseOptions) (Result, error) {
	info, err := os.Stat(path)
	if err != nil {
		return Result{}, err
	}
	if info.IsDir() {
		if opts.Format != "" && Format(strings.ToLower(opts.Format)) != FormatInsomnia {
			return Result{}, fmt.Errorf("directory imports only support Insomnia format")
		}
		return parseInsomniaDirectory(path, opts.EnvironmentFiles)
	}
	data, err := readImportFile(path)
	if err != nil {
		return Result{}, err
	}
	format, err := Detect(data, opts.Format)
	if err != nil {
		return Result{}, fmt.Errorf("%s: %w", path, err)
	}
	switch format {
	case FormatPostman:
		return parsePostman(data, path, opts.EnvironmentFiles)
	case FormatInsomnia:
		return parseInsomnia(data, path, opts.EnvironmentFiles)
	default:
		return Result{}, fmt.Errorf("unsupported import format %q", format)
	}
}

func readImportFile(path string) ([]byte, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if info.Size() > maxImportFileSize {
		return nil, fmt.Errorf("import file %s exceeds the %d MiB limit", path, maxImportFileSize>>20)
	}
	return os.ReadFile(path)
}

func Detect(data []byte, requested string) (Format, error) {
	if requested != "" {
		switch Format(strings.ToLower(requested)) {
		case FormatPostman, FormatInsomnia:
			return Format(strings.ToLower(requested)), nil
		default:
			return "", fmt.Errorf("unknown format %q", requested)
		}
	}

	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return "", fmt.Errorf("empty import file")
	}
	var jsonShape map[string]any
	if json.Unmarshal(trimmed, &jsonShape) == nil {
		if _, ok := jsonShape["info"]; ok {
			return FormatPostman, nil
		}
		if _, ok := jsonShape["resources"]; ok {
			return FormatInsomnia, nil
		}
	}

	var yamlShape struct {
		Type string `yaml:"type"`
	}
	if yaml.Unmarshal(trimmed, &yamlShape) == nil {
		if strings.HasPrefix(yamlShape.Type, "collection.insomnia.rest/") ||
			strings.HasPrefix(yamlShape.Type, "environment.insomnia.rest/") {
			return FormatInsomnia, nil
		}
		if strings.HasPrefix(yamlShape.Type, "spec.insomnia.rest/") {
			return "", fmt.Errorf("OpenAPI-backed Insomnia documents are not supported yet")
		}
	}
	return "", fmt.Errorf("could not detect Postman or Insomnia format")
}

var insomniaVarRE = regexp.MustCompile(`\{\{\s*_\.\s*([A-Za-z0-9_.-]+)\s*\}\}`)

func normalizeVariables(value string) string {
	return insomniaVarRE.ReplaceAllString(value, "{{$1}}")
}

func addEnvironment(workspace *Workspace, env ImportedEnvironment) {
	if env.Name == "" {
		env.Name = "base"
	}
	if env.Variables == nil {
		env.Variables = map[string]string{}
	}
	base := env.Name
	for i := 2; ; i++ {
		duplicate := false
		for _, existing := range workspace.Environments {
			if existing.Name == env.Name {
				duplicate = true
				break
			}
		}
		if !duplicate {
			break
		}
		env.Name = fmt.Sprintf("%s-%d", base, i)
	}
	workspace.Environments = append(workspace.Environments, env)
}

func stringValue(value any) string {
	if value == nil {
		return ""
	}
	if s, ok := value.(string); ok {
		return s
	}
	if number, ok := value.(json.Number); ok {
		return number.String()
	}
	return fmt.Sprint(value)
}

func readEnvironmentFile(path string) (ImportedEnvironment, error) {
	data, err := readImportFile(path)
	if err != nil {
		return ImportedEnvironment{}, err
	}
	var jsonEnv struct {
		Name   string `json:"name" yaml:"name"`
		Values []struct {
			Key     string `json:"key" yaml:"key"`
			Value   any    `json:"value" yaml:"value"`
			Enabled *bool  `json:"enabled" yaml:"enabled"`
		} `json:"values" yaml:"values"`
		Data map[string]any `json:"data" yaml:"data"`
	}
	if err := json.Unmarshal(data, &jsonEnv); err != nil {
		if err := yaml.Unmarshal(data, &jsonEnv); err != nil {
			return ImportedEnvironment{}, fmt.Errorf("parsing environment %s: %w", path, err)
		}
	}
	env := ImportedEnvironment{Name: jsonEnv.Name, Variables: map[string]string{}}
	for _, item := range jsonEnv.Values {
		if item.Enabled != nil && !*item.Enabled {
			continue
		}
		if item.Key != "" {
			env.Variables[item.Key] = normalizeVariables(stringValue(item.Value))
		}
	}
	for key, value := range jsonEnv.Data {
		env.Variables[key] = normalizeVariables(stringValue(value))
	}
	return env, nil
}
