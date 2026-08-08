package collection

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type Header struct {
	Name  string `yaml:"name"`
	Value string `yaml:"value"`
}

// Param is one name/value query parameter.
type Param struct {
	Name  string `yaml:"name"`
	Value string `yaml:"value"`
}

type Auth struct {
	Type     string `yaml:"type"` // none, basic, bearer, apikey
	Username string `yaml:"username,omitempty"`
	Password string `yaml:"password,omitempty"`
	Token    string `yaml:"token,omitempty"`
	KeyName  string `yaml:"keyName,omitempty"`
	KeyValue string `yaml:"keyValue,omitempty"`
	KeyIn    string `yaml:"keyIn,omitempty"` // header or query
}

type Request struct {
	Name    string   `yaml:"name"`
	Method  string   `yaml:"method"`
	URL     string   `yaml:"url"`
	Query   []Param  `yaml:"query,omitempty"`
	Headers []Header `yaml:"headers,omitempty"`
	Auth    *Auth    `yaml:"auth,omitempty"`
	Body    string   `yaml:"body,omitempty"`
	Pre     string   `yaml:"pre,omitempty"`
	Post    string   `yaml:"post,omitempty"`
}

type Kind int

const (
	Dir Kind = iota
	Req
)

// Entry is one node of the collection tree: a directory or a request,
// with its nesting depth and on-disk path.
type Entry struct {
	Kind  Kind
	Name  string
	Depth int
	Path  string
	Req   *Request
}

const environmentsDir = "environments"

func isYAML(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".yaml" || ext == ".yml"
}

// LoadFile reads and parses a single request file, defaulting the method
// to GET when omitted.
func LoadFile(path string) (*Request, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var r Request
	if err := yaml.Unmarshal(data, &r); err != nil {
		return nil, err
	}
	if r.Method == "" {
		r.Method = "GET"
	}
	return &r, nil
}

// Load walks root and returns a flattened, depth-annotated tree of
// directories and requests in lexical order.
func Load(root string) ([]Entry, error) {
	var entries []Entry
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == root {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		depth := strings.Count(rel, string(filepath.Separator))
		if d.IsDir() {
			// environments/ holds variable sets, not requests — skip it
			if d.Name() == environmentsDir {
				return filepath.SkipDir
			}
			entries = append(entries, Entry{Kind: Dir, Name: d.Name(), Depth: depth, Path: path})
			return nil
		}
		if !isYAML(path) {
			return nil
		}
		req, err := LoadFile(path)
		if err != nil {
			return fmt.Errorf("loading %s: %w", path, err)
		}
		name := req.Name
		if name == "" {
			// fall back to the filename so anonymous files still show up
			name = strings.TrimSuffix(d.Name(), filepath.Ext(path))
		}
		entries = append(entries, Entry{Kind: Req, Name: name, Depth: depth, Path: path, Req: req})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return entries, nil
}

var slugRe = regexp.MustCompile(`[^a-z0-9]+`)

// Slug lowercases name and collapses runs of non-alphanumerics into a
// single dash, yielding a safe filename (e.g. "create post" → "create-post").
func Slug(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	s = slugRe.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}

// Save writes r as YAML. If path is empty a path is derived from the
// request name inside root. The path written to is returned.
func Save(root, path string, r *Request) (string, error) {
	if r.Name == "" {
		return "", fmt.Errorf("request name is required")
	}
	if path == "" {
		path = filepath.Join(root, Slug(r.Name)+".yaml")
	}
	// make parent dirs so nested saves (and first-time saves) succeed
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	data, err := yaml.Marshal(r)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", err
	}
	return path, nil
}

// ErrProtected is returned when Delete is asked to remove something that
// must never be deleted from the UI (the collection root, environments/).
var ErrProtected = errors.New("refusing to delete a protected path")

// Rename rewrites the request at oldPath under a new slug path derived
// from name, then removes the old file. The renamed request is returned
// with the new path.
func Rename(root, oldPath, name string) (*Request, string, error) {
	req, err := LoadFile(oldPath)
	if err != nil {
		return nil, "", err
	}
	req.Name = name
	newPath := filepath.Join(filepath.Dir(oldPath), Slug(name)+".yaml")
	if _, err := Save(root, newPath, req); err != nil {
		return nil, "", err
	}
	if err := os.Remove(oldPath); err != nil {
		return nil, "", err
	}
	return req, newPath, nil
}

// Delete removes a request file or a directory subtree (a folder in the
// collection). The collection root and environments/ are protected.
func Delete(root, path string) error {
	clean := filepath.Clean(path)
	if clean == filepath.Clean(root) {
		return ErrProtected
	}
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return err
	}
	if rel == environmentsDir || strings.HasPrefix(rel, environmentsDir+string(filepath.Separator)) {
		return ErrProtected
	}
	fi, err := os.Stat(clean)
	if err != nil {
		return err
	}
	if fi.IsDir() {
		return os.RemoveAll(clean)
	}
	return os.Remove(clean)
}

type Environment struct {
	Variables map[string]string `yaml:"variables"`
}

// SaveEnvironment writes (or replaces) an environment file
// <root>/environments/<slug>.yaml.
func SaveEnvironment(root, name string, vars map[string]string) error {
	dir := filepath.Join(root, environmentsDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	if vars == nil {
		vars = map[string]string{}
	}
	data, err := yaml.Marshal(Environment{Variables: vars})
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, Slug(name)+".yaml"), data, 0o644)
}

// LoadEnvironments reads <root>/environments/*.yaml and returns the// variables keyed by environment name plus the sorted list of names.
func LoadEnvironments(root string) (map[string]map[string]string, []string, error) {
	dir := filepath.Join(root, environmentsDir)
	envs := map[string]map[string]string{}
	items, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return envs, nil, nil
		}
		return nil, nil, err
	}
	var names []string
	for _, item := range items {
		if item.IsDir() || !isYAML(item.Name()) {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, item.Name()))
		if err != nil {
			return nil, nil, err
		}
		var env Environment
		if err := yaml.Unmarshal(data, &env); err != nil {
			return nil, nil, fmt.Errorf("loading environment %s: %w", item.Name(), err)
		}
		name := strings.TrimSuffix(item.Name(), filepath.Ext(item.Name()))
		if env.Variables == nil {
			env.Variables = map[string]string{}
		}
		envs[name] = env.Variables
		names = append(names, name)
	}
	sort.Strings(names)
	return envs, names, nil
}
