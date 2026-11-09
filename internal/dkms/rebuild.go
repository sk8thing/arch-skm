package dkms

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/sk8thing/arch-skm/internal/command"
)

func Rebuild(ctx context.Context, modules []string, all bool, stdin io.Reader, stdout, stderr io.Writer) error {
	entries, err := installedEntries(ctx)
	if err != nil {
		return err
	}

	if all {
		modules = statusNames(entries)
	}
	if len(modules) == 0 {
		if all {
			fmt.Fprintln(stdout, "No installed DKMS modules")
			return nil
		}

		return fmt.Errorf("no modules specified")
	}

	installed := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		installed[entry.Name] = struct{}{}
	}

	var errs []error
	for _, name := range modules {
		if err := validateModule(name, installed); err != nil {
			errs = append(errs, err)
			continue
		}

		specs := moduleSpecs(entries, name)
		if len(specs) == 0 {
			errs = append(errs, fmt.Errorf("module %q has no installed version in dkms status", name))
			continue
		}

		var rebuildErrs []error
		for _, spec := range specs {
			if err := rebuildModule(ctx, spec, stdin, stdout, stderr); err != nil {
				rebuildErrs = append(rebuildErrs, fmt.Errorf("rebuild %s: %w", name, err))
			}
		}
		if len(rebuildErrs) > 0 {
			errs = append(errs, rebuildErrs...)
			continue
		}

		fmt.Fprintf(stdout, "%s: rebuilt\n", name)
	}

	return errors.Join(errs...)
}

func moduleSpecs(entries []statusEntry, name string) []string {
	var specs []string
	seen := make(map[string]bool)

	for _, entry := range entries {
		if entry.Name != name {
			continue
		}

		spec := entry.Spec()
		if !seen[spec] {
			seen[spec] = true
			specs = append(specs, spec)
		}
	}

	return specs
}

func rebuildModule(ctx context.Context, module string, stdin io.Reader, stdout, stderr io.Writer) error {
	runner := command.Runner{Stdin: stdin, Stdout: stdout, Stderr: stderr}

	if err := runner.Run(ctx, command.Options{Sudo: true}, "dkms", "remove", module); err != nil {
		return fmt.Errorf("remove %s: %w", module, err)
	}

	if err := runner.Run(ctx, command.Options{Sudo: true}, "dkms", "install", module); err != nil {
		return fmt.Errorf("install %s: %w", module, err)
	}

	return nil
}
