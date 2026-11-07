package kconfig

import (
	"strings"
)

const trustedKeys = "CONFIG_SYSTEM_TRUSTED_KEYS="

func updateTrustedKeys(line string, value string) (string, bool) {
	if !strings.HasPrefix(line, trustedKeys) {
		return line, false
	}
	return trustedKeysValue(value), true
}

func trustedKeysValue(value string) string {
	return trustedKeys + `"` + value + `"`
}
