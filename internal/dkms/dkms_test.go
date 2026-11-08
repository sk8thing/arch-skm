package dkms

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseStatusEntries(t *testing.T) {
	output := `
nvidia/535.129.03: installed
nvidia/535.129.03, 6.7.0-arch1-1, x86_64: installed
vboxhost/7.0.8: installed
deprecated/1.0: removed
zstd: not installed
broken: added
empty-always-skipped:
no colon at all
`
	entries := parseStatusEntries(output)

	want := []statusEntry{
		{Name: "nvidia", Version: "535.129.03"},
		{Name: "nvidia", Version: "535.129.03", Kernel: "6.7.0-arch1-1"},
		{Name: "vboxhost", Version: "7.0.8"},
	}

	if len(entries) != len(want) {
		t.Fatalf("parseStatusEntries() = %+v, want %+v", entries, want)
	}
	for i := range want {
		if entries[i] != want[i] {
			t.Errorf("parseStatusEntries()[%d] = %+v, want %+v", i, entries[i], want[i])
		}
	}
}

func TestParseStatusEntriesAcceptsCaseVariants(t *testing.T) {
	entries := parseStatusEntries("nvidia/535: Installed\nzstd: INSTALLED\n")

	want := []statusEntry{
		{Name: "nvidia", Version: "535"},
		{Name: "zstd"},
	}
	if !equalEntries(entries, want) {
		t.Fatalf("parseStatusEntries() = %+v, want %+v", entries, want)
	}
}

func TestParseStatusEntriesEmpty(t *testing.T) {
	if entries := parseStatusEntries(""); len(entries) != 0 {
		t.Errorf("parseStatusEntries(empty) = %+v, want no entries", entries)
	}
}

func equalEntries(got, want []statusEntry) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range want {
		if got[i] != want[i] {
			return false
		}
	}

	return true
}

func TestStatusEntrySpec(t *testing.T) {
	tests := []struct {
		entry statusEntry
		want  string
	}{
		{statusEntry{Name: "nvidia", Version: "535.129.03"}, "nvidia/535.129.03"},
		{statusEntry{Name: "vboxhost"}, "vboxhost"},
	}

	for _, tt := range tests {
		if got := tt.entry.Spec(); got != tt.want {
			t.Errorf("Spec() = %q, want %q", got, tt.want)
		}
	}
}

func TestStatusNames(t *testing.T) {
	entries := []statusEntry{
		{Name: "nvidia"},
		{Name: "nvidia"},
		{Name: "vboxhost"},
		{Name: "nvidia"},
	}

	got := statusNames(entries)
	want := []string{"nvidia", "vboxhost"}

	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("statusNames() = %v, want %v", got, want)
	}
}

func TestModuleSpecsPrefersCurrentKernel(t *testing.T) {
	entries := []statusEntry{
		{Name: "nvidia", Version: "535.129.03", Kernel: "6.6.7-arch1-1"},
		{Name: "nvidia", Version: "535.129.03", Kernel: "6.7.0-arch1-1"},
		{Name: "nvidia", Version: "535.129.03", Kernel: "6.7.0-arch1-1"},
		{Name: "nvidia", Version: "536.20.02", Kernel: "6.6.7-arch1-1"},
		{Name: "vboxhost", Version: "7.0.8", Kernel: "6.7.0-arch1-1"},
	}

	got := moduleSpecs(entries, "nvidia", "6.7.0-arch1-1")
	if len(got) != 1 || got[0] != "nvidia/535.129.03" {
		t.Errorf("moduleSpecs(current) = %v, want [nvidia/535.129.03]", got)
	}
}

func TestModuleSpecsFallsBackToAll(t *testing.T) {
	entries := []statusEntry{
		{Name: "nvidia", Version: "535.129.03", Kernel: "6.6.7-arch1-1"},
		{Name: "nvidia", Version: "535.129.03", Kernel: "6.7.0-arch1-1"},
		{Name: "nvidia", Version: "536.20.02", Kernel: "6.6.7-arch1-1"},
		{Name: "vboxhost", Version: "7.0.8", Kernel: "6.7.0-arch1-1"},
	}

	got := moduleSpecs(entries, "nvidia", "9.9.0-arch1-1")
	want := []string{"nvidia/535.129.03", "nvidia/536.20.02"}
	if len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("moduleSpecs(no current kernel) = %v, want %v", got, want)
	}
}

func TestModuleSpecsUnknownModule(t *testing.T) {
	if got := moduleSpecs([]statusEntry{{Name: "nvidia"}}, "zstd", "6.7.0-arch1-1"); len(got) != 0 {
		t.Errorf("moduleSpecs(unknown) = %v, want no specs", got)
	}
}

func TestValidateName(t *testing.T) {
	tests := []struct {
		name    string
		wantErr string
	}{
		{name: "", wantErr: "module name is empty"},
		{name: ".", wantErr: "invalid module name"},
		{name: "..", wantErr: "invalid module name"},
		{name: "foo/bar", wantErr: "invalid module name"},
		{name: `foo\bar`, wantErr: "invalid module name"},
		{name: "nvidia", wantErr: ""},
		{name: "virtualbox-host-dkms", wantErr: ""},
	}

	for _, tt := range tests {
		err := validateName(tt.name)
		if tt.wantErr == "" {
			if err != nil {
				t.Errorf("validateName(%q) error: %v", tt.name, err)
			}
			continue
		}
		if err == nil {
			t.Errorf("validateName(%q) succeeded, want error", tt.name)
		} else if !strings.Contains(err.Error(), tt.wantErr) {
			t.Errorf("validateName(%q) error = %q, want it to mention %q", tt.name, err, tt.wantErr)
		}
	}
}

func TestValidateModule(t *testing.T) {
	installed := map[string]struct{}{"nvidia": {}}

	if err := validateModule("nvidia", installed); err != nil {
		t.Errorf("validateModule(installed) error: %v", err)
	}

	err := validateModule("zstd", installed)
	if err == nil {
		t.Fatal("validateModule(not installed) succeeded, want error")
	}
	if !strings.Contains(err.Error(), "not installed") {
		t.Errorf("validateModule() error = %q, want it to mention not installed", err)
	}

	err = validateModule("../escape", installed)
	if err == nil {
		t.Fatal("validateModule(path-like) succeeded, want error")
	}
	if strings.Contains(err.Error(), "not installed") {
		t.Errorf("validateModule(path-like) error = %q, want the name validation message", err)
	}
}

func TestModuleConf(t *testing.T) {
	if got := moduleConf("/etc/dkms", "nvidia"); got != "/etc/dkms/nvidia.conf" {
		t.Errorf("moduleConf() = %q, want %q", got, "/etc/dkms/nvidia.conf")
	}
}

func TestIsConfigured(t *testing.T) {
	dir := t.TempDir()

	path := filepath.Join(dir, "nvidia.conf")
	if err := os.WriteFile(path, []byte("# hook\n"), 0644); err != nil {
		t.Fatalf("write conf: %v", err)
	}

	got, err := isConfigured(path)
	if err != nil || got != true {
		t.Errorf("isConfigured(existing) = %v, %v; want true, nil", got, err)
	}

	got, err = isConfigured(filepath.Join(dir, "missing.conf"))
	if err != nil || got != false {
		t.Errorf("isConfigured(missing) = %v, %v; want false, nil", got, err)
	}

	// A dangling symlink is treated as configured: dkms reads the link itself.
	if err := os.Symlink(filepath.Join(dir, "nowhere.conf"), filepath.Join(dir, "dangling.conf")); err != nil {
		t.Fatal(err)
	}
	got, err = isConfigured(filepath.Join(dir, "dangling.conf"))
	if err != nil || got != true {
		t.Errorf("isConfigured(dangling symlink) = %v, %v; want true, nil", got, err)
	}
}

func TestIsConfiguredPermissionError(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("permissions are not enforced for root")
	}

	dir := filepath.Join(t.TempDir(), "locked")
	if err := os.Mkdir(dir, 0000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		os.Chmod(dir, 0755)
	})

	path := filepath.Join(dir, "nvidia.conf")
	got, err := isConfigured(path)
	if err == nil {
		t.Fatal("isConfigured(unreadable) succeeded, want error")
	}
	if got != false {
		t.Errorf("isConfigured(unreadable) = %v, want false", got)
	}
	if errors.Is(err, fs.ErrNotExist) {
		t.Errorf("isConfigured(unreadable) error = %q, want a non-not-exist error", err)
	}
}

func TestModuleState(t *testing.T) {
	if got := (Module{Name: "nvidia", Configured: true}).State(); got != "configured" {
		t.Errorf("State() = %q, want configured", got)
	}
	if got := (Module{Name: "nvidia", Configured: false}).State(); got != "installed" {
		t.Errorf("State() = %q, want installed", got)
	}
}
