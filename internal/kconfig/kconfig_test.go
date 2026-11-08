package kconfig

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTempFile(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "config")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	return path
}

func TestNewFromFile(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		wantKey  KeyType
		wantHash HashType
		wantErr  bool
	}{
		{
			name: "rsa stops once both key and hash are found",
			content: `
# misc
CONFIG_MODULE_SIG_KEY_TYPE_RSA=y
CONFIG_MODULE_SIG_KEY_TYPE_ECDSA=n
CONFIG_MODULE_SIG_HASH="sha512"
CONFIG_MODULE_SIG_HASH="sha256"
`,
			wantKey:  KeyTypeRSA,
			wantHash: HashTypeSHA512,
		},
		{
			name: "ecdsa sha512",
			content: `
CONFIG_MODULE_SIG_KEY_TYPE_ECDSA=y
CONFIG_MODULE_SIG_HASH="sha512"
`,
			wantKey:  KeyTypeECDSA,
			wantHash: HashTypeSHA512,
		},
		{
			name:     "sha1",
			content:  "CONFIG_MODULE_SIG_KEY_TYPE_RSA=y\nCONFIG_MODULE_SIG_HASH=sha1\n",
			wantKey:  KeyTypeRSA,
			wantHash: HashTypeSHA1,
		},
		{
			name:    "key type not enabled",
			content: "CONFIG_MODULE_SIG_KEY_TYPE_RSA=n\nCONFIG_MODULE_SIG_HASH=sha256\n",
			wantErr: true,
		},
		{
			name:    "unsupported hash",
			content: "CONFIG_MODULE_SIG_KEY_TYPE_RSA=y\nCONFIG_MODULE_SIG_HASH=sha3-256\n",
			wantErr: true,
		},
		{
			name:    "no module signing options",
			content: "CONFIG_EFI=y\n",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writeTempFile(t, tt.content)

			cfg, err := NewFromFile(path)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("NewFromFile(%q) succeeded, want error", path)
				}
				if !strings.Contains(err.Error(), "missing/unsupported") {
					t.Errorf("NewFromFile(%q) error = %q, want it to mention missing/unsupported", path, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("NewFromFile(%q) error: %v", path, err)
			}

			if got := cfg.KeyType(); got != tt.wantKey {
				t.Errorf("KeyType() = %q, want %q", got, tt.wantKey)
			}
			if got := cfg.HashType(); got != tt.wantHash {
				t.Errorf("HashType() = %q, want %q", got, tt.wantHash)
			}
		})
	}
}

func TestNewFromFileMissingFile(t *testing.T) {
	if _, err := NewFromFile(filepath.Join(t.TempDir(), "nonexistent")); err == nil {
		t.Fatal("NewFromFile() succeeded, want error")
	}
}

func TestUpdateTrustedKeysReplacesExisting(t *testing.T) {
	content := `CONFIG_MODULE_SIG_KEY_TYPE_RSA=y
CONFIG_MODULE_SIG_HASH="sha256"
CONFIG_EFI=y
CONFIG_SYSTEM_TRUSTED_KEYS=""
`
	path := writeTempFile(t, content)
	cfg, err := NewFromFile(path)
	if err != nil {
		t.Fatalf("NewFromFile() error: %v", err)
	}

	value := "/home/user/keys/signing_prv.pem"
	if err := cfg.UpdateTrustedKeys(value); err != nil {
		t.Fatalf("UpdateTrustedKeys() error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}

	want := `CONFIG_MODULE_SIG_KEY_TYPE_RSA=y
CONFIG_MODULE_SIG_HASH="sha256"
CONFIG_EFI=y
CONFIG_SYSTEM_TRUSTED_KEYS="` + value + `"
`
	if string(data) != want {
		t.Errorf("updated config:\n%s\nwant:\n%s", data, want)
	}
}

func TestUpdateTrustedKeysAppendsWhenMissing(t *testing.T) {
	path := writeTempFile(t, "CONFIG_MODULE_SIG_KEY_TYPE_RSA=y\nCONFIG_MODULE_SIG_HASH=\"sha256\"\nCONFIG_EFI=y\n")
	cfg, err := NewFromFile(path)
	if err != nil {
		t.Fatalf("NewFromFile() error: %v", err)
	}

	if err := cfg.UpdateTrustedKeys("./keys/signing_prv.pem"); err != nil {
		t.Fatalf("UpdateTrustedKeys() error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}

	want := "CONFIG_MODULE_SIG_KEY_TYPE_RSA=y\nCONFIG_MODULE_SIG_HASH=\"sha256\"\nCONFIG_EFI=y\n" + trustedKeysValue("./keys/signing_prv.pem") + "\n"
	if string(data) != want {
		t.Errorf("updated config = %q, want %q", data, want)
	}
}

func TestUpdateTrustedKeysReplacesAllOccurrences(t *testing.T) {
	path := writeTempFile(t, strings.Join([]string{
		"CONFIG_MODULE_SIG_KEY_TYPE_RSA=y",
		"CONFIG_MODULE_SIG_HASH=\"sha256\"",
		"CONFIG_SYSTEM_TRUSTED_KEYS=\"old-one\"",
		"CONFIG_EFI=y",
		"CONFIG_SYSTEM_TRUSTED_KEYS=\"old-two\"",
	}, "\n")+"\n")

	cfg, err := NewFromFile(path)
	if err != nil {
		t.Fatalf("NewFromFile() error: %v", err)
	}

	if err := cfg.UpdateTrustedKeys("new"); err != nil {
		t.Fatalf("UpdateTrustedKeys() error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}

	if got := strings.Count(string(data), trustedKeysValue("new")); got != 2 {
		t.Errorf("found trusted keys line %d times, want 2", got)
	}
}

func TestUpdateTrustedKeysLeavesNoTempFile(t *testing.T) {
	path := writeTempFile(t, "CONFIG_MODULE_SIG_KEY_TYPE_RSA=y\nCONFIG_MODULE_SIG_HASH=\"sha256\"\n"+trustedKeysValue("old")+"\n")
	cfg, err := NewFromFile(path)
	if err != nil {
		t.Fatalf("NewFromFile() error: %v", err)
	}

	if err := cfg.UpdateTrustedKeys("new"); err != nil {
		t.Fatalf("UpdateTrustedKeys() error: %v", err)
	}

	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		t.Fatalf("read temp directory: %v", err)
	}
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".tmp") {
			t.Errorf("temp file %q left behind", entry.Name())
		}
	}
}

func TestUpdateTrustedKeysNonExistent(t *testing.T) {
	cfg := &KConfig{path: filepath.Join(t.TempDir(), "nonexistent")}

	if err := cfg.UpdateTrustedKeys("value"); err == nil {
		t.Fatal("UpdateTrustedKeys() succeeded, want error")
	}
}
