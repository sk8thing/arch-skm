package helper

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeExecutable(t *testing.T, dir, name string) string {
	t.Helper()

	return writeFile(t, dir, name, []byte("binary"), 0755)
}

func writeFile(t *testing.T, dir, name string, content []byte, mode os.FileMode) string {
	t.Helper()

	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, content, mode); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}

	return path
}

func TestVerifyExecutableFile(t *testing.T) {
	dir := t.TempDir()

	execFile := writeExecutable(t, dir, "exec")
	plainFile := writeFile(t, dir, "plain", []byte("text"), 0644)
	missing := filepath.Join(dir, "missing")

	if err := VerifyExecutableFile("runtime helper", execFile); err != nil {
		t.Errorf("VerifyExecutableFile(executable) error: %v", err)
	}

	err := VerifyExecutableFile("runtime helper", plainFile)
	if err == nil {
		t.Fatal("VerifyExecutableFile(plain file) succeeded, want error")
	}
	if !strings.Contains(err.Error(), "is not executable") {
		t.Errorf("error = %q, want it to mention the missing executable bit", err)
	}

	err = VerifyExecutableFile("runtime helper", dir)
	if err == nil {
		t.Fatal("VerifyExecutableFile(directory) succeeded, want error")
	}
	if !strings.Contains(err.Error(), "is not a regular file") {
		t.Errorf("error = %q, want it to mention the file type", err)
	}

	err = VerifyExecutableFile("runtime helper", missing)
	if err == nil {
		t.Fatal("VerifyExecutableFile(missing) succeeded, want error")
	}
	if !strings.Contains(err.Error(), "runtime helper") {
		t.Errorf("error = %q, want it to name the label", err)
	}
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("error = %q, want it to wrap fs.ErrNotExist", err)
	}
}

func TestVerifyRegularFile(t *testing.T) {
	dir := t.TempDir()

	// No executable bit required.
	plainFile := writeFile(t, dir, "plain", []byte("text"), 0644)
	if err := VerifyRegularFile("package", plainFile); err != nil {
		t.Errorf("VerifyRegularFile(plain file) error: %v", err)
	}

	if err := VerifyRegularFile("package", writeExecutable(t, dir, "exec")); err != nil {
		t.Errorf("VerifyRegularFile(executable) error: %v", err)
	}

	err := VerifyRegularFile("package", dir)
	if err == nil {
		t.Fatal("VerifyRegularFile(directory) succeeded, want error")
	}
	if !strings.Contains(err.Error(), "package") {
		t.Errorf("error = %q, want it to name the label", err)
	}

	if err := VerifyRegularFile("package", filepath.Join(dir, "missing")); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("VerifyRegularFile(missing) error = %q, want fs.ErrNotExist", err)
	}
}

func TestResolveExplicitPath(t *testing.T) {
	dir := t.TempDir()

	buf := t.TempDir()
	t.Setenv("SKM_HELPER", filepath.Join(buf, "should-not-be-used"))

	path := writeExecutable(t, dir, "skm")

	got, err := Resolve(path)
	if err != nil {
		t.Fatalf("Resolve() error: %v", err)
	}
	if want, err := filepath.Abs(path); err != nil {
		t.Fatalf("abs: %v", err)
	} else if got != want {
		t.Errorf("Resolve() = %q, want %q", got, want)
	}
}

func TestResolveExplicitPathErrors(t *testing.T) {
	t.Setenv("SKM_HELPER", "")

	if _, err := Resolve(filepath.Join(t.TempDir(), "missing")); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("Resolve(missing) error = %q, want fs.ErrNotExist", err)
	}
}

func TestResolveEnvVar(t *testing.T) {
	dir := t.TempDir()

	envPath := writeExecutable(t, dir, "env-skm")
	t.Setenv("SKM_HELPER", envPath)

	got, err := Resolve("")
	if err != nil {
		t.Fatalf("Resolve() error: %v", err)
	}
	if want, err := filepath.Abs(envPath); err != nil {
		t.Fatalf("abs: %v", err)
	} else if got != want {
		t.Errorf("Resolve() = %q, want %q", got, want)
	}
}

func TestResolveExplicitBeatsEnv(t *testing.T) {
	dir := t.TempDir()

	explicit := writeExecutable(t, dir, "explicit-skm")
	t.Setenv("SKM_HELPER", writeExecutable(t, dir, "env-skm"))

	got, err := Resolve(explicit)
	if err != nil {
		t.Fatalf("Resolve() error: %v", err)
	}
	if want, err := filepath.Abs(explicit); err != nil {
		t.Fatalf("abs: %v", err)
	} else if got != want {
		t.Errorf("Resolve() = %q, want the explicit path %q", got, want)
	}
}

func TestResolveEnvVarErrorsSilently(t *testing.T) {
	dir := t.TempDir()

	// An invalid SKM_HELPER is reported even though a sibling helper exists,
	// because explicit sources never fall through to the search list.
	writeExecutable(t, dir, "skm")

	t.Setenv("SKM_HELPER", filepath.Join(dir, "missing"))
	if _, err := Resolve(""); err == nil {
		t.Fatal("Resolve() succeeded with an invalid env helper, want error")
	}
}

func TestResolveSiblingCandidate(t *testing.T) {
	// The helper is first looked up as a sibling of the executable; the test
	// binary's directory plays that role temporarily.
	exePath, err := ExecutablePath()
	if err != nil {
		t.Fatalf("ExecutablePath() error: %v", err)
	}

	sibling := filepath.Join(filepath.Dir(exePath), "skm")
	t.Cleanup(func() {
		os.Remove(sibling)
	})
	writeExecutable(t, filepath.Dir(exePath), "skm")

	t.Setenv("SKM_HELPER", "")

	got, err := Resolve("")
	if err != nil {
		t.Fatalf("Resolve() error: %v", err)
	}
	if got != sibling {
		t.Errorf("Resolve() = %q, want sibling %q", got, sibling)
	}
}

func TestResolveNotFound(t *testing.T) {
	// Without a sibling helper and without /usr/lib/skm/skm installed, the
	// search must fail with a message that names the candidates.
	exePath, err := ExecutablePath()
	if err != nil {
		t.Fatalf("ExecutablePath() error: %v", err)
	}

	sibling := filepath.Join(filepath.Dir(exePath), "skm")
	t.Cleanup(func() {
		os.Remove(sibling)
	})

	t.Setenv("SKM_HELPER", "")

	installedStat, installedErr := os.Stat("/usr/lib/skm/skm")

	_, err = Resolve("")
	if err == nil {
		t.Skip("a helper exists in the search list, cannot test the failure path")
	}
	if !strings.Contains(err.Error(), "runtime helper skm not found") {
		t.Errorf("Resolve() error = %q, want it to mention the missing helper", err)
	}
	if installedErr != nil {
		if !strings.Contains(err.Error(), filepath.Dir(exePath)) {
			t.Errorf("Resolve() error = %q, want it to list the candidate directories", err)
		}
	} else {
		// /usr/lib/skm/skm exists but no sibling does: still an unexpected
		// situation for the test environment, keep the skip notice honest.
		t.Skipf("/usr/lib/skm/skm exists (%v); the search order test is not meaningful", installedStat)
	}
}

func TestExecutablePath(t *testing.T) {
	path, err := ExecutablePath()
	if err != nil {
		t.Fatalf("ExecutablePath() error: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat executable: %v", err)
	}
	if !info.Mode().IsRegular() || info.Mode().Perm()&0111 == 0 {
		t.Errorf("executable %s = %v, want a regular file with an execute bit", path, info.Mode())
	}
	if !filepath.IsAbs(path) {
		t.Errorf("ExecutablePath() = %q, want an absolute path", path)
	}
}
