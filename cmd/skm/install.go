package skm

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/sk8thing/arch-skm-go/internal/install"
	"github.com/spf13/cobra"
)

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install files and create certs-local directory",
	RunE: func(cmd *cobra.Command, args []string) error {
		path, _ := cmd.Flags().GetString("path")

		absPath, err := filepath.Abs(path)
		if err != nil {
			return fmt.Errorf("failed to resolve path: %w", err)
		}

		info, err := os.Stat(absPath)
		if os.IsNotExist(err) {
			return fmt.Errorf("path does not exist: %s", absPath)
		} else if err != nil {
			return fmt.Errorf("failed to access path: %w", err)
		}

		if !info.IsDir() {
			return fmt.Errorf("path is not a directory: %s", absPath)
		}

		err = install.InstallFiles(absPath)
		if err != nil {
			return fmt.Errorf("failed to patch PKGBUILD: %w", err)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(installCmd)

	installCmd.Flags().StringP("path", "p", "", "Path to kernel build directory")
	installCmd.MarkFlagRequired("path")
}
