package certgen

import (
	"crypto/rand"
	"crypto/sha1"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"time"

	"github.com/sk8thing/arch-skm-go/internal/kconfig"
	"github.com/sk8thing/arch-skm-go/internal/keygen"
)

var sigAlgoMap = map[kconfig.KeyType]map[kconfig.HashType]x509.SignatureAlgorithm{
	kconfig.KeyTypeRSA: {
		kconfig.HashTypeSHA1:   x509.SHA1WithRSA,
		kconfig.HashTypeSHA256: x509.SHA256WithRSA,
		kconfig.HashTypeSHA384: x509.SHA384WithRSA,
		kconfig.HashTypeSHA512: x509.SHA512WithRSA,
	},
	kconfig.KeyTypeECDSA: {
		kconfig.HashTypeSHA1:   x509.ECDSAWithSHA1,
		kconfig.HashTypeSHA256: x509.ECDSAWithSHA256,
		kconfig.HashTypeSHA384: x509.ECDSAWithSHA384,
		kconfig.HashTypeSHA512: x509.ECDSAWithSHA512,
	},
}

func getSignatureAlgorithm(keyType kconfig.KeyType, hashType kconfig.HashType) (x509.SignatureAlgorithm, error) {
	algoMap, found := sigAlgoMap[keyType]
	if !found {
		return x509.UnknownSignatureAlgorithm, fmt.Errorf("unsupported key type: %v", keyType)
	}

	algo, found := algoMap[hashType]
	if !found {
		return x509.UnknownSignatureAlgorithm, fmt.Errorf("unsupported hash type %v for key type %v", hashType, keyType)
	}

	return algo, nil
}

func GenerateCertificate(keys keygen.KeyPair, config *kconfig.KConfig) ([]byte, error) {
	pub := keys.PublicKey()
	priv := keys.PrivateKey()
	algo, err := getSignatureAlgorithm(config.KeyType(), config.HashType())
	if err != nil {
		return nil, fmt.Errorf("invalid signature algorithm: %w", err)
	}

	pubDER, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		return nil, fmt.Errorf("marshal public key: %w", err)
	}

	ski := sha1.Sum(pubDER)
	serialNumber, err := RandomSerialNumber()
	if err != nil {
		return nil, err
	}
	now := time.Now()

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName: "Local Out of tree kernel module signing key",
		},
		NotBefore:             now,
		NotAfter:              now.AddDate(0, 0, 36500),
		SignatureAlgorithm:    algo,
		BasicConstraintsValid: true,
		IsCA:                  false,
		KeyUsage:              x509.KeyUsageDigitalSignature,
		SubjectKeyId:          ski[:],
	}

	der, err := x509.CreateCertificate(rand.Reader, &template, &template, pub, priv)
	if err != nil {
		return nil, fmt.Errorf("create certificate: %w", err)
	}

	return der, nil
}
