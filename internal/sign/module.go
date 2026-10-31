package sign

import (
	"bytes"
	"fmt"
	"io"
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
	if isCompressed(ext) {
		return signCompressedModule(signFile, keyFile, certFile, hashType, modPath, ext, info)
	}

	return signPlainModule(signFile, keyFile, certFile, hashType, modPath, info)
}

func signPlainModule(signFile, keyFile, certFile, hashType, modPath string, info os.FileInfo) error {
	signed, err := isSignedFile(modPath)
	if err != nil {
		return err
	}

	tmpPath, err := copyTempModule(filepath.Dir(modPath), modPath)
	if err != nil {
		return err
	}
	defer os.Remove(tmpPath)

	if signed {
		if err := stripSignature(tmpPath); err != nil {
			return err
		}
	}

	if err := runSigner(signFile, hashType, keyFile, certFile, tmpPath); err != nil {
		return err
	}

	return replaceModule(tmpPath, modPath, info.Mode().Perm())
}

func signCompressedModule(signFile, keyFile, certFile, hashType, modPath, ext string, info os.FileInfo) error {
	data, err := readModule(modPath, ext)
	if err != nil {
		return err
	}

	tmpPath, err := writeTempModule(filepath.Dir(modPath), data)
	if err != nil {
		return err
	}
	defer os.Remove(tmpPath)

	if isSigned(data) {
		if err := stripSignature(tmpPath); err != nil {
			return err
		}
	}
	data = nil

	if err := runSigner(signFile, hashType, keyFile, certFile, tmpPath); err != nil {
		return err
	}

	signed, err := os.ReadFile(tmpPath)
	if err != nil {
		return fmt.Errorf("read signed module: %w", err)
	}

	compressed, err := compress(ext, signed)
	if err != nil {
		return err
	}

	if err := os.WriteFile(tmpPath, compressed, 0600); err != nil {
		return fmt.Errorf("write compressed module: %w", err)
	}

	return replaceModule(tmpPath, modPath, info.Mode().Perm())
}

func replaceModule(tmpPath, modPath string, mode os.FileMode) error {
	if err := os.Chmod(tmpPath, mode); err != nil {
		return fmt.Errorf("set module permissions: %w", err)
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

func copyTempModule(dir, srcPath string) (string, error) {
	src, err := os.Open(srcPath)
	if err != nil {
		return "", fmt.Errorf("read module: %w", err)
	}
	defer src.Close()

	file, err := os.CreateTemp(dir, tempPattern)
	if err != nil {
		return "", fmt.Errorf("create temporary module: %w", err)
	}
	path := file.Name()

	if _, err := io.Copy(file, src); err != nil {
		file.Close()
		os.Remove(path)
		return "", fmt.Errorf("write temporary module: %w", err)
	}

	if err := file.Close(); err != nil {
		os.Remove(path)
		return "", fmt.Errorf("close temporary module: %w", err)
	}

	return path, nil
}

func writeTempModule(dir string, data []byte) (string, error) {
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

	if err := file.Close(); err != nil {
		os.Remove(path)
		return "", fmt.Errorf("close temporary module: %w", err)
	}

	return path, nil
}

func isSignedFile(path string) (bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return false, fmt.Errorf("read module: %w", err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return false, fmt.Errorf("stat module: %w", err)
	}

	start := int64(0)
	if info.Size() > signatureChunk {
		start = info.Size() - signatureChunk
	}

	data := make([]byte, info.Size()-start)
	if _, err := file.ReadAt(data, start); err != nil && err != io.EOF {
		return false, fmt.Errorf("read module: %w", err)
	}

	return bytes.Contains(data, []byte(signatureMarker)), nil
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
