package main

import (
	"fmt"

	"github.com/sk8thing/arch-skm-go/internal/config"
)

func loadConfig() (config.Config, error) {
	return config.Load()
}

func resolveBuildDir(cfg config.Config) (string, error) {
	if cfg.BuildDir == "" {
		return "", fmt.Errorf("build directory is required")
	}

	return cfg.BuildDir, nil
}
