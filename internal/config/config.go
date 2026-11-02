package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	DefaultMakepkgDir = "/tmp/makepkg"

	buildDirKey   = "BUILD_DIR"
	makepkgDirKey = "MAKEPKG_DIR"
)

type Config struct {
	BuildDir   string
	MakepkgDir string
	Path       string
}

func Load() (Config, error) {
	cfg, err := Default()
	if err != nil {
		return Config{}, err
	}

	path, err := find()
	if err != nil {
		return Config{}, err
	}
	if path == "" {
		return cfg, nil
	}

	values, err := parseFile(path)
	if err != nil {
		return Config{}, err
	}

	if value, ok := values[buildDirKey]; ok {
		cfg.BuildDir = value
	}
	if value, ok := values[makepkgDirKey]; ok {
		cfg.MakepkgDir = value
	}
	cfg.Path = path

	if cfg.BuildDir, err = filepath.Abs(cfg.BuildDir); err != nil {
		return Config{}, fmt.Errorf("resolve %s %q: %w", buildDirKey, cfg.BuildDir, err)
	}
	if cfg.MakepkgDir, err = filepath.Abs(cfg.MakepkgDir); err != nil {
		return Config{}, fmt.Errorf("resolve %s %q: %w", makepkgDirKey, cfg.MakepkgDir, err)
	}

	return cfg, nil
}

func Default() (Config, error) {
	home, err := configHome()
	if err != nil {
		return Config{}, err
	}

	buildDir, err := filepath.Abs(filepath.Join(home, "build"))
	if err != nil {
		return Config{}, fmt.Errorf("resolve default %s: %w", buildDirKey, err)
	}

	return Config{
		BuildDir:   buildDir,
		MakepkgDir: DefaultMakepkgDir,
	}, nil
}

func DefaultBuildDir() string {
	cfg, err := Default()
	if err != nil {
		return ""
	}

	return cfg.BuildDir
}

func configHome() (string, error) {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return xdg, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}

	return home, nil
}

func find() (string, error) {
	var candidates []string
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		candidates = append(candidates, filepath.Join(xdg, "skm", "config"))
	}

	home, err := os.UserHomeDir()
	if err != nil {
		if len(candidates) == 0 {
			return "", fmt.Errorf("resolve home directory: %w", err)
		}
	} else {
		candidates = append(candidates, filepath.Join(home, ".config", "skm", "config"))
	}

	seen := make(map[string]bool, len(candidates))
	for _, candidate := range candidates {
		candidate = filepath.Clean(candidate)
		if seen[candidate] {
			continue
		}
		seen[candidate] = true

		info, err := os.Stat(candidate)
		switch {
		case err == nil && info.Mode().IsRegular():
			return candidate, nil
		case err == nil:
			return "", fmt.Errorf("config path %q is not a regular file", candidate)
		case os.IsNotExist(err):
			continue
		default:
			return "", fmt.Errorf("access config %q: %w", candidate, err)
		}
	}

	return "", nil
}

func parseFile(path string) (map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open config %q: %w", path, err)
	}
	defer file.Close()

	values := make(map[string]string)
	scanner := bufio.NewScanner(file)
	lineNo := 0

	for scanner.Scan() {
		lineNo++

		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimSpace(strings.TrimPrefix(line, "export "))

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return nil, fmt.Errorf("parse %s:%d: expected KEY=VALUE", path, lineNo)
		}

		key = strings.TrimSpace(key)
		switch key {
		case buildDirKey, makepkgDirKey:
		default:
			continue
		}

		value = strings.TrimSpace(stripComment(value))
		value = unquote(value)
		value = os.ExpandEnv(value)
		if value == "" {
			return nil, fmt.Errorf("parse %s:%d: %s is empty", path, lineNo, key)
		}

		value, err = expandHome(value)
		if err != nil {
			return nil, fmt.Errorf("parse %s:%d: %w", path, lineNo, err)
		}

		values[key] = value
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read config %q: %w", path, err)
	}

	return values, nil
}

func stripComment(value string) string {
	var quote byte

	for i := 0; i < len(value); i++ {
		switch c := value[i]; {
		case quote != 0:
			if c == quote {
				quote = 0
			}
		case c == '\'' || c == '"':
			quote = c
		case c == '#' && (i == 0 || value[i-1] == ' ' || value[i-1] == '\t'):
			return value[:i]
		}
	}

	return value
}

func unquote(value string) string {
	if len(value) < 2 {
		return value
	}

	first, last := value[0], value[len(value)-1]
	if (first == '"' || first == '\'') && first == last {
		return value[1 : len(value)-1]
	}

	return value
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
