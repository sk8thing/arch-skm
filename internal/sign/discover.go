package sign

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var moduleExtensions = []string{".ko", ".ko.zst", ".ko.xz", ".ko.gz"}

func FindModules(dir string) ([]string, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("resolve module directory %q: %w", dir, err)
	}

	info, err := os.Stat(absDir)
	if err != nil {
		return nil, fmt.Errorf("access module directory %q: %w", absDir, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("module path is not a directory: %s", absDir)
	}

	entries, err := os.ReadDir(absDir)
	if err != nil {
		return nil, fmt.Errorf("read module directory %q: %w", absDir, err)
	}

	var modules []string
	for _, entry := range entries {
		if entry.IsDir() || !isModuleName(entry.Name()) {
			continue
		}
		modules = append(modules, filepath.Join(absDir, entry.Name()))
	}

	return modules, nil
}

func isModuleName(name string) bool {
	for _, ext := range moduleExtensions {
		if strings.HasSuffix(name, ext) {
			return true
		}
	}

	return false
}
