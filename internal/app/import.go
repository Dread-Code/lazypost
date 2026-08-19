package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"lazypost/internal/collection"
	"lazypost/internal/importer"
)

type ImportOptions struct {
	Target string
	Force  bool
	DryRun bool
	Strict bool
}

type ImportSummary struct {
	SourceName   string
	Target       string
	Folders      int
	Requests     int
	Environments int
	Warnings     []importer.Warning
}

type plannedRequest struct {
	Path string
	Req  collection.Request
}

type importPlan struct {
	Requests     []plannedRequest
	Folders      [][]string
	Environments []importer.ImportedEnvironment
	Warnings     []importer.Warning
}

// ImportCollection validates a normalized import, then writes it into a
// staged sibling directory before replacing the target. Parsing and all
// collision checks happen before the first output mutation.
func ImportCollection(result importer.Result, opts ImportOptions) (ImportSummary, error) {
	if opts.Target == "" {
		return ImportSummary{}, fmt.Errorf("import target is required")
	}
	plan := planImport(result)
	warnings := append(append([]importer.Warning{}, result.Warnings...), plan.Warnings...)
	summary := ImportSummary{
		SourceName:   result.Name,
		Target:       opts.Target,
		Folders:      len(plan.Folders),
		Requests:     len(plan.Requests),
		Environments: len(plan.Environments),
		Warnings:     warnings,
	}
	if opts.Strict && len(warnings) > 0 {
		return summary, fmt.Errorf("strict import rejected %d warning(s)", len(warnings))
	}
	if opts.DryRun {
		return summary, nil
	}

	if info, err := os.Stat(opts.Target); err == nil {
		if !info.IsDir() {
			return summary, fmt.Errorf("import target exists and is not a directory: %s", opts.Target)
		}
		if !opts.Force {
			return summary, fmt.Errorf("import target already exists; use --force to replace it: %s", opts.Target)
		}
	} else if !os.IsNotExist(err) {
		return summary, err
	}

	parent := filepath.Dir(filepath.Clean(opts.Target))
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return summary, err
	}
	stage, err := os.MkdirTemp(parent, ".lazypost-import-")
	if err != nil {
		return summary, err
	}
	defer os.RemoveAll(stage)

	if err := collection.WriteMarker(stage); err != nil {
		return summary, fmt.Errorf("initialize staged collection: %w", err)
	}
	for _, folder := range plan.Folders {
		if path := safeImportPath(stage, folder...); path != stage {
			if err := os.MkdirAll(path, 0o755); err != nil {
				return summary, fmt.Errorf("create folder %s: %w", filepath.Join(folder...), err)
			}
		}
	}
	for _, item := range plan.Requests {
		if _, err := collection.Save(stage, filepath.Join(stage, item.Path), &item.Req); err != nil {
			return summary, fmt.Errorf("write request %s: %w", item.Path, err)
		}
	}
	for _, env := range plan.Environments {
		if err := collection.SaveEnvironment(stage, env.Name, env.Variables); err != nil {
			return summary, fmt.Errorf("write environment %s: %w", env.Name, err)
		}
	}
	if err := replaceTarget(opts.Target, stage, opts.Force); err != nil {
		return summary, err
	}
	return summary, nil
}

func planImport(result importer.Result) importPlan {
	plan := importPlan{}
	used := map[string]bool{}
	multipleWorkspaces := len(result.Workspaces) > 1
	for _, workspace := range result.Workspaces {
		workspaceName := workspace.Name
		if workspaceName == "" {
			workspaceName = "workspace"
		}
		workspacePath := []string{workspaceName}
		plan.Folders = append(plan.Folders, workspacePath)
		for _, folder := range workspace.Folders {
			plan.Folders = append(plan.Folders, append(append([]string{}, workspacePath...), folder...))
		}
		for _, env := range workspace.Environments {
			if multipleWorkspaces {
				env.Name = workspaceName + "--" + env.Name
			}
			plan.Environments = append(plan.Environments, env)
		}
		for _, item := range workspace.Requests {
			segments := make([]string, 0, len(item.Path)+len(workspacePath))
			for _, part := range append(workspacePath, item.Path...) {
				if safe := collection.Slug(part); safe != "" {
					segments = append(segments, safe)
				}
			}
			name := collection.Slug(item.Request.Name)
			if name == "" {
				name = "untitled"
				plan.Warnings = append(plan.Warnings, importer.Warning{Path: strings.Join(append(workspacePath, item.Path...), "/"), Message: "empty request name replaced with untitled"})
			}
			base := filepath.Join(append(segments, name+".yaml")...)
			candidate := base
			for n := 2; used[candidate]; n++ {
				ext := filepath.Ext(base)
				stem := strings.TrimSuffix(base, ext)
				candidate = stem + "-" + strconv.Itoa(n) + ext
			}
			if candidate != base {
				plan.Warnings = append(plan.Warnings, importer.Warning{Path: strings.Join(append(append(workspacePath, item.Path...), item.Request.Name), "/"), Message: "filename collision resolved as " + candidate})
			}
			used[candidate] = true
			plan.Requests = append(plan.Requests, plannedRequest{Path: candidate, Req: item.Request})
		}
	}
	return plan
}

func safeImportPath(root string, parts ...string) string {
	path := root
	for _, part := range parts {
		if safe := collection.Slug(part); safe != "" {
			path = filepath.Join(path, safe)
		}
	}
	return path
}

func replaceTarget(target, stage string, force bool) error {
	if !force {
		return os.Rename(stage, target)
	}
	parent := filepath.Dir(filepath.Clean(target))
	if _, err := os.Stat(target); os.IsNotExist(err) {
		return os.Rename(stage, target)
	} else if err != nil {
		return err
	}
	backup, err := os.MkdirTemp(parent, ".lazypost-import-backup-")
	if err != nil {
		return err
	}
	if err := os.Remove(backup); err != nil {
		return err
	}
	if err := os.Rename(target, backup); err != nil {
		return fmt.Errorf("stage existing target: %w", err)
	}
	if err := os.Rename(stage, target); err != nil {
		_ = os.Rename(backup, target)
		return fmt.Errorf("activate imported collection: %w", err)
	}
	if err := os.RemoveAll(backup); err != nil {
		return fmt.Errorf("remove import backup: %w", err)
	}
	return nil
}
