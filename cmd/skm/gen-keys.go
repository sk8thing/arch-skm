package main

import (
	"fmt"
	"path/filepath"

	"github.com/sk8thing/arch-skm-go/internal/certgen"
	"github.com/sk8thing/arch-skm-go/internal/kconfig"
	"github.com/sk8thing/arch-skm-go/internal/keygen"
	"github.com/sk8thing/arch-skm-go/internal/utils"
	"github.com/spf13/cobra"
)

var genKeysCmd = &cobra.Command{
	Use:   "gen-keys",
	Short: "Generate new key pair for signing out of tree kernel modules",
	RunE: func(cmd *cobra.Command, args []string) error {
		configGlob, _ := cmd.Flags().GetString("config")
		matches, err := filepath.Glob(configGlob)
		if err != nil || len(matches) == 0 {
			return fmt.Errorf("no kernel config files found matching: %s", configGlob)
		}

		exePath, err := utils.GetExecutablePath()
		if err != nil {
			return fmt.Errorf("cannot get executable path: %w", err)
		}
		path := filepath.Dir(exePath)

		var refConfig *kconfig.KConfig
		configs := make([]*kconfig.KConfig, 0, len(matches))
		for _, path := range matches {
			config, err := kconfig.NewFromFile(path)
			if err != nil {
				return fmt.Errorf("failed to parse config %s: %w", path, err)
			}

			configs = append(configs, config)

			if refConfig == nil {
				refConfig = config
				continue
			}
			if config.KeyType() != refConfig.KeyType() {
				return fmt.Errorf("key types must be same across configs: %s vs %s", refConfig.KeyType(), config.KeyType())
			}
			if config.HashType() != refConfig.HashType() {
				return fmt.Errorf("hash types must be same across configs: %s vs %s", refConfig.HashType(), config.HashType())
			}
		}

		keys, err := keygen.GenerateKeyPair(refConfig)
		if err != nil {
			return fmt.Errorf("failed to generate keys: %w", err)
		}

		cert, err := certgen.GenerateCertificate(keys, refConfig)
		if err != nil {
			return fmt.Errorf("failed to generate certificate: %w", err)
		}

		trustedKeysPath, err := certgen.ExportCertificate(path, cert, keys, refConfig)
		if err != nil {
			return fmt.Errorf("failed to export certificate: %w", err)
		}

		for _, config := range configs {
			if err := config.UpdateTrustedKeys(trustedKeysPath); err != nil {
				return fmt.Errorf("failed to update config trusted keys: %w", err)
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(genKeysCmd)

	genKeysCmd.Flags().StringP("config", "c", "./config", "Kernel config file(s), use quotes for wildcards e.g. '../config*'")
}
