package skm

import (
	"fmt"
	"path/filepath"

	"github.com/sk8thing/arch-skm-go/internal/install"
	"github.com/spf13/cobra"
)

var installCertsCmd = &cobra.Command{
	Use:   "install-certs <destination>",
	Short: "Install signing keys and binary into a kernel build directory",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		destPath, err := filepath.Abs(args[0])
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
