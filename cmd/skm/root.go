package main

import (
	"github.com/sk8thing/arch-skm-go/internal/version"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:     "skm",
	Short:   "Sign Kernel Modules",
	Version: version.Version,
	Long:    "Toolkit for signing out-of-tree kernel modules on Arch Linux",
}
