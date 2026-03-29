package skm

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/sk8thing/arch-skm-go/internal/patch"
	"github.com/spf13/cobra"
)

var patchCmd = &cobra.Command{
	Use:   "patch",
	Short: "Patch the kernel PKGBUILD file",
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

		err = patch.PatchPKGBUILD(absPath)
		if err != nil {
			return fmt.Errorf("failed to patch PKGBUILD: %w", err)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(patchCmd)

	patchCmd.Flags().StringP("path", "p", "", "Path to the directory containing the PKGBUILD file")
	patchCmd.MarkFlagRequired("path")
}
