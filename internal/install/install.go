package install

import (
	"fmt"
	"os"
	"path/filepath"
)

func InstallFiles(destPath string) error {
	certsLocalPath := filepath.Join(destPath, "certs-local")
	if err := os.MkdirAll(certsLocalPath, 0755); err != nil {
		return fmt.Errorf("failed to create certs-local directory: %w", err)
	}

	if err := copyCurrentBinary(certsLocalPath); err != nil {
		return fmt.Errorf("failed to copy binary: %w", err)
	}

	if err := copyDkmsFiles(certsLocalPath); err != nil {
		return fmt.Errorf("failed to copy DKMS files: %w", err)
	}

	return nil
}
