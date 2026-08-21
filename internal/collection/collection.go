package collection

import (
	"errors"
	"fmt"
	"io/fs"
	"net/url"
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

const (
	// ConfigDir and ConfigFile mark a directory as a collection by intent.
	ConfigDir  = "config"
	ConfigFile = "config.yaml"

	// MarkerFile is the legacy marker retained for migration.
	MarkerFile = ".lazypost"

	configVersion = 1
)

// Marker is the collection marker stored in config/config.yaml. Name and
// Root are populated only when loading a legacy .lazypost marker so the
// current session remains compatible until its first write.
type Marker struct {
	Version    int    `yaml:"version,omitempty"`
	Name       string `yaml:"-"`
	Root       string `yaml:"-"`
	Legacy     bool   `yaml:"-"`
	LegacyPath string `yaml:"-"`
}

// LoadMarker reads config/config.yaml, falling back to the legacy .lazypost
// marker. It returns (nil, nil) when neither marker exists.
func LoadMarker(dir string) (*Marker, error) {
	configPath := filepath.Join(dir, ConfigDir, ConfigFile)
	data, err := os.ReadFile(configPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
		legacyPath := filepath.Join(dir, MarkerFile)
		data, err = os.ReadFile(legacyPath)
		if err != nil {
			if os.IsNotExist(err) {
				return nil, nil
			}
			return nil, err
		}
		var legacy struct {
			Name string `yaml:"name"`
			Root string `yaml:"root,omitempty"`
		}
		if err := yaml.Unmarshal(data, &legacy); err != nil {
			return nil, fmt.Errorf("parsing %s: %w", legacyPath, err)
		}
		return &Marker{
			Name:       legacy.Name,
			Root:       legacy.Root,
			Legacy:     true,
			LegacyPath: legacyPath,
		}, nil
	}
	var m Marker
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", configPath, err)
	}
	if m.Version != configVersion {
		return nil, fmt.Errorf("unsupported config version %d in %s", m.Version, configPath)
	}
	return &m, nil
}

// WriteMarker creates the collection marker at dir and removes a legacy
// marker there after the new marker is safely written.
func WriteMarker(dir string) error {
	data, err := yaml.Marshal(Marker{Version: configVersion})
	if err != nil {
		return err
	}
	root, err := collectionRoot(dir)
	if err != nil {
		return err
	}
	configDir := filepath.Join(root, ConfigDir)
	if _, err := safeCollectionPath(root, configDir, false); err != nil {
		return err
	}
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return err
	}
	markerPath, err := safeCollectionPath(root, filepath.Join(configDir, ConfigFile), false)
	if err != nil {
		return err
	}
	if err := writeAtomic(markerPath, data, 0o644); err != nil {
		return err
	}
	if err := os.Remove(filepath.Join(root, MarkerFile)); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// MigrateLegacy writes a new marker at root and removes legacy marker paths
// after the write succeeds. Legacy name/root values are intentionally not
// copied into the new marker contract.
func MigrateLegacy(root string, legacyPaths ...string) error {
	if err := WriteMarker(root); err != nil {
		return err
	}
	for _, path := range legacyPaths {
		if filepath.Clean(path) == filepath.Clean(filepath.Join(root, MarkerFile)) {
			continue
		}
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

// collectionRoot validates root without resolving it to a different display
// path. Keeping the caller's relative/absolute spelling preserves the paths
// exposed by the collection tree while validation uses an absolute path.
func collectionRoot(root string) (string, error) {
	if strings.TrimSpace(root) == "" {
		return "", fmt.Errorf("collection root is required")
	}
	clean := filepath.Clean(root)
	abs, err := filepath.Abs(clean)
	if err != nil {
		return "", err
	}
	info, err := os.Lstat(abs)
	if err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("%w: %s", ErrSymlink, clean)
		}
		if !info.IsDir() {
			return "", fmt.Errorf("collection root is not a directory: %s", clean)
		}
		return clean, nil
	}
	if !os.IsNotExist(err) {
		return "", err
	}
	return clean, nil
}

// safeCollectionPath validates a path lexically and rejects existing
// symlink components so collection writes and deletes cannot follow a path
// outside root. Missing components are allowed because callers may create
// them immediately afterward.
func safeCollectionPath(root, path string, allowRoot bool) (string, error) {
	root = filepath.Clean(root)
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	if path == "" {
		return "", fmt.Errorf("collection path is required")
	}
	clean := filepath.Clean(path)
	if !filepath.IsAbs(clean) {
		candidateAbs, candidateErr := filepath.Abs(clean)
		if candidateErr != nil {
			return "", candidateErr
		}
		candidateRel, candidateErr := filepath.Rel(rootAbs, candidateAbs)
		if candidateErr != nil || candidateRel == ".." || strings.HasPrefix(candidateRel, ".."+string(filepath.Separator)) {
			clean = filepath.Join(root, clean)
		}
	}
	pathAbs, err := filepath.Abs(clean)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(rootAbs, pathAbs)
	if err != nil {
		return "", err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("%w: %s", ErrOutsideRoot, path)
	}
	if !allowRoot && rel == "." {
		return "", fmt.Errorf("%w: %s", ErrProtected, path)
	}
	if err := rejectSymlinkComponents(rootAbs, pathAbs); err != nil {
		return "", err
	}
	return clean, nil
}

func rejectSymlinkComponents(root, path string) error {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return err
	}
	if rel == "." {
		return nil
	}
	current := root
	for _, part := range strings.Split(rel, string(filepath.Separator)) {
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if os.IsNotExist(err) {
			return nil
		}
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("%w: %s", ErrSymlink, current)
		}
	}
	return nil
}

func fileMode(path string, fallback fs.FileMode) (fs.FileMode, error) {
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return fallback, nil
	}
	if err != nil {
		return 0, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return 0, fmt.Errorf("%w: %s", ErrSymlink, path)
	}
	if info.IsDir() {
		return 0, fmt.Errorf("path is a directory: %s", path)
	}
	return info.Mode().Perm(), nil
}

func writeTemp(path string, data []byte, mode fs.FileMode) (string, error) {
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".tmp-")
	if err != nil {
		return "", err
	}
	tmpPath := tmp.Name()
	cleanup := func() {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
	}
	if err := tmp.Chmod(mode); err != nil {
		cleanup()
		return "", err
	}
	if _, err := tmp.Write(data); err != nil {
		cleanup()
		return "", err
	}
	if err := tmp.Sync(); err != nil {
		cleanup()
		return "", err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return "", err
	}
	return tmpPath, nil
}

func writeAtomic(path string, data []byte, mode fs.FileMode) error {
	tmpPath, err := writeTemp(path, data, mode)
	if err != nil {
		return err
	}
	defer os.Remove(tmpPath)
	return os.Rename(tmpPath, path)
}

// writeAtomicNoReplace activates a staged file without replacing an existing
// destination. A same-directory hard link gives create operations atomic
// no-clobber behavior on the supported Unix platforms.
func writeAtomicNoReplace(path string, data []byte, mode fs.FileMode) error {
	tmpPath, err := writeTemp(path, data, mode)
	if err != nil {
		return err
	}
	defer os.Remove(tmpPath)
	if err := os.Link(tmpPath, path); err != nil {
		if os.IsExist(err) {
			return fmt.Errorf("%w: %s", ErrConflict, path)
		}
		return err
	}
	return nil
}

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
		// hidden entries (.git, .DS_Store, node_modules, …) are not part
		// of a collection — important when the root is the current
		// directory ([[Design - open the current directory as a
		// collection]])
		if strings.HasPrefix(d.Name(), ".") {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("%w: %s", ErrSymlink, path)
		}
		if d.IsDir() {
			// Root config/ and environments/ hold collection metadata, not
			// requests. Nested config/ folders remain ordinary user folders.
			if d.Name() == environmentsDir || (d.Name() == ConfigDir && path == filepath.Join(root, ConfigDir)) {
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

// DefaultName derives a request name from the last URL path segment when
// the user hasn't provided one (e.g. ".../posts/42" → "42").
func DefaultName(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err == nil && u.Path != "" && u.Path != "/" {
		segs := strings.Split(strings.Trim(u.Path, "/"), "/")
		last := segs[len(segs)-1]
		if last != "" {
			return Slug(last)
		}
	}
	return "untitled"
}

// Save writes r as YAML. If path is empty a path is derived from the
// request name inside root. The path written to is returned.
func Save(root, path string, r *Request) (string, error) {
	if r == nil {
		return "", fmt.Errorf("request is required")
	}
	if r.Name == "" {
		return "", fmt.Errorf("request name is required")
	}
	rootPath, err := collectionRoot(root)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(rootPath, 0o755); err != nil {
		return "", err
	}
	if path == "" {
		name := Slug(r.Name)
		if name == "" {
			return "", fmt.Errorf("%w: request name %q", ErrInvalidName, r.Name)
		}
		path = filepath.Join(rootPath, name+".yaml")
	}
	path, err = safeCollectionPath(rootPath, path, false)
	if err != nil {
		return "", err
	}
	// make parent dirs so nested saves (and first-time saves) succeed
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	if path, err = safeCollectionPath(rootPath, path, false); err != nil {
		return "", err
	}
	data, err := yaml.Marshal(r)
	if err != nil {
		return "", err
	}
	mode, err := fileMode(path, 0o600)
	if err != nil {
		return "", err
	}
	if err := writeAtomic(path, data, mode); err != nil {
		return "", err
	}
	return path, nil
}

var (
	// ErrProtected is returned when Delete is asked to remove something that
	// must never be deleted from the UI (the collection root, environments/).
	ErrProtected = errors.New("refusing to delete a protected path")
	// ErrConflict is returned when a create or rename would replace data.
	ErrConflict = errors.New("collection path already exists")
	// ErrOutsideRoot is returned when a collection operation escapes its root.
	ErrOutsideRoot = errors.New("collection path is outside the root")
	// ErrSymlink is returned when a collection operation would follow a symlink.
	ErrSymlink = errors.New("symlinks are not supported in collection paths")
	// ErrInvalidName is returned when a name cannot produce a safe path.
	ErrInvalidName = errors.New("name does not produce a safe path")
)

// CreateFolder creates a new directory under parent without replacing an
// existing path. The returned path is suitable for the collection tree.
func CreateFolder(root, parent, name string) (string, error) {
	rootPath, err := collectionRoot(root)
	if err != nil {
		return "", err
	}
	parent, err = safeCollectionPath(rootPath, parent, true)
	if err != nil {
		return "", err
	}
	info, err := os.Lstat(parent)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf("folder parent is not a directory: %s", parent)
	}
	slug := Slug(name)
	if slug == "" {
		return "", fmt.Errorf("%w: folder name %q", ErrInvalidName, name)
	}
	path, err := safeCollectionPath(rootPath, filepath.Join(parent, slug), false)
	if err != nil {
		return "", err
	}
	if err := os.Mkdir(path, 0o755); err != nil {
		if os.IsExist(err) {
			return "", fmt.Errorf("%w: %s", ErrConflict, path)
		}
		return "", err
	}
	return path, nil
}

// CreateRequest creates a blank request without replacing an existing file.
func CreateRequest(root, parent, name string) (*Request, string, error) {
	rootPath, err := collectionRoot(root)
	if err != nil {
		return nil, "", err
	}
	parent, err = safeCollectionPath(rootPath, parent, true)
	if err != nil {
		return nil, "", err
	}
	info, err := os.Lstat(parent)
	if err != nil {
		return nil, "", err
	}
	if !info.IsDir() {
		return nil, "", fmt.Errorf("request parent is not a directory: %s", parent)
	}
	slug := Slug(name)
	if slug == "" {
		return nil, "", fmt.Errorf("%w: request name %q", ErrInvalidName, name)
	}
	path, err := safeCollectionPath(rootPath, filepath.Join(parent, slug+".yaml"), false)
	if err != nil {
		return nil, "", err
	}
	req := &Request{Name: name, Method: "GET"}
	data, err := yaml.Marshal(req)
	if err != nil {
		return nil, "", err
	}
	if err := writeAtomicNoReplace(path, data, 0o600); err != nil {
		return nil, "", err
	}
	return req, path, nil
}

// Rename rewrites the request at oldPath under a new slug path derived
// from name, then removes the old file. The renamed request is returned
// with the new path.
func Rename(root, oldPath, name string) (*Request, string, error) {
	rootPath, err := collectionRoot(root)
	if err != nil {
		return nil, "", err
	}
	oldPath, err = safeCollectionPath(rootPath, oldPath, false)
	if err != nil {
		return nil, "", err
	}
	req, err := LoadFile(oldPath)
	if err != nil {
		return nil, "", err
	}
	req.Name = name
	slug := Slug(name)
	if slug == "" {
		return nil, "", fmt.Errorf("%w: request name %q", ErrInvalidName, name)
	}
	newPath, err := safeCollectionPath(rootPath, filepath.Join(filepath.Dir(oldPath), slug+".yaml"), false)
	if err != nil {
		return nil, "", err
	}
	if filepath.Clean(newPath) == filepath.Clean(oldPath) {
		if _, err := Save(rootPath, newPath, req); err != nil {
			return nil, "", err
		}
		return req, newPath, nil
	}
	if _, err := os.Lstat(newPath); err == nil {
		return nil, "", fmt.Errorf("%w: %s", ErrConflict, newPath)
	} else if !os.IsNotExist(err) {
		return nil, "", err
	}
	if _, err := Save(rootPath, newPath, req); err != nil {
		return nil, "", err
	}
	if err := os.Remove(oldPath); err != nil {
		_ = os.Remove(newPath)
		return nil, "", err
	}
	return req, newPath, nil
}

// Delete removes a request file or a directory subtree (a folder in the
// collection). The collection root and environments/ are protected.
func Delete(root, path string) error {
	rootPath, err := collectionRoot(root)
	if err != nil {
		return err
	}
	clean, err := safeCollectionPath(rootPath, path, true)
	if err != nil {
		return err
	}
	if filepath.Clean(clean) == filepath.Clean(rootPath) {
		return ErrProtected
	}
	rel, err := filepath.Rel(rootPath, clean)
	if err != nil {
		return err
	}
	if rel == environmentsDir || strings.HasPrefix(rel, environmentsDir+string(filepath.Separator)) {
		return ErrProtected
	}
	fi, err := os.Lstat(clean)
	if err != nil {
		return err
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("%w: %s", ErrSymlink, clean)
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
	return saveEnvironment(root, name, vars, false)
}

// CreateEnvironment creates an environment without replacing an existing
// environment file.
func CreateEnvironment(root, name string, vars map[string]string) error {
	return saveEnvironment(root, name, vars, true)
}

func saveEnvironment(root, name string, vars map[string]string, noReplace bool) error {
	rootPath, err := collectionRoot(root)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(rootPath, 0o755); err != nil {
		return err
	}
	slug := Slug(name)
	if slug == "" {
		return fmt.Errorf("%w: environment name %q", ErrInvalidName, name)
	}
	dir := filepath.Join(rootPath, environmentsDir)
	if _, err := safeCollectionPath(rootPath, dir, false); err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	if dir, err = safeCollectionPath(rootPath, dir, false); err != nil {
		return err
	}
	if vars == nil {
		vars = map[string]string{}
	}
	data, err := yaml.Marshal(Environment{Variables: vars})
	if err != nil {
		return err
	}
	path, err := safeCollectionPath(rootPath, filepath.Join(dir, slug+".yaml"), false)
	if err != nil {
		return err
	}
	mode, err := fileMode(path, 0o600)
	if err != nil {
		return err
	}
	if noReplace {
		return writeAtomicNoReplace(path, data, mode)
	}
	return writeAtomic(path, data, mode)
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
		path := filepath.Join(dir, item.Name())
		info, err := os.Lstat(path)
		if err != nil {
			return nil, nil, err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil, nil, fmt.Errorf("%w: %s", ErrSymlink, path)
		}
		data, err := os.ReadFile(path)
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
