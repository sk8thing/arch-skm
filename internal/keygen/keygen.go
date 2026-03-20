package keygen

import (
	"crypto"
	"fmt"

	"github.com/sk8thing/arch-skm-go/internal/kconfig"
)

type KeyPair interface {
	PrivateKey() crypto.PrivateKey
	PublicKey() crypto.PublicKey
}

func GenerateKeyPair(config *kconfig.KConfig) (KeyPair, error) {
	keyType := config.KeyType()
	switch keyType {
	case kconfig.KeyTypeRSA:
		return generateRSAKey()
	case kconfig.KeyTypeECDSA:
		return generateECDSAKey()
	default:
		return nil, fmt.Errorf("unsupported key type: %v", keyType)
	}
}
