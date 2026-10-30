package main

import (
	"fmt"
	"path/filepath"

	"github.com/sk8thing/arch-skm-go/internal/install"
	"github.com/spf13/cobra"
)

const installCertsExample = `  # Install keys and binary into certs-local
  skm install-certs ./certs-local`

var installCertsCmd = &cobra.Command{
	Use:     "install-certs [destination]",
	Short:   "Install signing keys and binary into kernel build directory",
	Example: installCertsExample,
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dir := args[0]
		destPath, err := filepath.Abs(dir)
		if err != nil {
			return fmt.Errorf("failed to resolve destination: %w", err)
		}

		if err := install.InstallCertificates(destPath); err != nil {
			return fmt.Errorf("failed to install certificates: %w", err)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(installCertsCmd)
}
