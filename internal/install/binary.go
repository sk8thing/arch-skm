package install

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/sk8thing/arch-skm-go/internal/utils"
)

func copyCurrentBinary(destPath string) error {
	exePath, err := utils.GetExecutablePath()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	srcFile, err := os.Open(exePath)
	if err != nil {
		return fmt.Errorf("failed to open executable: %w", err)
	}
	defer srcFile.Close()

	binaryPath := filepath.Join(destPath, "skm")
	destFile, err := os.Create(binaryPath)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, srcFile); err != nil {
		return fmt.Errorf("failed to copy binary: %w", err)
	}

	if err := destFile.Chmod(0755); err != nil {
		return fmt.Errorf("failed to set permissions: %w", err)
	}

	return nil
}
