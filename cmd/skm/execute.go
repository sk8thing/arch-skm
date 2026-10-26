package skm

import "os"

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
