package makepkg

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/sk8thing/arch-skm/internal/command"
)

type Options struct {
	Dir        string
	MakepkgDir string
	SyncDeps   bool
	CleanBuild bool
	NoConfirm  bool
	ExtraArgs  []string
	Stdin      io.Reader
	Stdout     io.Writer
	Stderr     io.Writer
}

func Args(opts Options) []string {
	args := []string{"-L"}
	if opts.SyncDeps {
		args = append(args, "-s")
	}
	if opts.CleanBuild {
		args = append(args, "-C")
	}
	if opts.NoConfirm {
		args = append(args, "--noconfirm")
	}

	return append(args, opts.ExtraArgs...)
}

func Build(ctx context.Context, opts Options) error {
	if opts.Dir == "" {
		return fmt.Errorf("build directory is required")
	}

	if opts.Stdin == nil {
		opts.Stdin = os.Stdin
	}
	if opts.Stdout == nil {
		opts.Stdout = os.Stdout
	}
	if opts.Stderr == nil {
		opts.Stderr = os.Stderr
	}

	runner := command.Runner{
		Stdin:  opts.Stdin,
		Stdout: opts.Stdout,
		Stderr: opts.Stderr,
	}

	commandOpts := command.Options{Dir: opts.Dir}
	if opts.MakepkgDir != "" {
		commandOpts.Env = setEnv(os.Environ(), "BUILDDIR", opts.MakepkgDir)
	}

	return runner.Run(ctx, commandOpts, "makepkg", Args(opts)...)
}

func setEnv(env []string, key, value string) []string {
	prefix := key + "="
	updated := make([]string, 0, len(env)+1)
	for _, entry := range env {
		if !strings.HasPrefix(entry, prefix) {
			updated = append(updated, entry)
		}
	}

	return append(updated, prefix+value)
}
