package certgen

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/sk8thing/arch-skm-go/internal/kconfig"
	"github.com/sk8thing/arch-skm-go/internal/keygen"
)

const (
	certSuffix = "_crt.crt"
	prvSuffix  = "_prv.pem"
	keySuffix  = "_key.pem"
)

func ExportCertificate(path string, filename string, cert []byte, keys keygen.KeyPair, config *kconfig.KConfig) (trustedKeysPath string, err error) {
	timestamp := time.Now().Format("20060102-1504")
	dir := filepath.Join(path, timestamp)
	err = os.MkdirAll(dir, 0755)
	if err != nil {
		return "", err
	}
	crtPath := filepath.Join(dir, filename+certSuffix)
	prvPath := filepath.Join(dir, filename+prvSuffix)
	keyPath := filepath.Join(dir, filename+keySuffix)

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

	return keyPath, nil
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
