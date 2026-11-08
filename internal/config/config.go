package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

const (
	buildDirKey   = "BUILD_DIR"
	makepkgDirKey = "MAKEPKG_DIR"
)

type Config struct {
	BuildDir   string
	MakepkgDir string
}

func Load() (Config, error) {
	v := viper.New()
	v.SetConfigName("skm.conf")
	v.SetConfigType("env")
	if dir, err := os.UserConfigDir(); err == nil {
		v.AddConfigPath(filepath.Join(dir, "skm"))
	}
	v.SetDefault(buildDirKey, "~/build")
	v.SetDefault(makepkgDirKey, "/tmp/makepkg")
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		if _, ok := errors.AsType[viper.ConfigFileNotFoundError](err); !ok {
			return Config{}, fmt.Errorf("load config: %w", err)
		}
	}

	cfg := Config{
		BuildDir:   v.GetString(buildDirKey),
		MakepkgDir: v.GetString(makepkgDirKey),
	}

	var err error
	if cfg.BuildDir, err = absolutePath(buildDirKey, cfg.BuildDir); err != nil {
		return Config{}, err
	}
	if cfg.MakepkgDir, err = absolutePath(makepkgDirKey, cfg.MakepkgDir); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func absolutePath(key, path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("%s is empty", key)
	}

	value, err := expandHome(path)
	if err != nil {
		return "", fmt.Errorf("resolve %s %q: %w", key, path, err)
	}

	absPath, err := filepath.Abs(value)
	if err != nil {
		return "", fmt.Errorf("resolve %s %q: %w", key, value, err)
	}

	return absPath, nil
}

func expandHome(path string) (string, error) {
	if path != "~" && !strings.HasPrefix(path, "~/") {
		return path, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	if path == "~" {
		return home, nil
	}

	return filepath.Join(home, path[2:]), nil
}
