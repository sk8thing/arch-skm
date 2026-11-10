package keygen

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/mldsa"
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

func TestGenerateKeyPairMLDSA(t *testing.T) {
	tests := []struct {
		name     string
		keyType  kconfig.KeyType
		wantName string
		wantPub  int
	}{
		{"mldsa-44", kconfig.KeyTypeMLDSA44, "ML-DSA-44", mldsa.MLDSA44PublicKeySize},
		{"mldsa-65", kconfig.KeyTypeMLDSA65, "ML-DSA-65", mldsa.MLDSA65PublicKeySize},
		{"mldsa-87", kconfig.KeyTypeMLDSA87, "ML-DSA-87", mldsa.MLDSA87PublicKeySize},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			signer, err := GenerateKeyPair(newTestConfig(t, tt.keyType))
			if err != nil {
				t.Fatalf("GenerateKeyPair() error: %v", err)
			}

			priv, ok := signer.(*mldsa.PrivateKey)
			if !ok {
				t.Fatalf("GenerateKeyPair() returned %T, want *mldsa.PrivateKey", signer)
			}

			pub := priv.PublicKey()
			if pub.Parameters().String() != tt.wantName {
				t.Errorf("parameters = %v, want %v", pub.Parameters(), tt.wantName)
			}
			if len(pub.Bytes()) != tt.wantPub {
				t.Errorf("public key size = %d, want %d", len(pub.Bytes()), tt.wantPub)
			}
			if !pub.Equal(priv.Public().(*mldsa.PublicKey)) {
				t.Error("signer did not expose the private key's public half")
			}

			// The key must round-trip through its seed encoding and produce
			// verifiable signatures.
			decoded, err := mldsa.NewPrivateKey(priv.PublicKey().Parameters(), priv.Bytes())
			if err != nil {
				t.Fatalf("NewPrivateKey() error: %v", err)
			}
			if !decoded.Equal(priv) {
				t.Error("seed encoding does not round-trip to the same private key")
			}

			message := []byte("kernel module payload")
			signature, err := priv.Sign(nil, message, nil)
			if err != nil {
				t.Fatalf("Sign() error: %v", err)
			}
			if err := mldsa.Verify(pub, message, signature, nil); err != nil {
				t.Errorf("Verify() error: %v", err)
			}
		})
	}
}
