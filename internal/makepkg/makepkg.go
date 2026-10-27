// Package makepkg runs makepkg inside a prepared kernel build directory.
package makepkg

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

type Options struct {
	Dir        string
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

	path, err := exec.LookPath("makepkg")
	if err != nil {
		return fmt.Errorf("find makepkg: %w", err)
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

	args := Args(opts)
	cmd := exec.CommandContext(ctx, path, args...)
	cmd.Dir = opts.Dir
	cmd.Stdin = opts.Stdin
	cmd.Stdout = opts.Stdout
	cmd.Stderr = opts.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("makepkg %s: %w", strings.Join(args, " "), err)
	}

	return nil
}
