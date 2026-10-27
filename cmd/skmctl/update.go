package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sk8thing/arch-skm-go/internal/helper"
	"github.com/sk8thing/arch-skm-go/internal/install"
	"github.com/sk8thing/arch-skm-go/internal/kernel"
	"github.com/sk8thing/arch-skm-go/internal/patch"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:       "update <kernel>",
	Short:     "Clone or update a kernel package, patch it, and seed its build directory",
	Long:      "Clone or update the Arch Linux packaging repository for a kernel, patch the PKGBUILD for local module signing, and seed certs-local in its build directory. Existing checkouts are reset to their upstream branch before patching; untracked files such as certs-local are preserved.",
	Args:      cobra.ExactArgs(1),
	ValidArgs: kernel.Supported(),
	RunE: func(cmd *cobra.Command, args []string) error {
		variant := args[0]
		if _, err := kernel.RepoURL(variant); err != nil {
			return err
		}

		buildDir, _ := cmd.Flags().GetString("build-dir")
		buildDir, err := expandHome(buildDir)
		if err != nil {
			return err
		}

		buildDir, err = filepath.Abs(buildDir)
		if err != nil {
			return fmt.Errorf("resolve build directory: %w", err)
		}

		force, _ := cmd.Flags().GetBool("force")
		skmPathFlag, _ := cmd.Flags().GetString("skm")
		skmPath, err := helper.Resolve(skmPathFlag)
		if err != nil {
			return err
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Updating %s in %s\n", variant, buildDir)

		dest, err := kernel.UpdateRepo(cmd.Context(), kernel.UpdateOptions{
			Kernel:   variant,
			BuildDir: buildDir,
			Force:    force,
			Stdout:   cmd.OutOrStdout(),
			Stderr:   cmd.ErrOrStderr(),
		})
		if err != nil {
			return err
		}

		if err := patch.PatchPKGBUILD(dest); err != nil {
			return fmt.Errorf("patch PKGBUILD in %q: %w", dest, err)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Patched %s\n", filepath.Join(dest, "PKGBUILD"))

		if err := install.SeedBuildDir(dest, skmPath); err != nil {
			return fmt.Errorf("seed build directory %q: %w", dest, err)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Seeded %s\n", filepath.Join(dest, "certs-local"))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)

	updateCmd.Flags().String("build-dir", defaultBuildDir(), "Directory containing kernel build directories")
	updateCmd.Flags().Bool("force", false, "Remove and re-clone an existing kernel build directory instead of resetting it")
	updateCmd.Flags().String("skm", "", "Path to the skm runtime helper (default: sibling skm or /usr/lib/skm/skm)")
}

func defaultBuildDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	return filepath.Join(home, "build")
}

func expandHome(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("build directory is required")
	}
	if path != "~" && !strings.HasPrefix(path, "~/") {
		return path, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	if path == "~" {
		return home, nil
	}

	return filepath.Join(home, path[2:]), nil
}
