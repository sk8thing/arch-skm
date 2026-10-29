package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/sk8thing/arch-skm-go/internal/kernel"
	"github.com/spf13/cobra"
)

var installCmd = &cobra.Command{
	Use:       "install <kernel>",
	Short:     "Install built kernel packages with pacman",
	Long:      "Install the packages built for a kernel with pacman -U. The package list is taken from the current PKGBUILD and the packages must already exist.",
	Args:      cobra.ExactArgs(1),
	ValidArgs: kernel.Supported(),
	RunE: func(cmd *cobra.Command, args []string) error {
		variant := args[0]
		if _, err := kernel.RepoURL(variant); err != nil {
			return err
		}

		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		buildDir, err := resolveBuildDir(cfg)
		if err != nil {
			return err
		}

		dir := filepath.Join(buildDir, variant)
		if err := validateBuildDir(dir); err != nil {
			return err
		}

		packages, err := listBuiltPackages(cmd, dir, variant)
		if err != nil {
			return err
		}

		noConfirm, _ := cmd.Flags().GetBool("noconfirm")
		fmt.Fprintf(cmd.OutOrStdout(), "Installing %s\n", strings.Join(packages, " "))

		return runPacman(cmd, packages, noConfirm)
	},
}

func init() {
	rootCmd.AddCommand(installCmd)

	installCmd.Flags().Bool("noconfirm", false, "Do not ask for confirmation (--noconfirm)")
}

func listBuiltPackages(cmd *cobra.Command, dir string, variant string) ([]string, error) {
	path, err := exec.LookPath("makepkg")
	if err != nil {
		return nil, fmt.Errorf("find makepkg: %w", err)
	}

	listCmd := exec.CommandContext(cmd.Context(), path, "--packagelist")
	listCmd.Dir = dir

	output, err := listCmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && len(exitErr.Stderr) > 0 {
			return nil, fmt.Errorf("makepkg --packagelist: %s", strings.TrimSpace(string(exitErr.Stderr)))
		}
		return nil, fmt.Errorf("makepkg --packagelist: %w", err)
	}

	var packages []string
	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if !filepath.IsAbs(line) {
			line = filepath.Join(dir, line)
		}

		info, err := os.Stat(line)
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("built package %q not found; run 'skmctl build %s' first", line, variant)
		}
		if err != nil {
			return nil, fmt.Errorf("access package %q: %w", line, err)
		}
		if !info.Mode().IsRegular() {
			return nil, fmt.Errorf("package path %q is not a regular file", line)
		}

		packages = append(packages, line)
	}

	if len(packages) == 0 {
		return nil, fmt.Errorf("makepkg --packagelist returned no packages for %s", variant)
	}

	return packages, nil
}

func runPacman(cmd *cobra.Command, packages []string, noConfirm bool) error {
	pacmanPath, err := exec.LookPath("pacman")
	if err != nil {
		return fmt.Errorf("find pacman: %w", err)
	}

	args := []string{"-U"}
	if noConfirm {
		args = append(args, "--noconfirm")
	}
	args = append(args,
		"--overwrite", "/etc/dkms/kernel-sign.conf",
		"--overwrite", "/etc/dkms/kernel-sign.sh",
	)
	args = append(args, packages...)

	command := pacmanPath
	if os.Geteuid() != 0 {
		sudoPath, err := exec.LookPath("sudo")
		if err != nil {
			return fmt.Errorf("find sudo: %w", err)
		}
		command = sudoPath
		args = append([]string{pacmanPath}, args...)
	}

	pacmanCmd := exec.CommandContext(cmd.Context(), command, args...)
	pacmanCmd.Stdin = cmd.InOrStdin()
	pacmanCmd.Stdout = cmd.OutOrStdout()
	pacmanCmd.Stderr = cmd.ErrOrStderr()

	if err := pacmanCmd.Run(); err != nil {
		return fmt.Errorf("%s %s: %w", command, strings.Join(args, " "), err)
	}

	return nil
}
