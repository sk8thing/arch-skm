package keygen

import (
	"crypto"
	"fmt"

	"github.com/sk8thing/arch-skm/internal/kconfig"
)

func GenerateKeyPair(config *kconfig.KConfig) (crypto.Signer, error) {
	switch config.KeyType() {
	case kconfig.KeyTypeRSA:
		return generateRSAKey()
	case kconfig.KeyTypeECDSA:
		return generateECDSAKey()
	default:
		return nil, fmt.Errorf("unsupported key type: %v", config.KeyType())
	}
}
