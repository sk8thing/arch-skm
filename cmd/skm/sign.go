package main

import (
	"fmt"

	"github.com/sk8thing/arch-skm/internal/sign"
	"github.com/spf13/cobra"
)

const signExample = `  # Sign a single kernel module
  skm sign ./module.ko

  # Sign multiple kernel modules
  skm sign ./module1.ko ./module2.ko.zst ./module3.ko.gz

  # Sign all kernel modules found in a directory
  skm sign --dir ./modules

  # Limit the number of modules signed concurrently
  skm sign --dir ./modules --jobs 4`

var (
	dirFlag  string
	jobsFlag int
)

var signCmd = &cobra.Command{
	Use:     "sign [modules...]",
	Short:   "Sign out of tree kernel modules",
	Example: signExample,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if dirFlag != "" && len(args) > 0 {
			return fmt.Errorf("cannot use module directory together with module list")
		}
		if jobsFlag < 0 {
			return fmt.Errorf("jobs must be zero or greater")
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

		if err := sign.SignModules(cmd.Context(), modules, jobsFlag); err != nil {
			return fmt.Errorf("failed to sign modules: %w", err)
		}

		return nil
	},
}

func init() {
	signCmd.Flags().StringVarP(&dirFlag, "dir", "d", "", "Directory containing kernel modules to sign")
	signCmd.Flags().IntVarP(&jobsFlag, "jobs", "j", 0, "Number of modules to sign concurrently (0 = number of CPUs)")

	rootCmd.AddCommand(signCmd)
}
