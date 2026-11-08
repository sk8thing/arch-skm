package makepkg

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestArgs(t *testing.T) {
	tests := []struct {
		name string
		opts Options
		want []string
	}{
		{
			name: "defaults",
			opts: Options{},
			want: []string{"-L"},
		},
		{
			name: "sync deps",
			opts: Options{SyncDeps: true},
			want: []string{"-L", "-s"},
		},
		{
			name: "clean build",
			opts: Options{CleanBuild: true},
			want: []string{"-L", "-C"},
		},
		{
			name: "no confirm",
			opts: Options{NoConfirm: true},
			want: []string{"-L", "--noconfirm"},
		},
		{
			name: "all flags",
			opts: Options{SyncDeps: true, CleanBuild: true, NoConfirm: true},
			want: []string{"-L", "-s", "-C", "--noconfirm"},
		},
		{
			name: "extra args appended last",
			opts: Options{CleanBuild: true, ExtraArgs: []string{"-j$(nproc)", "--makepkg-vars=X"}},
			want: []string{"-L", "-C", "-j$(nproc)", "--makepkg-vars=X"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Args(tt.opts)
			if len(got) != len(tt.want) {
				t.Fatalf("Args() = %v, want %v", got, tt.want)
			}
			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Errorf("Args()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestSetEnv(t *testing.T) {
	env := []string{
		"PATH=/usr/bin",
		"BUILDDIR=/old",
		"MAKEFLAGS=-j4",
		"HOME=/home/user",
	}

	got := setEnv(env, "BUILDDIR", "/new")

	want := []string{
		"PATH=/usr/bin",
		"MAKEFLAGS=-j4",
		"HOME=/home/user",
		"BUILDDIR=/new",
	}

	if len(got) != len(want) {
		t.Fatalf("setEnv() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("setEnv()[%d] = %q, want %q", i, got[i], want[i])
		}
	}

	// The original slice is not mutated.
	if !strings.Contains(env[1], "old") {
		t.Error("setEnv() mutated the input slice")
	}
}

func TestSetEnvAppendsWithoutExistingEntry(t *testing.T) {
	got := setEnv([]string{"PATH=/usr/bin"}, "BUILDDIR", "/build")

	if len(got) != 2 || got[1] != "BUILDDIR=/build" {
		t.Errorf("setEnv() = %v, want the appended BUILDDIR entry", got)
	}
}

func TestBuildRequiresDirectory(t *testing.T) {
	if err := Build(t.Context(), Options{}); err == nil {
		t.Fatal("Build() succeeded, want error")
	} else if !strings.Contains(err.Error(), "build directory is required") {
		t.Errorf("Build() error = %q, want it to mention the required build directory", err)
	}
}

func TestBuildPropagatesCommandErrors(t *testing.T) {
	// Either makepkg is not on PATH or the child fails to chdir; whichever
	// happens, the error must surface to the caller.
	opts := Options{Dir: filepath.Join(t.TempDir(), "nothing-here")}

	if err := Build(t.Context(), opts); err == nil {
		t.Fatal("Build() succeeded, want error")
	}
}
