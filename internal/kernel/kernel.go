package kernel

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/sk8thing/arch-skm/internal/command"
)

type definition struct {
	Name string
	URL  string
}

var definitions = []definition{
	{Name: "linux", URL: "https://gitlab.archlinux.org/archlinux/packaging/packages/linux.git"},
	{Name: "linux-hardened", URL: "https://gitlab.archlinux.org/archlinux/packaging/packages/linux-hardened.git"},
	{Name: "linux-lts", URL: "https://gitlab.archlinux.org/archlinux/packaging/packages/linux-lts.git"},
	{Name: "linux-zen", URL: "https://gitlab.archlinux.org/archlinux/packaging/packages/linux-zen.git"},
	{Name: "linux-rt", URL: "https://gitlab.archlinux.org/archlinux/packaging/packages/linux-rt.git"},
	{Name: "linux-libre", URL: "https://aur.archlinux.org/linux-libre.git"},
	{Name: "linux-xanmod", URL: "https://aur.archlinux.org/linux-xanmod.git"},
}

func Supported() []string {
	supported := make([]string, len(definitions))
	for i, definition := range definitions {
		supported[i] = definition.Name
	}
	return supported
}

func RepoURL(variant string) (string, error) {
	for _, definition := range definitions {
		if definition.Name == variant {
			return definition.URL, nil
		}
	}

	return "", fmt.Errorf("unsupported kernel %q (supported: %s)", variant, strings.Join(Supported(), ", "))
}

type UpdateOptions struct {
	Kernel   string
	BuildDir string
	Force    bool
	Stdout   io.Writer
	Stderr   io.Writer
}

func UpdateRepo(ctx context.Context, opts UpdateOptions) (string, error) {
	repoURL, err := RepoURL(opts.Kernel)
	if err != nil {
		return "", err
	}

	dest, err := updateRepoURL(ctx, repoURL, opts)
	if err != nil {
		return "", err
	}

	if _, err := os.Stat(filepath.Join(dest, "PKGBUILD")); err != nil {
		return "", fmt.Errorf("kernel package %q has no PKGBUILD: %w", dest, err)
	}

	return dest, nil
}

func updateRepoURL(ctx context.Context, repoURL string, opts UpdateOptions) (string, error) {
	if opts.BuildDir == "" {
		return "", fmt.Errorf("build directory is required")
	}
	if opts.Kernel == "" {
		return "", fmt.Errorf("kernel variant is required")
	}

	dest := filepath.Join(opts.BuildDir, opts.Kernel)

	if opts.Force {
		if err := os.RemoveAll(dest); err != nil {
			return "", fmt.Errorf("remove existing build directory %q: %w", dest, err)
		}
	}

	info, err := os.Stat(dest)
	switch {
	case err == nil:
		if !info.IsDir() {
			return "", fmt.Errorf("build path %q is not a directory", dest)
		}
		if err := updateExisting(ctx, repoURL, dest, opts); err != nil {
			return "", fmt.Errorf("update existing clone %q: %w (use --force to re-clone)", dest, err)
		}
	case os.IsNotExist(err):
		if err := os.MkdirAll(opts.BuildDir, 0755); err != nil {
			return "", fmt.Errorf("create build directory %q: %w", opts.BuildDir, err)
		}
		if err := clone(ctx, repoURL, dest, opts); err != nil {
			_ = os.RemoveAll(dest)
			return "", fmt.Errorf("clone %s: %w", repoURL, err)
		}
	default:
		return "", fmt.Errorf("access build path %q: %w", dest, err)
	}

	return dest, nil
}

func updateExisting(ctx context.Context, repoURL, dest string, opts UpdateOptions) error {
	gitDir := filepath.Join(dest, ".git")
	info, err := os.Stat(gitDir)
	if err != nil {
		return fmt.Errorf("%q is not a git clone", dest)
	}
	if !info.IsDir() {
		return fmt.Errorf("%q is not a git clone", dest)
	}

	remote, err := gitOutput(ctx, dest, "remote", "get-url", "origin")
	if err != nil {
		return fmt.Errorf("read origin remote: %w", err)
	}
	if strings.TrimSpace(remote) != repoURL {
		return fmt.Errorf("origin remote is %q, expected %q", strings.TrimSpace(remote), repoURL)
	}

	if err := runGit(ctx, dest, opts, "fetch", "--prune", "origin"); err != nil {
		return err
	}

	return runGit(ctx, dest, opts, "reset", "--hard", "@{u}")
}

func clone(ctx context.Context, repoURL, dest string, opts UpdateOptions) error {
	return runGit(ctx, "", opts, "clone", repoURL, dest)
}

func runGit(ctx context.Context, dir string, opts UpdateOptions, args ...string) error {
	runner := command.Runner{Stdout: opts.Stdout, Stderr: opts.Stderr}

	return runner.Run(ctx, command.Options{Dir: dir}, "git", args...)
}

func gitOutput(ctx context.Context, dir string, args ...string) (string, error) {
	return command.Runner{}.Output(ctx, command.Options{Dir: dir}, "git", args...)
}
