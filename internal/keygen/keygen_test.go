package keygen

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rsa"
	"os"
	"path/filepath"
	"testing"

	"github.com/sk8thing/arch-skm/internal/kconfig"
)

func newTestConfig(t *testing.T, keyType kconfig.KeyType) *kconfig.KConfig {
	t.Helper()

	content := "CONFIG_MODULE_SIG_KEY_TYPE_" + string(keyType) + "=y\nCONFIG_MODULE_SIG_HASH=\"sha256\"\n"
	path := filepath.Join(t.TempDir(), "config")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	config, err := kconfig.NewFromFile(path)
	if err != nil {
		t.Fatalf("NewFromFile() error: %v", err)
	}

	return config
}

func TestGenerateKeyPairRSA(t *testing.T) {
	signer, err := GenerateKeyPair(newTestConfig(t, kconfig.KeyTypeRSA))
	if err != nil {
		t.Fatalf("GenerateKeyPair() error: %v", err)
	}

	priv, ok := signer.(*rsa.PrivateKey)
	if !ok {
		t.Fatalf("GenerateKeyPair() returned %T, want *rsa.PrivateKey", signer)
	}
	if priv.N.BitLen() != DefaultKeySize {
		t.Errorf("RSA key size = %d, want %d", priv.N.BitLen(), DefaultKeySize)
	}
	if err := priv.Validate(); err != nil {
		t.Errorf("generated RSA key does not validate: %v", err)
	}

	// crypto.Signer contract: the public key is derivable from the signer.
	pub, ok := signer.Public().(*rsa.PublicKey)
	if !ok || pub.N.Cmp(priv.N) != 0 || pub.E != priv.E {
		t.Error("signer did not expose the private key's public half")
	}
}

func TestGenerateKeyPairECDSA(t *testing.T) {
	signer, err := GenerateKeyPair(newTestConfig(t, kconfig.KeyTypeECDSA))
	if err != nil {
		t.Fatalf("GenerateKeyPair() error: %v", err)
	}

	priv, ok := signer.(*ecdsa.PrivateKey)
	if !ok {
		t.Fatalf("GenerateKeyPair() returned %T, want *ecdsa.PrivateKey", signer)
	}
	if priv.Curve != elliptic.P384() {
		t.Errorf("ECDSA curve = %v, want P-384", priv.Curve)
	}

	pub, ok := signer.Public().(*ecdsa.PublicKey)
	if !ok || !pub.Equal(&priv.PublicKey) {
		t.Error("signer did not expose the private key's public half")
	}
}
