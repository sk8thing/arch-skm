package kconfig

import (
	"fmt"
	"strings"
)

const trustedKeys = "CONFIG_SYSTEM_TRUSTED_KEYS="

func updateTrustedKeys(line string, value string) (string, bool) {
	if !strings.HasPrefix(line, trustedKeys) {
		return line, false
	}
	return trustedKeys + fmt.Sprintf(`"%s"`, value), true
}
