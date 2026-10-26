package sign

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	stripBinary     = "/usr/bin/strip"
	signatureMarker = "Module signature"
	signatureChunk  = 100
	tempPattern     = ".skm-*"
)

func signModule(signFile, keyFile, certFile, hashType, modPath string) error {
	info, err := os.Stat(modPath)
	if err != nil {
		return fmt.Errorf("access module: %w", err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("module is not a regular file")
	}

	ext := filepath.Ext(modPath)
	data, err := readModule(modPath, ext)
	if err != nil {
		return err
	}

	tmpPath, err := writeTempModule(filepath.Dir(modPath), data, info.Mode().Perm())
	if err != nil {
		return err
	}
	defer os.Remove(tmpPath)

	if isSigned(data) {
		if err := stripSignature(tmpPath); err != nil {
			return err
		}
	}

	if err := runSigner(signFile, hashType, keyFile, certFile, tmpPath); err != nil {
		return err
	}

	if isCompressed(ext) {
		signed, err := os.ReadFile(tmpPath)
		if err != nil {
			return fmt.Errorf("read signed module: %w", err)
		}

		compressed, err := compress(ext, signed)
		if err != nil {
			return err
		}

		if err := os.WriteFile(tmpPath, compressed, info.Mode().Perm()); err != nil {
			return fmt.Errorf("write compressed module: %w", err)
		}
	}

	if err := os.Rename(tmpPath, modPath); err != nil {
		return fmt.Errorf("replace module: %w", err)
	}

	return nil
}

func readModule(path, ext string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read module: %w", err)
	}

	return decompress(ext, data)
}

func writeTempModule(dir string, data []byte, mode os.FileMode) (string, error) {
	file, err := os.CreateTemp(dir, tempPattern)
	if err != nil {
		return "", fmt.Errorf("create temporary module: %w", err)
	}
	path := file.Name()

	if _, err := file.Write(data); err != nil {
		file.Close()
		os.Remove(path)
		return "", fmt.Errorf("write temporary module: %w", err)
	}

	if err := file.Chmod(mode); err != nil {
		file.Close()
		os.Remove(path)
		return "", fmt.Errorf("set temporary module permissions: %w", err)
	}

	if err := file.Close(); err != nil {
		os.Remove(path)
		return "", fmt.Errorf("close temporary module: %w", err)
	}

	return path, nil
}

func isSigned(data []byte) bool {
	start := 0
	if len(data)-signatureChunk >= signatureChunk {
		start = len(data) - signatureChunk
	}

	return bytes.Contains(data[start:], []byte(signatureMarker))
}

func stripSignature(path string) error {
	output, err := exec.Command(stripBinary, "--strip-debug", path).CombinedOutput()
	if err != nil {
		return commandError("strip module signature", err, output)
	}

	return nil
}

func runSigner(signFile, hashType, keyFile, certFile, modPath string) error {
	output, err := exec.Command(signFile, hashType, keyFile, certFile, modPath).CombinedOutput()
	if err != nil {
		return commandError("run kernel signer", err, output)
	}

	return nil
}

func commandError(action string, err error, output []byte) error {
	message := strings.TrimSpace(string(output))
	if message == "" {
		return fmt.Errorf("%s: %w", action, err)
	}

	return fmt.Errorf("%s: %w: %s", action, err, message)
}
