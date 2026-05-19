package statefiles

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"terraform-cmdb/internal/inventory"
	"terraform-cmdb/internal/terraformstate"
)

type LoadResult struct {
	Snapshot inventory.Snapshot
	Files    []string
	Errors   []error
}

func LoadDirectory(dir string) LoadResult {
	result := LoadResult{}

	entries, err := stateFilePaths(dir)
	if err != nil {
		result.Errors = append(result.Errors, err)
		result.Snapshot.LastError = joinErrors(result.Errors)
		return result
	}

	var terraformVersions []string
	for _, path := range entries {
		content, err := os.ReadFile(path)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("%s: %w", path, err))
			continue
		}

		parsed, err := terraformstate.Parse(content)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("%s: %w", path, err))
			continue
		}

		result.Files = append(result.Files, path)
		result.Snapshot.RawResources += parsed.RawResources
		result.Snapshot.Machines = append(result.Snapshot.Machines, parsed.Machines...)
		if parsed.Terraform != "" {
			terraformVersions = appendUnique(terraformVersions, parsed.Terraform)
		}
	}

	result.Snapshot.FileName = sourceName(dir, result.Files)
	result.Snapshot.SourceFiles = append([]string(nil), result.Files...)
	result.Snapshot.Terraform = strings.Join(terraformVersions, ", ")
	result.Snapshot.LastError = joinErrors(result.Errors)
	return result
}

func stateFilePaths(dir string) ([]string, error) {
	info, err := os.Stat(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s 不是目录", dir)
	}

	var paths []string
	visited := map[string]bool{}
	if err := walkStateFiles(dir, visited, &paths); err != nil {
		return nil, err
	}

	sort.Strings(paths)
	return paths, nil
}

func walkStateFiles(dir string, visited map[string]bool, paths *[]string) error {
	realDir, err := filepath.EvalSymlinks(dir)
	if err != nil {
		return err
	}
	if visited[realDir] {
		return nil
	}
	visited[realDir] = true

	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		path := filepath.Join(dir, entry.Name())
		if entry.IsDir() {
			if shouldSkipDir(entry.Name()) {
				continue
			}
			if err := walkStateFiles(path, visited, paths); err != nil {
				return err
			}
			continue
		}

		if entry.Type()&os.ModeSymlink != 0 {
			targetInfo, err := os.Stat(path)
			if err != nil {
				return err
			}
			if targetInfo.IsDir() {
				if shouldSkipDir(entry.Name()) {
					continue
				}
				if err := walkStateFiles(path, visited, paths); err != nil {
					return err
				}
				continue
			}
		}

		if isStateFile(path) {
			*paths = append(*paths, path)
		}
	}
	return nil
}

func shouldSkipDir(name string) bool {
	switch name {
	case ".git", ".terraform", "node_modules", ".idea", ".vscode":
		return true
	default:
		return false
	}
}

func isStateFile(path string) bool {
	name := strings.ToLower(filepath.Base(path))
	return strings.HasSuffix(name, ".json") || strings.HasSuffix(name, ".tfstate")
}

func sourceName(dir string, files []string) string {
	if len(files) == 0 {
		return dir + " (0 files)"
	}
	return fmt.Sprintf("%s (%d files)", dir, len(files))
}

func joinErrors(errs []error) string {
	if len(errs) == 0 {
		return ""
	}

	var parts []string
	for _, err := range errs {
		parts = append(parts, err.Error())
	}
	return strings.Join(parts, "; ")
}

func appendUnique(values []string, additions ...string) []string {
	seen := map[string]bool{}
	for _, value := range values {
		seen[value] = true
	}
	for _, value := range additions {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		values = append(values, value)
	}
	sort.Strings(values)
	return values
}
