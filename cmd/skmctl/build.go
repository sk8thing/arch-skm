package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sk8thing/arch-skm-go/internal/kernel"
	"github.com/sk8thing/arch-skm-go/internal/makepkg"
	"github.com/spf13/cobra"
)

const signingMarker = "Out-of-tree module signing"

var buildCmd = &cobra.Command{
	Use:       "build <kernel> [-- <makepkg args>...]",
	Short:     "Build a prepared kernel package with makepkg",
	Long:      "Run makepkg for a kernel build directory prepared by 'skmctl update'. The directory must contain a patched PKGBUILD and a seeded certs-local directory. Arguments after '--' are passed through to makepkg.",
	Args:      cobra.MinimumNArgs(1),
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

		syncDeps, _ := cmd.Flags().GetBool("syncdeps")
		cleanBuild, _ := cmd.Flags().GetBool("cleanbuild")
		noConfirm, _ := cmd.Flags().GetBool("noconfirm")

		fmt.Fprintf(cmd.OutOrStdout(), "Building %s in %s\n", variant, dir)

		return makepkg.Build(cmd.Context(), makepkg.Options{
			Dir:        dir,
			MakepkgDir: cfg.MakepkgDir,
			SyncDeps:   syncDeps,
			CleanBuild: cleanBuild,
			NoConfirm:  noConfirm,
			ExtraArgs:  args[1:],
			Stdin:      cmd.InOrStdin(),
			Stdout:     cmd.OutOrStdout(),
			Stderr:     cmd.ErrOrStderr(),
		})
	},
}

func init() {
	rootCmd.AddCommand(buildCmd)

	buildCmd.Flags().Bool("syncdeps", true, "Install missing dependencies with pacman (-s)")
	buildCmd.Flags().Bool("cleanbuild", false, "Remove the existing source directory before building (-C)")
	buildCmd.Flags().Bool("noconfirm", false, "Do not ask for confirmation (--noconfirm)")
}

func validateBuildDir(dir string) error {
	info, err := os.Stat(dir)
	if os.IsNotExist(err) {
		return fmt.Errorf("build directory %q does not exist; run 'skmctl update %s' first", dir, filepath.Base(dir))
	}
	if err != nil {
		return fmt.Errorf("access build directory %q: %w", dir, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("build path %q is not a directory", dir)
	}

	pkgbuildPath := filepath.Join(dir, "PKGBUILD")
	content, err := os.ReadFile(pkgbuildPath)
	if os.IsNotExist(err) {
		return fmt.Errorf("build directory %q has no PKGBUILD; run 'skmctl update %s' first", dir, filepath.Base(dir))
	}
	if err != nil {
		return fmt.Errorf("read PKGBUILD %q: %w", pkgbuildPath, err)
	}
	if !bytes.Contains(content, []byte(signingMarker)) {
		return fmt.Errorf("PKGBUILD %q is not patched for local module signing; run 'skmctl update %s' first", pkgbuildPath, filepath.Base(dir))
	}

	skmPath := filepath.Join(dir, "certs-local", "skm")
	info, err = os.Stat(skmPath)
	if os.IsNotExist(err) {
		return fmt.Errorf("build directory %q has no certs-local/skm; run 'skmctl update %s' first", dir, filepath.Base(dir))
	}
	if err != nil {
		return fmt.Errorf("access runtime helper %q: %w", skmPath, err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("runtime helper %q is not a regular file", skmPath)
	}
	if info.Mode().Perm()&0111 == 0 {
		return fmt.Errorf("runtime helper %q is not executable", skmPath)
	}

	return nil
}
