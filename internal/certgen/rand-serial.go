package certgen

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

func RandomSerialNumber() (*big.Int, error) {
	serialBytes := make([]byte, 16)
	_, err := rand.Read(serialBytes)
	if err != nil {
		return nil, fmt.Errorf("generate random serial id: %w", err)
	}

	serial := new(big.Int).SetBytes(serialBytes)
	return serial, nil
}
