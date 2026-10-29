package main

import "github.com/spf13/cobra"

var rootCmd = &cobra.Command{
	Use:   "skm",
	Short: "Sign Kernel Modules",
	Long:  "Toolkit for signing out-of-tree kernel modules on Arch Linux",
}
