package main

import (
	"fmt"
	"path/filepath"

	"github.com/sk8thing/arch-skm/internal/config"
	"github.com/sk8thing/arch-skm/internal/kernel"
)

type target struct {
	Variant  string
	Config   config.Config
	BuildDir string
	Dir      string
}

func resolveTarget(variant string) (target, error) {
	if _, err := kernel.RepoURL(variant); err != nil {
		return target{}, err
	}

	cfg, err := config.Load()
	if err != nil {
		return target{}, err
	}

	if cfg.BuildDir == "" {
		return target{}, fmt.Errorf("build directory is required")
	}

	return target{
		Variant:  variant,
		Config:   cfg,
		BuildDir: cfg.BuildDir,
		Dir:      filepath.Join(cfg.BuildDir, variant),
	}, nil
}
