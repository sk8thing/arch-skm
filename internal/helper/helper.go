package helper

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const installedPath = "/usr/lib/skm/skm"

func VerifyExecutableFile(label, path string) error {
	return verifyFile(label, path, true)
}

func VerifyRegularFile(label, path string) error {
	return verifyFile(label, path, false)
}

func verifyFile(label, path string, executable bool) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("access %s %q: %w", label, path, err)
	}

	if !info.Mode().IsRegular() {
		return fmt.Errorf("%s %q is not a regular file", label, path)
	}
	if executable && info.Mode().Perm()&0111 == 0 {
		return fmt.Errorf("%s %q is not executable", label, path)
	}

	return nil
}

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

	if err := VerifyExecutableFile(source, absPath); err != nil {
		return "", err
	}

	return absPath, nil
}
