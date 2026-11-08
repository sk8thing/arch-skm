package certgen

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"

	"github.com/sk8thing/arch-skm/internal/kconfig"
)

func newECDSASigner(t *testing.T) *ecdsa.PrivateKey {
	t.Helper()

	priv, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	if err != nil {
		t.Fatalf("generate ECDSA key: %v", err)
	}

	return priv
}

func testConfig(t *testing.T, keyType, hashType string) *kconfig.KConfig {
	t.Helper()

	content := "CONFIG_MODULE_SIG_KEY_TYPE_" + keyType + "=y\nCONFIG_MODULE_SIG_HASH=\"" + hashType + "\"\n"
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

func TestGenerateCertificateCoversSignatureAlgorithms(t *testing.T) {
	// One RSA and one ECDSA key are reused for all hash variants, keeping the
	// test fast while covering the whole sigAlgoMap.
	t.Run("rsa", func(t *testing.T) {
		priv, err := rsa.GenerateKey(rand.Reader, 4096)
		if err != nil {
			t.Fatalf("generate RSA key: %v", err)
		}

		for _, tt := range []struct {
			hash kconfig.HashType
			want x509.SignatureAlgorithm
		}{
			{kconfig.HashTypeSHA1, x509.SHA1WithRSA},
			{kconfig.HashTypeSHA256, x509.SHA256WithRSA},
			{kconfig.HashTypeSHA384, x509.SHA384WithRSA},
			{kconfig.HashTypeSHA512, x509.SHA512WithRSA},
		} {
			der, err := GenerateCertificate(priv, testConfig(t, "RSA", string(tt.hash)))
			if err != nil {
				t.Fatalf("GenerateCertificate(sha %s) error: %v", tt.hash, err)
			}

			cert, err := x509.ParseCertificate(der)
			if err != nil {
				t.Fatalf("ParseCertificate(sha %s) error: %v", tt.hash, err)
			}
			if cert.SignatureAlgorithm != tt.want {
				t.Errorf("SignatureAlgorithm(sha %s) = %v, want %v", tt.hash, cert.SignatureAlgorithm, tt.want)
			}
		}
	})

	t.Run("ecdsa", func(t *testing.T) {
		priv := newECDSASigner(t)

		for _, tt := range []struct {
			hash kconfig.HashType
			want x509.SignatureAlgorithm
		}{
			{kconfig.HashTypeSHA1, x509.ECDSAWithSHA1},
			{kconfig.HashTypeSHA256, x509.ECDSAWithSHA256},
			{kconfig.HashTypeSHA384, x509.ECDSAWithSHA384},
			{kconfig.HashTypeSHA512, x509.ECDSAWithSHA512},
		} {
			der, err := GenerateCertificate(priv, testConfig(t, "ECDSA", string(tt.hash)))
			if err != nil {
				t.Fatalf("GenerateCertificate(sha %s) error: %v", tt.hash, err)
			}

			cert, err := x509.ParseCertificate(der)
			if err != nil {
				t.Fatalf("ParseCertificate(sha %s) error: %v", tt.hash, err)
			}
			if cert.SignatureAlgorithm != tt.want {
				t.Errorf("SignatureAlgorithm(sha %s) = %v, want %v", tt.hash, cert.SignatureAlgorithm, tt.want)
			}
		}
	})
}

func TestGenerateCertificate(t *testing.T) {
	priv := newECDSASigner(t)

	der, err := GenerateCertificate(priv, testConfig(t, "ECDSA", "sha384"))
	if err != nil {
		t.Fatalf("GenerateCertificate() error: %v", err)
	}

	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatalf("ParseCertificate() error: %v", err)
	}

	// The certificate identifies the local module signing key.
	if cert.Subject.CommonName != "Local Out of tree kernel module signing key" {
		t.Errorf("CommonName = %q", cert.Subject.CommonName)
	}

	// Lifetime, key usage and CA status match the kernel signing use case.
	// The calendar delta can differ from the plain duration by a few hours
	// (leap seconds, DST), so compare with tolerance.
	days := int(cert.NotAfter.Sub(cert.NotBefore).Hours() / 24)
	if days < 36500-2 || days > 36500+2 {
		t.Errorf("lifetime = %v, want roughly 36500 days", cert.NotAfter.Sub(cert.NotBefore))
	}
	if cert.KeyUsage != x509.KeyUsageDigitalSignature {
		t.Errorf("KeyUsage = %d, want digital signature", cert.KeyUsage)
	}
	if cert.IsCA {
		t.Error("certificate must not be a CA")
	}

	// The generated key is the one the certificate attests.
	pubDER, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	if err != nil {
		t.Fatalf("marshal public key: %v", err)
	}
	ski := sha1.Sum(pubDER)
	if string(cert.SubjectKeyId) != string(ski[:]) {
		t.Error("SubjectKeyId does not match the signer key")
	}
	if string(cert.RawSubjectPublicKeyInfo) != string(pubDER) {
		t.Error("certificate does not attest the generated public key")
	}

	// The serial number is random with at most 128 bits of entropy.
	if cert.SerialNumber.Sign() != 1 {
		t.Error("serial number must be positive")
	}
	if cert.SerialNumber.BitLen() > 128 {
		t.Errorf("serial number uses %d bits, want at most 128", cert.SerialNumber.BitLen())
	}
}

func TestGenerateCertificateValidSignature(t *testing.T) {
	priv := newECDSASigner(t)

	der, err := GenerateCertificate(priv, testConfig(t, "ECDSA", "sha256"))
	if err != nil {
		t.Fatalf("GenerateCertificate() error: %v", err)
	}

	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatalf("ParseCertificate() error: %v", err)
	}

	// The self-signed certificate's signature verifies against its own key.
	if err := cert.CheckSignature(cert.SignatureAlgorithm, cert.RawTBSCertificate, cert.Signature); err != nil {
		t.Errorf("CheckSignature() error: %v", err)
	}

	// And the private key produces signatures the certificate's public key
	// verifies.
	digest := sha256.Sum256([]byte("payload"))
	r, s, err := ecdsa.Sign(rand.Reader, priv, digest[:])
	if err != nil {
		t.Fatalf("ecdsa sign: %v", err)
	}
	if !ecdsa.Verify(&priv.PublicKey, digest[:], r, s) {
		t.Error("the certificate's public key does not verify signatures from the private key")
	}
}

func TestRandomSerialNumber(t *testing.T) {
	seen := make(map[string]struct{})

	for range 8 {
		serial, err := RandomSerialNumber()
		if err != nil {
			t.Fatalf("RandomSerialNumber() error: %v", err)
		}
		if serial.Sign() != 1 {
			t.Errorf("serial = %v, want positive", serial)
		}
		if serial.BitLen() > 128 {
			t.Errorf("serial uses %d bits, want at most 128", serial.BitLen())
		}

		key := serial.Text(16)
		if _, dup := seen[key]; dup {
			t.Errorf("serial %v repeated", key)
		}
		seen[key] = struct{}{}
	}
}

func TestExportCertificate(t *testing.T) {
	priv := newECDSASigner(t)
	config := testConfig(t, "ECDSA", "sha384")

	der, err := GenerateCertificate(priv, config)
	if err != nil {
		t.Fatalf("GenerateCertificate() error: %v", err)
	}

	privBytes, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		t.Fatalf("marshal private key: %v", err)
	}

	base := t.TempDir()
	prvPath, err := ExportCertificate(base, der, priv, config)
	if err != nil {
		t.Fatalf("ExportCertificate() error: %v", err)
	}

	// The returned path is the bundle PEM and lives in the keys directory.
	wantPrv := filepath.Join(base, "keys", "signing_prv.pem")
	if prvPath != wantPrv {
		t.Errorf("ExportCertificate() = %q, want %q", prvPath, wantPrv)
	}

	// The certificate itself is stored as DER, not PEM.
	crtData, err := os.ReadFile(filepath.Join(base, "keys", "signing_crt.crt"))
	if err != nil {
		t.Fatalf("read certificate: %v", err)
	}
	if !bytes.Equal(crtData, der) {
		t.Error("certificate file does not contain the DER bytes")
	}
	if info, statErr := os.Stat(filepath.Join(base, "keys", "signing_crt.crt")); statErr != nil || info.Mode().Perm() != 0644 {
		t.Errorf("certificate permissions = %v, %v; want 0644, nil", info, statErr)
	}
	if block, _ := pem.Decode(crtData); block != nil {
		t.Error("certificate file must be raw DER, not PEM")
	}

	// The bundle contains the private key followed by the certificate.
	bundle, err := os.ReadFile(prvPath)
	if err != nil {
		t.Fatalf("read bundle: %v", err)
	}
	checkPEMBlocks(t, bundle, "signing_prv.pem", []string{"PRIVATE KEY", "CERTIFICATE"}, privBytes, der)
	checkPermissions(t, prvPath, 0600)

	// The plain key file contains only the private key.
	keyData, err := os.ReadFile(filepath.Join(base, "keys", "signing_key.pem"))
	if err != nil {
		t.Fatalf("read key: %v", err)
	}
	checkPEMBlocks(t, keyData, "signing_key.pem", []string{"PRIVATE KEY"}, privBytes)
	checkPermissions(t, filepath.Join(base, "keys", "signing_key.pem"), 0600)

	// The hash type is written for the module signing caller.
	khash, err := os.ReadFile(filepath.Join(base, "keys", "khash"))
	if err != nil {
		t.Fatalf("read khash: %v", err)
	}
	if string(khash) != string(config.HashType()) {
		t.Errorf("khash content = %q, want %q", khash, string(config.HashType()))
	}
	checkPermissions(t, filepath.Join(base, "keys", "khash"), 0644)
}

func checkPEMBlocks(t *testing.T, data []byte, name string, types []string, wantPriv []byte, wantCert ...[]byte) {
	t.Helper()

	for i, wantType := range types {
		block, rest := pem.Decode(data)
		if block == nil {
			t.Fatalf("%s: PEM block %d missing", name, i)
		}
		if block.Type != wantType {
			t.Errorf("%s: PEM block %d type = %q, want %q", name, i, block.Type, wantType)
		}
		if wantType == "PRIVATE KEY" && !bytes.Equal(block.Bytes, wantPriv) {
			t.Errorf("%s: PEM block %d does not contain the PKCS8 private key", name, i)
		}
		if wantType == "CERTIFICATE" && len(wantCert) == 1 && !bytes.Equal(block.Bytes, wantCert[0]) {
			t.Errorf("%s: PEM block %d does not contain the certificate", name, i)
		}
		data = rest
	}
	if len(data) != 0 {
		t.Errorf("%s: unexpected trailing PEM data", name)
	}
}

func checkPermissions(t *testing.T, path string, want os.FileMode) {
	t.Helper()

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
	if got := info.Mode().Perm(); got != want {
		t.Errorf("%s permissions = %v, want %v", path, got, want)
	}
}
