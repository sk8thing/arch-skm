package keygen

import (
	"crypto/mldsa"
	"fmt"
)

func generateMLDSAKey(params mldsa.Parameters) (*mldsa.PrivateKey, error) {
	priv, err := mldsa.GenerateKey(params)
	if err != nil {
		return nil, fmt.Errorf("generate %s key: %w", params, err)
	}

	return priv, nil
}
