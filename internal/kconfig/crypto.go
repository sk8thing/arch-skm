package kconfig

import (
	"strings"
)

type KeyType string
type HashType string

const (
	KeyTypeRSA   KeyType = "RSA"
	KeyTypeECDSA KeyType = "ECDSA"

	HashTypeSHA1   HashType = "sha1"
	HashTypeSHA256 HashType = "sha256"
	HashTypeSHA384 HashType = "sha384"
	HashTypeSHA512 HashType = "sha512"
)

var keyTypeMap = map[string]KeyType{
	string(KeyTypeRSA):   KeyTypeRSA,
	string(KeyTypeECDSA): KeyTypeECDSA,
}

var hashTypeMap = map[string]HashType{
	string(HashTypeSHA1):   HashTypeSHA1,
	string(HashTypeSHA256): HashTypeSHA256,
	string(HashTypeSHA384): HashTypeSHA384,
	string(HashTypeSHA512): HashTypeSHA512,
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
