package main

import (
	"fmt"
	"path/filepath"

	"github.com/sk8thing/arch-skm/internal/certgen"
	"github.com/sk8thing/arch-skm/internal/helper"
	"github.com/sk8thing/arch-skm/internal/kconfig"
	"github.com/sk8thing/arch-skm/internal/keygen"
	"github.com/spf13/cobra"
)

const genKeysExample = `  # Read config from current directory
  skm gen-keys ./config.x86_64

  # Use quotes for wildcards
  skm gen-keys './config*'`

var genKeysCmd = &cobra.Command{
	Use:     "gen-keys [config]",
	Short:   "Generate new key pair",
	Example: genKeysExample,
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		configGlob := args[0]
		matches, err := filepath.Glob(configGlob)
		if err != nil || len(matches) == 0 {
			return fmt.Errorf("no kernel config files found matching: %s", configGlob)
		}

		exePath, err := helper.ExecutablePath()
		if err != nil {
			return fmt.Errorf("cannot get executable path: %w", err)
		}
		exeDir := filepath.Dir(exePath)

		configs, err := loadConfigs(matches)
		if err != nil {
			return err
		}
		refConfig := configs[0]

		keys, err := keygen.GenerateKeyPair(refConfig)
		if err != nil {
			return fmt.Errorf("failed to generate keys: %w", err)
		}

		cert, err := certgen.GenerateCertificate(keys, refConfig)
		if err != nil {
			return fmt.Errorf("failed to generate certificate: %w", err)
		}

		trustedKeysPath, err := certgen.ExportCertificate(exeDir, cert, keys, refConfig)
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
}

func loadConfigs(matches []string) ([]*kconfig.KConfig, error) {
	configs := make([]*kconfig.KConfig, 0, len(matches))

	for _, match := range matches {
		config, err := kconfig.NewFromFile(match)
		if err != nil {
			return nil, fmt.Errorf("failed to parse config %s: %w", match, err)
		}

		configs = append(configs, config)

		ref := configs[0]
		if config.KeyType() != ref.KeyType() {
			return nil, fmt.Errorf("key types must be same across configs: %s vs %s", ref.KeyType(), config.KeyType())
		}
		if config.HashType() != ref.HashType() {
			return nil, fmt.Errorf("hash types must be same across configs: %s vs %s", ref.HashType(), config.HashType())
		}
	}

	return configs, nil
}
