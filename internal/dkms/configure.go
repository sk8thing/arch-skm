package dkms

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func Install(ctx context.Context, modules []string, stdin io.Reader, stdout, stderr io.Writer) error {
	installed, err := installedSet(ctx)
	if err != nil {
		return err
	}

	source := filepath.Join(defaultDir, kernelSignConf)
	info, err := os.Stat(source)
	if err != nil {
		return fmt.Errorf("access signing hook %s: %w", source, err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("signing hook %s is not a regular file", source)
	}

	var errs []error
	for _, name := range modules {
		if err := validateModule(name, installed); err != nil {
			errs = append(errs, err)
			continue
		}

		target := moduleConf(defaultDir, name)
		linked, err := linkedTo(target, source)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if linked {
			fmt.Fprintf(stdout, "%s: already configured\n", name)
			continue
		}

		if err := run(ctx, stdin, stdout, stderr, "ln", "-s", source, target); err != nil {
			errs = append(errs, fmt.Errorf("configure %s: %w", name, err))
			continue
		}

		fmt.Fprintf(stdout, "%s: configured\n", name)
	}

	return errors.Join(errs...)
}

func Remove(ctx context.Context, modules []string, stdin io.Reader, stdout, stderr io.Writer) error {
	installed, err := installedSet(ctx)
	if err != nil {
		return err
	}

	var errs []error
	for _, name := range modules {
		if err := validateModule(name, installed); err != nil {
			errs = append(errs, err)
			continue
		}

		target := moduleConf(defaultDir, name)
		info, err := os.Lstat(target)
		if errors.Is(err, fs.ErrNotExist) {
			fmt.Fprintf(stdout, "%s: not configured\n", name)
			continue
		}
		if err != nil {
			errs = append(errs, fmt.Errorf("access %s: %w", target, err))
			continue
		}
		if info.IsDir() {
			errs = append(errs, fmt.Errorf("%s is a directory; refusing to remove it", target))
			continue
		}

		if err := run(ctx, stdin, stdout, stderr, "rm", target); err != nil {
			errs = append(errs, fmt.Errorf("remove %s: %w", name, err))
			continue
		}

		fmt.Fprintf(stdout, "%s: removed\n", name)
	}

	return errors.Join(errs...)
}

func linkedTo(target, source string) (bool, error) {
	info, err := os.Lstat(target)
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("access %s: %w", target, err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		return false, fmt.Errorf("%s already exists and is not a symlink", target)
	}

	dest, err := os.Readlink(target)
	if err != nil {
		return false, fmt.Errorf("read symlink %s: %w", target, err)
	}
	if !filepath.IsAbs(dest) {
		dest = filepath.Join(filepath.Dir(target), dest)
	}
	if filepath.Clean(dest) == filepath.Clean(source) {
		return true, nil
	}

	return false, fmt.Errorf("%s already exists and points to %s, not %s", target, dest, source)
}

func validateModule(name string, installed map[string]struct{}) error {
	if err := validateName(name); err != nil {
		return err
	}

	if _, ok := installed[name]; !ok {
		return fmt.Errorf("module %q is not installed according to dkms status", name)
	}

	return nil
}

func validateName(name string) error {
	switch {
	case name == "":
		return fmt.Errorf("module name is empty")
	case name == "." || name == "..":
		return fmt.Errorf("invalid module name %q", name)
	case strings.ContainsAny(name, `/\`):
		return fmt.Errorf("invalid module name %q: must be a DKMS package name", name)
	}

	return nil
}

func run(ctx context.Context, stdin io.Reader, stdout, stderr io.Writer, name string, args ...string) error {
	path, err := exec.LookPath(name)
	if err != nil {
		return fmt.Errorf("find %s: %w", name, err)
	}

	command := path
	commandArgs := args
	if os.Geteuid() != 0 {
		sudoPath, err := exec.LookPath("sudo")
		if err != nil {
			return fmt.Errorf("find sudo: %w", err)
		}
		command = sudoPath
		commandArgs = append([]string{path}, args...)
	}

	cmd := exec.CommandContext(ctx, command, commandArgs...)
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s %s: %w", command, strings.Join(commandArgs, " "), err)
	}

	return nil
}
