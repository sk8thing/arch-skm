package kconfig

import (
	"strings"
)

type KeyType string
type HashType string

const (
	KeyTypeRSA   KeyType = "RSA"
	KeyTypeECDSA KeyType = "ECDSA"

	KeyTypeMLDSA44 KeyType = "MLDSA_44"
	KeyTypeMLDSA65 KeyType = "MLDSA_65"
	KeyTypeMLDSA87 KeyType = "MLDSA_87"

	HashTypeSHA1    HashType = "sha1"
	HashTypeSHA256  HashType = "sha256"
	HashTypeSHA384  HashType = "sha384"
	HashTypeSHA512  HashType = "sha512"
	HashTypeSHA3256 HashType = "sha3-256"
	HashTypeSHA3384 HashType = "sha3-384"
	HashTypeSHA3512 HashType = "sha3-512"
)

var keyTypeMap = map[string]KeyType{
	string(KeyTypeRSA):     KeyTypeRSA,
	string(KeyTypeECDSA):   KeyTypeECDSA,
	string(KeyTypeMLDSA44): KeyTypeMLDSA44,
	string(KeyTypeMLDSA65): KeyTypeMLDSA65,
	string(KeyTypeMLDSA87): KeyTypeMLDSA87,

	// Early development aliases
	"ML_DSA_44": KeyTypeMLDSA44,
	"ML_DSA_65": KeyTypeMLDSA65,
	"ML_DSA_87": KeyTypeMLDSA87,
}

var hashTypeMap = map[string]HashType{
	string(HashTypeSHA1):    HashTypeSHA1,
	string(HashTypeSHA256):  HashTypeSHA256,
	string(HashTypeSHA384):  HashTypeSHA384,
	string(HashTypeSHA512):  HashTypeSHA512,
	string(HashTypeSHA3256): HashTypeSHA3256,
	string(HashTypeSHA3384): HashTypeSHA3384,
	string(HashTypeSHA3512): HashTypeSHA3512,
}

const (
	hashPrefix = "CONFIG_MODULE_SIG_HASH="
	keyPrefix  = "CONFIG_MODULE_SIG_KEY_TYPE_"
)

func parseHashType(line string) (HashType, bool) {
	value, found := strings.CutPrefix(line, hashPrefix)
	if !found {
		return "", false
	}

	sanitized := strings.Trim(strings.TrimSpace(value), `"`)
	hashType, found := hashTypeMap[sanitized]
	return hashType, found
}

func parseKeyType(line string) (KeyType, bool) {
	key, value, found := strings.Cut(line, "=")
	if !found || !strings.HasPrefix(line, keyPrefix) || value != "y" {
		return "", false
	}

	suffix := strings.TrimPrefix(key, keyPrefix)
	keyType, found := keyTypeMap[suffix]
	return keyType, found
}
