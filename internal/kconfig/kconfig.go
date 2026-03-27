package kconfig

import (
	"bufio"
	"fmt"
	"os"
)

type KConfig struct {
	path     string
	hashType HashType
	keyType  KeyType
}

func (k *KConfig) HashType() HashType { return k.hashType }
func (k *KConfig) KeyType() KeyType   { return k.keyType }

func NewFromFile(path string) (*KConfig, error) {
	k := &KConfig{path: path}
	if err := k.parseCryptoParams(); err != nil {
		return nil, err
	}

	return k, nil
}

func (k *KConfig) UpdateTrustedKeys(value string) error {
	lines, err := k.readLines()
	if err != nil {
		return err
	}

	found := false
	for i, line := range lines {
		if updated, ok := updateTrustedKeys(line, value); ok {
			lines[i] = updated
			found = true
		}
	}

	if !found {
		lines = append(lines, trustedKeys+fmt.Sprintf(`"%s"`, value))
	}

	tmpPath := k.path + ".tmp"
	out, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("write temp kconfig %q: %w", tmpPath, err)
	}
	defer out.Close()

	writer := bufio.NewWriter(out)
	for _, l := range lines {
		if _, err := writer.WriteString(l + "\n"); err != nil {
			return err
		}
	}

	if err := writer.Flush(); err != nil {
		return err
	}

	if err := os.Rename(tmpPath, k.path); err != nil {
		return fmt.Errorf("overwrite kconfig %q: %w", k.path, err)
	}

	return nil
}

func (k *KConfig) parseCryptoParams() error {
	lines, err := k.readLines()
	if err != nil {
		return err
	}

	var hashType HashType
	var keyType KeyType

	for _, line := range lines {
		if hash, found := parseHashType(line); found {
			hashType = hash
		}
		if key, found := parseKeyType(line); found {
			keyType = key
		}
		if keyType != "" && hashType != "" {
			break
		}
	}

	if hashType == "" || keyType == "" {
		return fmt.Errorf("missing/unsupported hash or key: %q", k.path)
	}

	k.hashType = hashType
	k.keyType = keyType
	return nil
}

func (k *KConfig) readLines() ([]string, error) {
	file, err := os.Open(k.path)
	if err != nil {
		return nil, fmt.Errorf("open kconfig %q: %w", k.path, err)
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return lines, nil
}
