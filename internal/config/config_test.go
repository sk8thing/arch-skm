package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func isolate(t *testing.T) (home, cfgDir string) {
	t.Helper()

	home = t.TempDir()
	cfgDir = t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", cfgDir)
	t.Setenv("BUILD_DIR", "")
	t.Setenv("MAKEPKG_DIR", "")

	return home, cfgDir
}

func writeConf(t *testing.T, cfgDir, content string) {
	t.Helper()

	dir := filepath.Join(cfgDir, "skm")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("mkdir config directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "skm.conf"), []byte(content), 0644); err != nil {
		t.Fatalf("write config file: %v", err)
	}
}

func TestLoadDefaults(t *testing.T) {
	home, _ := isolate(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if want := filepath.Join(home, "build"); cfg.BuildDir != want {
		t.Errorf("BuildDir = %q, want %q", cfg.BuildDir, want)
	}
	if want := "/tmp/makepkg"; cfg.MakepkgDir != want {
		t.Errorf("MakepkgDir = %q, want %q", cfg.MakepkgDir, want)
	}
}

func TestLoadDefaultsWithoutXDG(t *testing.T) {
	home, _ := isolate(t)
	t.Setenv("XDG_CONFIG_HOME", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if want := filepath.Join(home, "build"); cfg.BuildDir != want {
		t.Errorf("BuildDir = %q, want %q", cfg.BuildDir, want)
	}
}

func TestLoadConfigFromXDGFallbackDir(t *testing.T) {
	home, _ := isolate(t)
	t.Setenv("XDG_CONFIG_HOME", "")

	diverted := filepath.Join(home, "diverted")
	writeConf(t, filepath.Join(home, ".config"), "BUILD_DIR="+diverted+"\n")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.BuildDir != diverted {
		t.Errorf("BuildDir = %q, want %q", cfg.BuildDir, diverted)
	}
}

func TestLoadFromFile(t *testing.T) {
	home, cfgDir := isolate(t)
	writeConf(t, cfgDir, `
# comment with BUILD_DIR=ignored
BUILD_DIR="~/build-from-conf"
export MAKEPKG_DIR=/tmp/makepkg-conf
`)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if want := filepath.Join(home, "build-from-conf"); cfg.BuildDir != want {
		t.Errorf("BuildDir = %q, want %q", cfg.BuildDir, want)
	}
	if want := "/tmp/makepkg-conf"; cfg.MakepkgDir != want {
		t.Errorf("MakepkgDir = %q, want %q", cfg.MakepkgDir, want)
	}
}

func TestLoadTildeOnly(t *testing.T) {
	home, cfgDir := isolate(t)
	writeConf(t, cfgDir, "BUILD_DIR=~\n")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.BuildDir != home {
		t.Errorf("BuildDir = %q, want %q", cfg.BuildDir, home)
	}
}

func TestLoadRelativePath(t *testing.T) {
	_, cfgDir := isolate(t)
	writeConf(t, cfgDir, "BUILD_DIR=relbuild\n")

	cwd := t.TempDir()
	t.Chdir(cwd)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if want := filepath.Join(cwd, "relbuild"); cfg.BuildDir != want {
		t.Errorf("BuildDir = %q, want %q", cfg.BuildDir, want)
	}
}

func TestLoadEnvVariableExpansion(t *testing.T) {
	home, cfgDir := isolate(t)
	writeConf(t, cfgDir, "BUILD_DIR=${HOME}/expected\n")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if want := filepath.Join(home, "expected"); cfg.BuildDir != want {
		t.Errorf("BuildDir = %q, want %q", cfg.BuildDir, want)
	}
}

func TestLoadEnvOverridesFile(t *testing.T) {
	_, cfgDir := isolate(t)
	t.Setenv("BUILD_DIR", "/env/build")
	writeConf(t, cfgDir, "BUILD_DIR=/file/build\n")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.BuildDir != "/env/build" {
		t.Errorf("BuildDir = %q, want %q", cfg.BuildDir, "/env/build")
	}
}

func TestLoadEmptyValue(t *testing.T) {
	_, cfgDir := isolate(t)
	writeConf(t, cfgDir, "BUILD_DIR=\n")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() succeeded, want error")
	}
	if !strings.Contains(err.Error(), "BUILD_DIR is empty") {
		t.Errorf("Load() error = %q, want it to mention BUILD_DIR is empty", err)
	}
}

func TestLoadParseError(t *testing.T) {
	_, cfgDir := isolate(t)
	writeConf(t, cfgDir, "BUILD_DIR=/tmp/ok\nnot-a-key-value\n")

	if _, err := Load(); err == nil {
		t.Fatal("Load() succeeded, want error")
	}
}

func TestExpandHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"plain", "/absolute/path", "/absolute/path"},
		{"empty", "", ""},
		{"tilde only", "~", home},
		{"tilde slash", "~" + string(os.PathSeparator), home},
		{"tilde path", "~/build", filepath.Join(home, "build")},
		{"no tilde prefix", "usr~build", "usr~build"},
		{"tilde embedded", "~build", "~build"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := expandHome(tt.input)
			if err != nil {
				t.Fatalf("expandHome(%q) error: %v", tt.input, err)
			}
			if got != tt.want {
				t.Errorf("expandHome(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
