package main

import (
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

	cfg, err := loadConfig()
	if err != nil {
		return target{}, err
	}

	buildDir, err := resolveBuildDir(cfg)
	if err != nil {
		return target{}, err
	}

	return target{
		Variant:  variant,
		Config:   cfg,
		BuildDir: buildDir,
		Dir:      filepath.Join(buildDir, variant),
	}, nil
}
