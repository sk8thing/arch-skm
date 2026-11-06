package helper

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const installedPath = "/usr/lib/skm/skm"

func Resolve(explicitPath string) (string, error) {
	if explicitPath != "" {
		return resolvePath("--skm", explicitPath)
	}

	if envPath := os.Getenv("SKM_HELPER"); envPath != "" {
		return resolvePath("SKM_HELPER", envPath)
	}

	var candidates []string
	if exePath, err := ExecutablePath(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(exePath), "skm"))
	}
	candidates = append(candidates, installedPath)

	for _, candidate := range candidates {
		path, err := resolvePath("", candidate)
		if err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("runtime helper skm not found; looked in: %s; set SKM_HELPER or use --skm", strings.Join(candidates, ", "))
}

func resolvePath(source, path string) (string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve %s %q: %w", source, path, err)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return "", fmt.Errorf("access %s %q: %w", source, absPath, err)
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("%s %q is not a regular file", source, absPath)
	}
	if info.Mode().Perm()&0111 == 0 {
		return "", fmt.Errorf("%s %q is not executable", source, absPath)
	}

	return absPath, nil
}
