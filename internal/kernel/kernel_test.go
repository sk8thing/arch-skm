package kernel

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSupported(t *testing.T) {
	want := []string{
		"linux",
		"linux-hardened",
		"linux-lts",
		"linux-zen",
		"linux-rt",
		"linux-libre",
		"linux-xanmod",
	}

	got := Supported()
	if len(got) != len(want) {
		t.Fatalf("Supported() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("Supported()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestRepoURL(t *testing.T) {
	tests := map[string]string{
		"linux":          "https://gitlab.archlinux.org/archlinux/packaging/packages/linux.git",
		"linux-hardened": "https://gitlab.archlinux.org/archlinux/packaging/packages/linux-hardened.git",
		"linux-lts":      "https://gitlab.archlinux.org/archlinux/packaging/packages/linux-lts.git",
		"linux-zen":      "https://gitlab.archlinux.org/archlinux/packaging/packages/linux-zen.git",
		"linux-rt":       "https://gitlab.archlinux.org/archlinux/packaging/packages/linux-rt.git",
		"linux-libre":    "https://aur.archlinux.org/linux-libre.git",
		"linux-xanmod":   "https://aur.archlinux.org/linux-xanmod.git",
	}

	for variant, want := range tests {
		got, err := RepoURL(variant)
		if err != nil {
			t.Errorf("RepoURL(%q) error: %v", variant, err)
			continue
		}
		if got != want {
			t.Errorf("RepoURL(%q) = %q, want %q", variant, got, want)
		}
	}
}

func TestRepoURLUnknown(t *testing.T) {
	_, err := RepoURL("linux-noinstallthis")
	if err == nil {
		t.Fatal("RepoURL(unknown) succeeded, want error")
	}
	if !strings.Contains(err.Error(), "unsupported kernel") {
		t.Errorf("RepoURL() error = %q, want it to mention the unsupported kernel", err)
	}
	if !strings.Contains(err.Error(), "linux-hardened") {
		t.Errorf("RepoURL() error = %q, want it to list supported kernels", err)
	}
}

func TestUpdateRepoUnknownKernel(t *testing.T) {
	if _, err := UpdateRepo(t.Context(), UpdateOptions{Kernel: "linux-nope", BuildDir: t.TempDir()}); err == nil {
		t.Fatal("UpdateRepo(unknown kernel) succeeded, want error")
	}
}

func TestUpdateRepoRequiresBuildDir(t *testing.T) {
	_, err := UpdateRepo(t.Context(), UpdateOptions{Kernel: "linux", BuildDir: ""})
	if err == nil {
		t.Fatal("UpdateRepo(empty build dir) succeeded, want error")
	}
	if !strings.Contains(err.Error(), "build directory is required") {
		t.Errorf("UpdateRepo() error = %q, want it to mention the required build directory", err)
	}
}

func TestUpdateRepoRejectsExistingNonGitDirectory(t *testing.T) {
	buildDir := t.TempDir()
	dest := filepath.Join(buildDir, "linux")
	if err := os.Mkdir(dest, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dest, "PKGBUILD.txt"), []byte("pivot"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	_, err := UpdateRepo(t.Context(), UpdateOptions{Kernel: "linux", BuildDir: buildDir})
	if err == nil {
		t.Fatal("UpdateRepo(non-git directory) succeeded, want error")
	}
	for _, part := range []string{"not a git clone", "--force"} {
		if !strings.Contains(err.Error(), part) {
			t.Errorf("UpdateRepo() error = %q, want it to mention %q", err, part)
		}
	}
}

func TestUpdateRepoRejectsFileAsBuildPath(t *testing.T) {
	buildDir := t.TempDir()
	dest := filepath.Join(buildDir, "linux")
	if err := os.WriteFile(dest, []byte("clobber"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	_, err := UpdateRepo(t.Context(), UpdateOptions{Kernel: "linux", BuildDir: buildDir})
	if err == nil {
		t.Fatal("UpdateRepo(file at build path) succeeded, want error")
	}
	if !strings.Contains(err.Error(), "is not a directory") {
		t.Errorf("UpdateRepo() error = %q, want it to mention the file type", err)
	}
}
