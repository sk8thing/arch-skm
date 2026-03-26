package utils

import (
	"os"
	"path/filepath"
)

func GetExecutablePath() (string, error) {
	exePath, err := os.Executable()
	if err != nil {
		return "", err
	}

	realPath, err := filepath.EvalSymlinks(exePath)
	if err != nil {
		return "", err
	}

	return realPath, nil
}
