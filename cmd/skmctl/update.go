package main

import (
	"fmt"
	"path/filepath"

	"github.com/sk8thing/arch-skm-go/internal/helper"
	"github.com/sk8thing/arch-skm-go/internal/install"
	"github.com/sk8thing/arch-skm-go/internal/kernel"
	"github.com/sk8thing/arch-skm-go/internal/patch"
	"github.com/spf13/cobra"
)

const updateExample = `  # Clone and prepare linux-hardened kernel
  skmctl update linux-hardened`

var updateCmd = &cobra.Command{
	Use:       "update [kernel]",
	Short:     "Clone or update a kernel package, patch it, and prepare build directory",
	Example:   updateExample,
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

		if err := install.PrepareBuildDir(dest, skmPath); err != nil {
			return fmt.Errorf("prepare build directory %q: %w", dest, err)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Prepared %s\n", filepath.Join(dest, "certs-local"))
		return nil
	},
}

func init() {
	updateCmd.Flags().BoolP("force", "f", false, "Remove and re-clone an existing kernel build directory instead of resetting it")
	updateCmd.Flags().String("skm", "", "Path to the skm runtime helper (default: sibling skm or /usr/lib/skm/skm)")

	rootCmd.AddCommand(updateCmd)
}
