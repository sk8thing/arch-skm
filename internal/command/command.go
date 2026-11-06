package command

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

type Runner struct {
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

type Options struct {
	Sudo bool
	Dir  string
	Env  []string
}

func (r Runner) Run(ctx context.Context, opts Options, name string, args ...string) error {
	cmd, err := r.command(ctx, opts, name, args)
	if err != nil {
		return err
	}

	cmd.Stdin = r.Stdin
	cmd.Stdout = r.Stdout
	cmd.Stderr = r.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s: %w", cmd.String(), err)
	}

	return nil
}

func (r Runner) Output(ctx context.Context, opts Options, name string, args ...string) (string, error) {
	cmd, err := r.command(ctx, opts, name, args)
	if err != nil {
		return "", err
	}

	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && len(exitErr.Stderr) > 0 {
			return "", fmt.Errorf("%s: %s", cmd.String(), strings.TrimSpace(string(exitErr.Stderr)))
		}

		return "", fmt.Errorf("%s: %w", cmd.String(), err)
	}

	return string(output), nil
}

func (r Runner) Combined(ctx context.Context, opts Options, name string, args ...string) (string, error) {
	cmd, err := r.command(ctx, opts, name, args)
	if err != nil {
		return "", err
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			return "", fmt.Errorf("%s: %w", cmd.String(), err)
		}

		return "", fmt.Errorf("%s: %w: %s", cmd.String(), err, message)
	}

	return string(output), nil
}

func (r Runner) command(ctx context.Context, opts Options, name string, args []string) (*exec.Cmd, error) {
	path, err := exec.LookPath(name)
	if err != nil {
		return nil, fmt.Errorf("find %s: %w", name, err)
	}

	command, commandArgs := path, args
	if opts.Sudo && os.Geteuid() != 0 {
		sudoPath, err := exec.LookPath("sudo")
		if err != nil {
			return nil, fmt.Errorf("find sudo: %w", err)
		}

		command = sudoPath
		commandArgs = append([]string{path}, args...)
	}

	cmd := exec.CommandContext(ctx, command, commandArgs...)
	cmd.Dir = opts.Dir
	cmd.Env = opts.Env

	return cmd, nil
}
