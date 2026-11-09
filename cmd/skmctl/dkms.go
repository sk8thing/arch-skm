package main

import (
	"fmt"

	"github.com/sk8thing/arch-skm/internal/dkms"
	"github.com/spf13/cobra"
)

const dkmsExample = `  # Show installed DKMS modules and their signing state
  skmctl dkms status

  # Enable module signing for DKMS packages
  skmctl dkms install nvidia vboxhost

  # Disable module signing for a DKMS packages
  skmctl dkms remove nvidia vboxhost

  # Rebuild DKMS modules for all kernels
  skmctl dkms rebuild nvidia

  # Rebuild all installed DKMS modules
  skmctl dkms rebuild --all`

var (
	allFlag bool
)

var dkmsCmd = &cobra.Command{
	Use:     "dkms",
	Short:   "Manage DKMS module signing",
	Example: dkmsExample,
	Long: `Manage which DKMS packages are signed with the kernel's out-of-tree key.

DKMS reads /etc/dkms/<package>.conf, so signing is enabled per package by
symlinking that file to /etc/dkms/kernel-sign.conf.`,
}

var dkmsStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show installed DKMS modules and whether signing is configured",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		modules, err := dkms.Status(cmd.Context())
		if err != nil {
			return err
		}

		out := cmd.OutOrStdout()
		if len(modules) == 0 {
			fmt.Fprintln(out, "No installed DKMS modules")
			return nil
		}

		for _, module := range modules {
			fmt.Fprintf(out, "%s: %s\n", module.Name, module.State())
		}

		return nil
	},
}

var dkmsInstallCmd = &cobra.Command{
	Use:   "install [modules...]",
	Short: "Enable module signing for DKMS packages",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return dkms.Install(cmd.Context(), args, cmd.InOrStdin(), cmd.OutOrStdout(), cmd.ErrOrStderr())
	},
}

var dkmsRemoveCmd = &cobra.Command{
	Use:   "remove [modules...]",
	Short: "Disable module signing for DKMS packages",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return dkms.Remove(cmd.Context(), args, cmd.InOrStdin(), cmd.OutOrStdout(), cmd.ErrOrStderr())
	},
}

var dkmsRebuildCmd = &cobra.Command{
	Use:   "rebuild [modules...]",
	Short: "Rebuild DKMS modules for all kernels",
	Args:  cobra.ArbitraryArgs,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if allFlag && len(args) > 0 {
			return fmt.Errorf("cannot use all flag together with module list")
		}
		if !allFlag && len(args) == 0 {
			return fmt.Errorf("requires at least 1 module unless --all is used")
		}

		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return dkms.Rebuild(cmd.Context(), args, allFlag, cmd.InOrStdin(), cmd.OutOrStdout(), cmd.ErrOrStderr())
	},
}

func init() {
	dkmsRebuildCmd.Flags().BoolVar(&allFlag, "all", false, "Rebuild all installed DKMS modules")

	dkmsCmd.AddCommand(dkmsStatusCmd, dkmsInstallCmd, dkmsRemoveCmd, dkmsRebuildCmd)

	rootCmd.AddCommand(dkmsCmd)
}
