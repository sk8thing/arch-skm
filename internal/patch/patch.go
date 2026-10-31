package patch

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	pkgbuildFile = "PKGBUILD"

	SigningMarker = "Out-of-tree module signing"
)

var patches = []func(string) (string, error){
	patchPrepareFunction,
	patchBuildFunction,
	patchPackageHeadersFunction,
	patchPkgnameDocs,
}

func PatchPKGBUILD(path string) error {
	pkgbuildPath := filepath.Join(path, pkgbuildFile)
	file, err := os.ReadFile(pkgbuildPath)
	if err != nil {
		return fmt.Errorf("read PKGBUILD %q: %w", pkgbuildPath, err)
	}

	content := string(file)
	updatedContent := content
	for _, patchFn := range patches {
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
