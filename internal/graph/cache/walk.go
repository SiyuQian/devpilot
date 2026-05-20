package cache

import (
	"io/fs"
	"path/filepath"
	"strings"
)

// WalkRepo returns every regular file in root, skipping VCS and dependency
// directories. Paths returned are repo-relative with forward slashes.
func WalkRepo(root string) ([]string, error) {
	var out []string
	skipDirs := map[string]bool{
		".git": true, "node_modules": true, "target": true, "vendor": true, ".devpilot": true,
	}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path == root {
				return nil
			}
			if skipDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		out = append(out, filepath.ToSlash(rel))
		return nil
	})
	return out, err
}

// FilterByParser keeps only files whose extension is recognised by parser.Registry.
// Callers pass a probe function so cache does not need to import parser directly.
func FilterByParser(paths []string, supported func(path string) bool) []string {
	var out []string
	for _, p := range paths {
		if supported(p) {
			out = append(out, p)
		}
	}
	return out
}

// IsHidden returns true for dot-prefixed segments other than "." and "..".
func IsHidden(name string) bool {
	return strings.HasPrefix(name, ".") && name != "." && name != ".."
}
