package install

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/sk8thing/arch-skm-go/internal/utils"
)

const keysDirName = "keys"

func InstallCertificates(destPath string) error {
	exePath, err := utils.GetExecutablePath()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	srcDir := filepath.Dir(exePath)
	srcKeysPath := filepath.Join(srcDir, keysDirName)
	destKeysPath := filepath.Join(destPath, keysDirName)

	if isSameOrSubdir(srcKeysPath, destKeysPath) {
		return fmt.Errorf("destination keys directory must differ from source: %s", destKeysPath)
	}

	if err := os.MkdirAll(destPath, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	if err := copyDir(srcKeysPath, destKeysPath); err != nil {
		return fmt.Errorf("failed to copy signing keys: %w", err)
	}

	if err := copyCurrentBinary(destPath); err != nil {
		return fmt.Errorf("failed to copy binary: %w", err)
	}

	return nil
}

func copyDir(srcDir string, destDir string) error {
	info, err := os.Stat(srcDir)
	if err != nil {
		return fmt.Errorf("access source directory %q: %w", srcDir, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("source path is not a directory: %s", srcDir)
	}

	return filepath.WalkDir(srcDir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return fmt.Errorf("resolve relative path for %q: %w", path, err)
		}
		destPath := filepath.Join(destDir, relPath)

		entryInfo, err := entry.Info()
		if err != nil {
			return fmt.Errorf("read file info for %q: %w", path, err)
		}
		mode := entryInfo.Mode().Perm()

		if entry.IsDir() {
			if err := os.MkdirAll(destPath, mode); err != nil {
				return fmt.Errorf("create directory %q: %w", destPath, err)
			}
			if err := os.Chmod(destPath, mode); err != nil {
				return fmt.Errorf("set directory permissions for %q: %w", destPath, err)
			}
			return nil
		}

		if !entry.Type().IsRegular() {
			return fmt.Errorf("unsupported file in key directory: %s", path)
		}

		if err := copyFile(path, destPath, mode); err != nil {
			return fmt.Errorf("copy %q: %w", path, err)
		}

		return nil
	})
}

func copyFile(srcPath string, destPath string, mode os.FileMode) error {
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("open source file: %w", err)
	}
	defer srcFile.Close()

	destFile, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return fmt.Errorf("create destination file: %w", err)
	}
	defer destFile.Close()

	if err := destFile.Chmod(mode); err != nil {
		return fmt.Errorf("set destination permissions: %w", err)
	}

	if _, err := io.Copy(destFile, srcFile); err != nil {
		return fmt.Errorf("copy file contents: %w", err)
	}

	return nil
}

func isSameOrSubdir(parent string, child string) bool {
	relPath, err := filepath.Rel(parent, child)
	if err != nil {
		return false
	}

	return relPath == "." || (relPath != ".." && !strings.HasPrefix(relPath, ".."+string(os.PathSeparator)))
}
