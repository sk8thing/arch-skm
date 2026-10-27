package install

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/sk8thing/arch-skm-go/internal/utils"
)

func copyCurrentBinary(destPath string) error {
	exePath, err := utils.GetExecutablePath()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	return copyBinary(exePath, filepath.Join(destPath, "skm"))
}

func copyBinary(srcPath, destPath string) error {
	srcAbs, err := filepath.Abs(srcPath)
	if err != nil {
		return fmt.Errorf("resolve source binary %q: %w", srcPath, err)
	}
	destAbs, err := filepath.Abs(destPath)
	if err != nil {
		return fmt.Errorf("resolve destination binary %q: %w", destPath, err)
	}
	if srcAbs == destAbs {
		return nil
	}

	info, err := os.Stat(srcAbs)
	if err != nil {
		return fmt.Errorf("access source binary %q: %w", srcAbs, err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("source binary %q is not a regular file", srcAbs)
	}

	if err := os.MkdirAll(filepath.Dir(destAbs), 0755); err != nil {
		return fmt.Errorf("create destination directory: %w", err)
	}

	if err := copyFile(srcAbs, destAbs, 0755); err != nil {
		return fmt.Errorf("copy %q to %q: %w", srcAbs, destAbs, err)
	}

	return nil
}
