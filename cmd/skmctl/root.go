package main

import "github.com/spf13/cobra"

var rootCmd = &cobra.Command{
	Use:   "skmctl",
	Short: "Build Arch Linux kernels with local module signing",
	Long: `Download kernel packaging files, prepare local signing assets, and build signed Arch Linux kernels

Configuration is read from $XDG_CONFIG_HOME/skm/config, falling back to $HOME/skm/config
If no config is found the following default values will be used:
BUILD_DIR=~/build
MAKEPKG_DIR=/tmp/makepkg`,
}
