package install

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sk8thing/arch-skm/internal/helper"
)

func TestIsSameOrSubdir(t *testing.T) {
	parent := filepath.Join(string(os.PathSeparator), "parent")

	tests := []struct {
		name  string
		child string
		same  bool
	}{
		{"identical", parent, true},
		{"direct child", filepath.Join(parent, "child"), true},
		{"grand child", filepath.Join(parent, "child", "grand"), true},
		{"sibling", filepath.Join(string(os.PathSeparator), "sibling"), false},
		{"similar name", parent + "suffix", false},
		{"parent of parent", filepath.Join(parent, ".."), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isSameOrSubdir(parent, tt.child); got != tt.same {
				t.Errorf("isSameOrSubdir(%q, %q) = %v, want %v", parent, tt.child, got, tt.same)
			}
		})
	}
}

func TestCopyBinary(t *testing.T) {
	dir := t.TempDir()

	srcPath := filepath.Join(dir, "src")
	if err := os.WriteFile(srcPath, []byte("binary payload"), 0644); err != nil {
		t.Fatalf("write source binary: %v", err)
	}

	// Destination directory does not exist yet; it must be created.
	destPath := filepath.Join(dir, "nested", "bin", "skm")
	if err := copyBinary(srcPath, destPath); err != nil {
		t.Fatalf("copyBinary() error: %v", err)
	}

	got, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("read destination: %v", err)
	}
	if string(got) != "binary payload" {
		t.Errorf("copied binary content = %q, want %q", got, "binary payload")
	}
	if info, err := os.Stat(destPath); err != nil || info.Mode().Perm() != 0755 {
		t.Errorf("copied binary permissions = %v, %v; want 0755, nil", info, err)
	}

	// Copying a file onto itself is a no-op and does not require the source
	// to exist.
	samePath := filepath.Join(dir, "missing")
	if err := copyBinary(samePath, samePath); err != nil {
		t.Errorf("copyBinary(same path) error: %v", err)
	}

	// Missing sources are reported.
	if err := copyBinary(filepath.Join(dir, "nonexistent"), filepath.Join(dir, "dest")); err == nil {
		t.Error("copyBinary(missing source) succeeded, want error")
	}
}

func TestCopyDir(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "keys")
	dest := filepath.Join(dir, "keys-copy")

	if err := os.MkdirAll(filepath.Join(src, "nested"), 0700); err != nil {
		t.Fatalf("mkdir source tree: %v", err)
	}

	files := map[string]struct {
		content string
		mode    os.FileMode
	}{
		"signing_key.pem": {"private", 0600},
		"signing_crt.crt": {"cert", 0644},
		"nested/khash":    {"sha512", 0644},
	}
	for name, file := range files {
		path := filepath.Join(src, name)
		if err := os.WriteFile(path, []byte(file.content), file.mode); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	if err := copyDir(src, dest); err != nil {
		t.Fatalf("copyDir() error: %v", err)
	}

	for name, file := range files {
		data, err := os.ReadFile(filepath.Join(dest, name))
		if err != nil {
			t.Errorf("read copied %s: %v", name, err)
			continue
		}
		if string(data) != file.content {
			t.Errorf("copied %s content = %q, want %q", name, data, file.content)
		}
		if info, err := os.Stat(filepath.Join(dest, name)); err != nil || info.Mode().Perm() != file.mode {
			t.Errorf("copied %s permissions = %v, %v; want %v, nil", name, info, err, file.mode)
		}
	}
	if info, err := os.Stat(filepath.Join(dest, "nested")); err != nil || info.Mode().Perm() != 0700 {
		t.Errorf("copied directory permissions = %v, %v; want 0700, nil", info, err)
	}
}

func TestCopyDirRejectsNonRegularFiles(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "keys")
	dest := filepath.Join(dir, "keys-copy")

	if err := os.MkdirAll(src, 0755); err != nil {
		t.Fatalf("mkdir source tree: %v", err)
	}
	if err := os.WriteFile(filepath.Join(src, "key.pem"), []byte("private"), 0644); err != nil {
		t.Fatalf("write key: %v", err)
	}
	target := filepath.Join(dir, "target")
	if err := os.WriteFile(target, []byte("target"), 0644); err != nil {
		t.Fatalf("write target: %v", err)
	}
	if err := os.Symlink(target, filepath.Join(src, "link")); err != nil {
		t.Fatalf("create symlink: %v", err)
	}

	err := copyDir(src, dest)
	if err == nil {
		t.Fatal("copyDir() succeeded, want error")
	}
	if !strings.Contains(err.Error(), "unsupported file in key directory") {
		t.Errorf("copyDir() error = %q, want it to mention unsupported files", err)
	}
}

func TestPrepareBuildDir(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "build-tree")

	source := filepath.Join(dir, "skm")
	if err := os.WriteFile(source, []byte("helper binary"), 0644); err != nil {
		t.Fatalf("write source binary: %v", err)
	}

	if err := PrepareBuildDir(dest, source); err != nil {
		t.Fatalf("PrepareBuildDir() error: %v", err)
	}

	certsLocal := filepath.Join(dest, "certs-local")
	if info, err := os.Stat(filepath.Join(certsLocal, "skm")); err != nil || info.Mode().Perm() != 0755 {
		t.Errorf("copied skm permissions = %v, %v; want 0755, nil", info, err)
	}

	script := filepath.Join(certsLocal, "dkms", "kernel-sign.sh")
	if info, err := os.Stat(script); err != nil || info.Mode().Perm() != 0755 {
		t.Errorf("kernel-sign.sh permissions = %v, %v; want 0755, nil", info, err)
	}

	conf := filepath.Join(certsLocal, "dkms", "kernel-sign.conf")
	if info, err := os.Stat(conf); err != nil || info.Mode().Perm() != 0644 {
		t.Errorf("kernel-sign.conf permissions = %v, %v; want 0644, nil", info, err)
	}
	if data, err := os.ReadFile(conf); err != nil || !strings.Contains(string(data), "POST_BUILD=") {
		t.Errorf("kernel-sign.conf content = %q, %v; want POST_BUILD= hook", data, err)
	}
}

func TestPrepareBuildDirRequiresSkmPath(t *testing.T) {
	dir := t.TempDir()

	if err := PrepareBuildDir(dir, ""); err == nil {
		t.Fatal("PrepareBuildDir() succeeded, want error")
	} else if !strings.Contains(err.Error(), "skm binary path is required") {
		t.Errorf("PrepareBuildDir() error = %q, want it to mention the missing path", err)
	}
}

func TestPrepareBuildDirMissingBinary(t *testing.T) {
	if err := PrepareBuildDir(t.TempDir(), filepath.Join(t.TempDir(), "nonexistent")); err == nil {
		t.Fatal("PrepareBuildDir() succeeded, want error")
	}
}

func TestInstallCertificates(t *testing.T) {
	exePath, err := helper.ExecutablePath()
	if err != nil {
		t.Fatalf("ExecutablePath() error: %v", err)
	}
	keysDir := filepath.Join(filepath.Dir(exePath), "keys")
	certs := map[string]string{
		"signing_key.pem": "private",
		"signing_crt.crt": "cert",
		"signing_prv.pem": "bundle",
		"khash":           "sha512",
	}

	// The command in production is a sibling of the keys directory. The test
	// binary's directory plays that role temporarily.
	if err := os.MkdirAll(keysDir, 0755); err != nil {
		t.Fatalf("create keys directory: %v", err)
	}
	for name, content := range certs {
		if err := os.WriteFile(filepath.Join(keysDir, name), []byte(content), 0644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	if err := os.Mkdir(filepath.Join(keysDir, "nested"), 0755); err != nil {
		t.Fatalf("mkdir nested directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(keysDir, "nested", "extra.pem"), []byte("extra"), 0644); err != nil {
		t.Fatalf("write nested key: %v", err)
	}
	t.Cleanup(func() {
		os.RemoveAll(keysDir)
	})

	dest := filepath.Join(t.TempDir(), "certs")
	if err := InstallCertificates(dest); err != nil {
		t.Fatalf("InstallCertificates() error: %v", err)
	}

	for name, content := range certs {
		data, err := os.ReadFile(filepath.Join(dest, "keys", name))
		if err != nil {
			t.Errorf("read copied %s: %v", name, err)
			continue
		}
		if string(data) != content {
			t.Errorf("copied %s content = %q, want %q", name, data, content)
		}
	}

	skm := filepath.Join(dest, "skm")
	info, err := os.Stat(skm)
	if err != nil {
		t.Fatalf("copied skm missing: %v", err)
	}
	if !info.Mode().IsRegular() || info.Mode().Perm()&0111 == 0 {
		t.Errorf("copied skm = %v, want an executable regular file", info.Mode())
	}
}

func TestInstallCertificatesRequiresDistinctDestination(t *testing.T) {
	exePath, err := helper.ExecutablePath()
	if err != nil {
		t.Fatalf("ExecutablePath() error: %v", err)
	}
	keysDir := filepath.Join(filepath.Dir(exePath), "keys")
	t.Cleanup(func() {
		os.RemoveAll(keysDir)
	})

	// Installing into the keys directory itself is refused, even (and
	// especially) when the directory does not exist.
	if err := InstallCertificates(keysDir); err == nil {
		t.Fatal("InstallCertificates(source) succeeded, want error")
	}
}

func TestInstallCertificatesMissingKeys(t *testing.T) {
	// No keys directory: the copy fails instead of silently producing an
	// empty destination.
	if err := InstallCertificates(filepath.Join(t.TempDir(), "certs")); err == nil {
		t.Fatal("InstallCertificates() succeeded, want error")
	}
}
