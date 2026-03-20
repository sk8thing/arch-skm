package keygen

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"fmt"
)

type ECDSAKeyPair struct {
	privateKey *ecdsa.PrivateKey
	publicKey  *ecdsa.PublicKey
}

func (k *ECDSAKeyPair) PrivateKey() crypto.PrivateKey { return k.privateKey }
func (k *ECDSAKeyPair) PublicKey() crypto.PublicKey   { return k.publicKey }

func generateECDSAKey() (*ECDSAKeyPair, error) {
	priv, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate ecdsa key: %w", err)
	}

	return &ECDSAKeyPair{
		privateKey: priv,
		publicKey:  &priv.PublicKey,
	}, nil
}
