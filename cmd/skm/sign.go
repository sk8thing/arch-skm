package main

import (
	"fmt"

	"github.com/sk8thing/arch-skm-go/internal/sign"
	"github.com/spf13/cobra"
)

const signExample = `  # Sign a single kernel module
  skm sign ./module.ko

  # Sign multiple kernel modules
  skm sign ./module1.ko ./module2.ko.zst ./module3.ko.gz

  # Sign all kernel modules found in a directory
  skm sign --dir ./modules`

var (
	dirFlag string
)

var signCmd = &cobra.Command{
	Use:     "sign [modules...]",
	Short:   "Sign out of tree kernel modules",
	Example: signExample,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if dirFlag != "" && len(args) > 0 {
			return fmt.Errorf("cannot use module directory together with module list")
		}

		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		modules := args
		if dirFlag != "" {
			var err error
			modules, err = sign.FindModules(dirFlag)
			if err != nil {
				return fmt.Errorf("failed to find modules: %w", err)
			}
		}

		if len(modules) == 0 {
			return fmt.Errorf("no modules to sign")
		}

		if err := sign.SignModules(modules); err != nil {
			return fmt.Errorf("failed to sign modules: %w", err)
		}

		return nil
	},
}

func init() {
	signCmd.Flags().StringVarP(&dirFlag, "dir", "d", "", "Directory containing kernel modules to sign")

	rootCmd.AddCommand(signCmd)
}
