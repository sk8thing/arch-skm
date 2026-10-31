package patch

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	pkgbuildFile  = "PKGBUILD"
	SigningMarker = "Out-of-tree module signing"
)

func PatchPKGBUILD(path string) error {
	pkgbuildPath := filepath.Join(path, pkgbuildFile)
	file, err := os.ReadFile(pkgbuildPath)
	if err != nil {
		return fmt.Errorf("read PKGBUILD %q: %w", pkgbuildPath, err)
	}

	configAtStart, err := hasKernelConfig(path)
	if err != nil {
		return err
	}

	steps := []func(string) (string, error){
		func(content string) (string, error) {
			return patchPrepareFunction(content, configAtStart)
		},
		patchBuildFunction,
		func(content string) (string, error) {
			return patchPackageHeadersFunction(content, configAtStart)
		},
		patchPkgnameDocs,
	}

	content := string(file)
	updatedContent := content
	for _, patchFn := range steps {
		updatedContent, err = patchFn(updatedContent)
		if err != nil {
			return fmt.Errorf("patch PKGBUILD %q: %w", pkgbuildPath, err)
		}
	}

	if updatedContent == content {
		return nil
	}

	if err := os.WriteFile(pkgbuildPath, []byte(updatedContent), 0644); err != nil {
		return fmt.Errorf("write PKGBUILD %q: %w", pkgbuildPath, err)
	}

	return nil
}

func hasKernelConfig(dir string) (bool, error) {
	matches, err := filepath.Glob(filepath.Join(dir, "config*"))
	if err != nil {
		return false, fmt.Errorf("find kernel config in %q: %w", dir, err)
	}

	for _, match := range matches {
		info, err := os.Stat(match)
		if err == nil && info.Mode().IsRegular() {
			return true, nil
		}
	}

	return false, nil
}
