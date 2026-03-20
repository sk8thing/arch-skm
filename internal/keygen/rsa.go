package keygen

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"fmt"
)

const (
	DefaultKeySize = 4096
)

type RSAKeyPair struct {
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
}

func (k *RSAKeyPair) PrivateKey() crypto.PrivateKey { return k.privateKey }
func (k *RSAKeyPair) PublicKey() crypto.PublicKey   { return k.publicKey }

func generateRSAKey() (*RSAKeyPair, error) {
	priv, err := rsa.GenerateKey(rand.Reader, DefaultKeySize)
	if err != nil {
		return nil, fmt.Errorf("generate rsa key: %w", err)
	}

	return &RSAKeyPair{
		privateKey: priv,
		publicKey:  &priv.PublicKey,
	}, nil
}
