package keygen

import (
	"crypto"
	"crypto/mldsa"
	"fmt"

	"github.com/sk8thing/arch-skm/internal/kconfig"
)

func GenerateKeyPair(config *kconfig.KConfig) (crypto.Signer, error) {
	switch config.KeyType() {
	case kconfig.KeyTypeRSA:
		return generateRSAKey()
	case kconfig.KeyTypeECDSA:
		return generateECDSAKey()
	case kconfig.KeyTypeMLDSA44:
		return generateMLDSAKey(mldsa.MLDSA44())
	case kconfig.KeyTypeMLDSA65:
		return generateMLDSAKey(mldsa.MLDSA65())
	case kconfig.KeyTypeMLDSA87:
		return generateMLDSAKey(mldsa.MLDSA87())
	default:
		return nil, fmt.Errorf("unsupported key type: %v", config.KeyType())
	}
}
