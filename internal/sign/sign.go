package sign

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sk8thing/arch-skm-go/internal/utils"
)

const (
	keysDirName     = "keys"
	signingKeyFile  = "signing_key.pem"
	signingCertFile = "signing_crt.crt"
	khashFile       = "khash"
	defaultHashType = "sha512"
)

func SignModules(paths []string) error {
	if len(paths) == 0 {
		return fmt.Errorf("no modules to sign")
	}

	signFile, keyFile, certFile, hashType, err := signingFiles()
	if err != nil {
		return err
	}

	for _, path := range paths {
		if err := signModule(signFile, keyFile, certFile, hashType, path); err != nil {
			return fmt.Errorf("sign module %q: %w", path, err)
		}
	}

	return nil
}

func signingFiles() (signFile string, keyFile string, certFile string, hashType string, err error) {
	exePath, err := utils.GetExecutablePath()
	if err != nil {
		return "", "", "", "", fmt.Errorf("get executable path: %w", err)
	}

	certDir := filepath.Dir(exePath)
	buildDir := filepath.Dir(certDir)
	keysDir := filepath.Join(certDir, keysDirName)

	signFile = filepath.Join(buildDir, "scripts", "sign-file")
	keyFile = filepath.Join(keysDir, signingKeyFile)
	certFile = filepath.Join(keysDir, signingCertFile)

	files := []struct {
		name string
		path string
	}{
		{"kernel signing tool", signFile},
		{"private key", keyFile},
		{"certificate", certFile},
	}

	for _, file := range files {
		if _, err := os.Stat(file.path); err != nil {
			return "", "", "", "", fmt.Errorf("%s %q: %w", file.name, file.path, err)
		}
	}

	hashType, err = readKhash(filepath.Join(keysDir, khashFile))
	if err != nil {
		return "", "", "", "", err
	}

	return signFile, keyFile, certFile, hashType, nil
}

func readKhash(path string) (string, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return defaultHashType, nil
	}
	if err != nil {
		return "", fmt.Errorf("read kernel hash %q: %w", path, err)
	}

	hashType := strings.TrimSpace(string(data))
	if hashType == "" {
		return defaultHashType, nil
	}

	return hashType, nil
}
