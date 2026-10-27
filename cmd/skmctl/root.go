package main

import "github.com/spf13/cobra"

var rootCmd = &cobra.Command{
	Use:   "skmctl",
	Short: "Build Arch Linux kernels with local module signing",
	Long:  "Download kernel packaging files, prepare local signing assets, and build signed Arch Linux kernels",
}
