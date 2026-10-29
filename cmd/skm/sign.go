package main

import (
	"fmt"

	"github.com/sk8thing/arch-skm-go/internal/sign"
	"github.com/spf13/cobra"
)

var signCmd = &cobra.Command{
	Use:   "sign [modules...]",
	Short: "Sign out of tree kernel modules",
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, _ := cmd.Flags().GetString("directory")

		modules := args
		if dir != "" {
			if len(args) > 0 {
				return fmt.Errorf("cannot use a module directory together with module paths")
			}

			var err error
			modules, err = sign.FindModules(dir)
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
	rootCmd.AddCommand(signCmd)

	signCmd.Flags().StringP("directory", "d", "", "Directory containing kernel modules to sign")
}
