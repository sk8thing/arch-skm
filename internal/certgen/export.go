package certgen

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sk8thing/arch-skm-go/internal/kconfig"
	"github.com/sk8thing/arch-skm-go/internal/keygen"
)

const (
	prefix     = "signing"
	certSuffix = "_crt.crt"
	prvSuffix  = "_prv.pem"
	keySuffix  = "_key.pem"
)

func ExportCertificate(path string, cert []byte, keys keygen.KeyPair, config *kconfig.KConfig) (trustedKeysPath string, err error) {
	dir := filepath.Join(path, "keys")
	err = os.MkdirAll(dir, 0755)
	if err != nil {
		return "", err
	}
	crtPath := filepath.Join(dir, prefix+certSuffix)
	prvPath := filepath.Join(dir, prefix+prvSuffix)
	keyPath := filepath.Join(dir, prefix+keySuffix)

	privBytes, err := x509.MarshalPKCS8PrivateKey(keys.PrivateKey())
	if err != nil {
		return "", err
	}

	err = writeCertificateDER(crtPath, cert)
	if err != nil {
		return "", err
	}

	err = writeKeyBundlePEM(prvPath, privBytes, cert)
	if err != nil {
		return "", err
	}

	err = writePrivateKeyPEM(keyPath, privBytes)
	if err != nil {
		return "", err
	}

	khashPath := filepath.Join(dir, "khash")
	if err := os.WriteFile(khashPath, []byte(config.HashType()), 0644); err != nil {
		return "", fmt.Errorf("write khash file: %w", err)
	}

	return prvPath, nil
}

func writeCertificateDER(filename string, cert []byte) error {
	if err := os.WriteFile(filename, cert, 0644); err != nil {
		return fmt.Errorf("write certificate DER: %w", err)
	}

	return nil
}

func writePrivateKeyPEM(filename string, priv []byte) error {
	return writePEMFile(filename, 0600, []*pem.Block{
		{
			Type:  "PRIVATE KEY",
			Bytes: priv,
		},
	})
}

func writeKeyBundlePEM(filename string, priv []byte, cert []byte) error {
	return writePEMFile(filename, 0600, []*pem.Block{
		{
			Type:  "PRIVATE KEY",
			Bytes: priv,
		},
		{
			Type:  "CERTIFICATE",
			Bytes: cert,
		},
	})

}

func writePEMFile(filename string, perm os.FileMode, blocks []*pem.Block) error {
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	defer file.Close()

	for _, block := range blocks {
		if err := pem.Encode(file, block); err != nil {
			return fmt.Errorf("encode PEM block: %w", err)
		}
	}

	return nil
}
