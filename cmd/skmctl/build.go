package main

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/sk8thing/arch-skm/internal/helper"
	"github.com/sk8thing/arch-skm/internal/kernel"
	"github.com/sk8thing/arch-skm/internal/makepkg"
	"github.com/sk8thing/arch-skm/internal/patch"
	"github.com/spf13/cobra"
)

const buildExample = `  # Build linux-hardened kernel
  skmctl build linux-hardened

  # Clean and build linux-hardened kernel
  skmctl build linux-hardened -C

  # Build linux-hardened kernel with extra makepkg flags
  skmctl build linux-hardened -- -j$(nproc)`

var buildCmd = &cobra.Command{
	Use:       "build [kernel] [-- <makepkg args>...]",
	Short:     "Build a patched kernel with makepkg",
	Example:   buildExample,
	Args:      cobra.MinimumNArgs(1),
	ValidArgs: kernel.Supported(),
	RunE: func(cmd *cobra.Command, args []string) error {
		t, err := resolveTarget(args[0])
		if err != nil {
			return err
		}

		if err := validateBuildDir(t.Dir); err != nil {
			return err
		}

		syncDeps, _ := cmd.Flags().GetBool("syncdeps")
		cleanBuild, _ := cmd.Flags().GetBool("cleanbuild")
		noConfirm, _ := cmd.Flags().GetBool("noconfirm")

		fmt.Fprintf(cmd.OutOrStdout(), "Building %s in %s\n", t.Variant, t.Dir)

		return makepkg.Build(cmd.Context(), makepkg.Options{
			Dir:        t.Dir,
			MakepkgDir: t.Config.MakepkgDir,
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
	buildCmd.Flags().BoolP("syncdeps", "s", true, "Install missing dependencies with pacman (-s)")
	buildCmd.Flags().BoolP("cleanbuild", "C", false, "Remove the existing source directory before building (-C)")
	buildCmd.Flags().Bool("noconfirm", false, "Do not ask for confirmation (--noconfirm)")

	rootCmd.AddCommand(buildCmd)
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
	if !bytes.Contains(content, []byte(patch.SigningMarker)) {
		return fmt.Errorf("PKGBUILD %q is not patched for local module signing; run 'skmctl update %s' first", pkgbuildPath, filepath.Base(dir))
	}

	skmPath := filepath.Join(dir, "certs-local", "skm")
	if err := helper.VerifyExecutableFile("runtime helper", skmPath); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("build directory %q has no certs-local/skm; run 'skmctl update %s' first", dir, filepath.Base(dir))
		}
		return err
	}

	return nil
}
