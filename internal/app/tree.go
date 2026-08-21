package app

import (
	"github.com/Dread-Code/lazypost/internal/collection"
)

// Collection operations: each mutates the collection tree on disk and
// returns the freshly reloaded tree, so "reload after every mutation"
// is an invariant owned by this layer instead of repeated at every call
// site in the model. The functions never touch UI types — the model
// applies the returned tree and its own UI effects.
//
// A reload failure after a successful mutation surfaces as an error, so
// the caller can tell the user the sidebar is stale.

// SaveRequest writes req to path (or derives one from its name) and
// returns the path written plus the reloaded tree.
func SaveRequest(dir, path string, req *collection.Request) (string, []collection.Entry, error) {
	newPath, err := collection.Save(dir, path, req)
	if err != nil {
		return "", nil, err
	}
	entries, err := collection.Load(dir)
	if err != nil {
		return "", nil, err
	}
	return newPath, entries, nil
}

// RenameRequest moves oldPath to a new slug path and returns the renamed
// request, its new path, and the reloaded tree.
func RenameRequest(dir, oldPath, name string) (*collection.Request, string, []collection.Entry, error) {
	req, newPath, err := collection.Rename(dir, oldPath, name)
	if err != nil {
		return nil, "", nil, err
	}
	entries, err := collection.Load(dir)
	if err != nil {
		return nil, "", nil, err
	}
	return req, newPath, entries, nil
}

// CreateFolder makes a new directory under parent and returns its path
// plus the reloaded tree.
func CreateFolder(root, parent, name string) (string, []collection.Entry, error) {
	path, err := collection.CreateFolder(root, parent, name)
	if err != nil {
		return "", nil, err
	}
	entries, err := collection.Load(root)
	if err != nil {
		return "", nil, err
	}
	return path, entries, nil
}

// CreateRequest writes a blank named request under parent and returns
// the request, its path, and the reloaded tree.
func CreateRequest(root, parent, name string) (*collection.Request, string, []collection.Entry, error) {
	req, path, err := collection.CreateRequest(root, parent, name)
	if err != nil {
		return nil, "", nil, err
	}
	entries, err := collection.Load(root)
	if err != nil {
		return nil, "", nil, err
	}
	return req, path, entries, nil
}

// DeleteEntry removes e from the tree and returns the reloaded tree.
func DeleteEntry(dir string, e *collection.Entry) ([]collection.Entry, error) {
	if err := collection.Delete(dir, e.Path); err != nil {
		return nil, err
	}
	return collection.Load(dir)
}

// MigrateCollection writes the new config marker and removes legacy marker
// paths. The new contract intentionally does not carry over name/root.
func MigrateCollection(root string, legacyPaths []string) error {
	return collection.MigrateLegacy(root, legacyPaths...)
}
