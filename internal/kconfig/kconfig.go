package kconfig

import (
	"bufio"
	"fmt"
	"os"
)

type KConfig struct {
	hashType HashType
	keyType  KeyType
}

func (k *KConfig) HashType() HashType { return k.hashType }
func (k *KConfig) KeyType() KeyType   { return k.keyType }

func NewFromFile(path string) (*KConfig, error) {
	k := &KConfig{}
	if err := k.parseCryptoParams(path); err != nil {
		return nil, err
	}

	return k, nil
}

func (k *KConfig) parseCryptoParams(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open kconfig %q: %w", path, err)
	}
	defer file.Close()

	var hashType HashType
	var keyType KeyType

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
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

	if err := scanner.Err(); err != nil {
		return err
	}

	if hashType == "" || keyType == "" {
		return fmt.Errorf("missing/unsupported hash or key: %q", path)
	}

	k.hashType = hashType
	k.keyType = keyType
	return nil
}
