package certgen

import (
	"crypto"
	"crypto/rand"
	"crypto/sha1"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"math/big"
	"time"

	"github.com/sk8thing/arch-skm/internal/kconfig"
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

var mldsaSigAlgoMap = map[kconfig.KeyType]x509.SignatureAlgorithm{
	kconfig.KeyTypeMLDSA44: x509.MLDSA44,
	kconfig.KeyTypeMLDSA65: x509.MLDSA65,
	kconfig.KeyTypeMLDSA87: x509.MLDSA87,
}

func getSignatureAlgorithm(keyType kconfig.KeyType, hashType kconfig.HashType) (x509.SignatureAlgorithm, error) {
	if algo, found := mldsaSigAlgoMap[keyType]; found {
		return algo, nil
	}

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

func GenerateCertificate(signer crypto.Signer, config *kconfig.KConfig) ([]byte, error) {
	pub := signer.Public()
	algo, err := getSignatureAlgorithm(config.KeyType(), config.HashType())
	if err != nil {
		return nil, fmt.Errorf("invalid signature algorithm: %w", err)
	}

	pubDER, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		return nil, fmt.Errorf("marshal public key: %w", err)
	}

	ski := sha1.Sum(pubDER)
	serialNumber, err := genRandSerial()
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

	der, err := x509.CreateCertificate(rand.Reader, &template, &template, pub, signer)
	if err != nil {
		return nil, fmt.Errorf("create certificate: %w", err)
	}

	return der, nil
}

func genRandSerial() (*big.Int, error) {
	serialBytes := make([]byte, 16)
	_, err := rand.Read(serialBytes)
	if err != nil {
		return nil, fmt.Errorf("generate random serial id: %w", err)
	}

	serial := new(big.Int).SetBytes(serialBytes)
	return serial, nil
}
