package keygen

import (
	"crypto/rand"
	"crypto/rsa"
	"fmt"
)

const (
	DefaultKeySize = 4096
)

func generateRSAKey() (*rsa.PrivateKey, error) {
	priv, err := rsa.GenerateKey(rand.Reader, DefaultKeySize)
	if err != nil {
		return nil, fmt.Errorf("generate rsa key: %w", err)
	}

	return priv, nil
}
