package install

import (
	"fmt"
	"os"
	"path/filepath"
)

func PrepareBuildDir(destPath, skmPath string) error {
	if skmPath == "" {
		return fmt.Errorf("skm binary path is required")
	}

	certsLocalPath := filepath.Join(destPath, "certs-local")
	if err := os.MkdirAll(certsLocalPath, 0755); err != nil {
		return fmt.Errorf("create certs-local directory: %w", err)
	}

	if err := copyBinary(skmPath, filepath.Join(certsLocalPath, "skm")); err != nil {
		return fmt.Errorf("copy skm binary: %w", err)
	}

	if err := copyDkmsFiles(certsLocalPath); err != nil {
		return fmt.Errorf("copy DKMS files: %w", err)
	}

	return nil
}
