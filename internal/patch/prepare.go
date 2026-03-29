package patch

import (
	"fmt"
	"regexp"
)

const prefix = `  # Out-of-tree module signing
  #
  ######################################################
  # this is added at the start of prepare() & replaces #
  # 'cd $$_srcname'                                     #
  ######################################################

  msg2 "Rebuilding local signing key..."

  cd certs-local

  msg2 "Updating kernel config with new key..."
  ./skm gen-keys --config '../config*'

  cd ../$$_srcname
`

var (
	rePrepare = regexp.MustCompile(`(?m)^prepare\(\)\s*\{\n`)
	reCd      = regexp.MustCompile(`(?m)^\s*cd\s+\$_srcname\s*$`)
)

func patchPrepareFunction(content string) (string, error) {
	block, start, end, err := getFunctionBlock("prepare", content)
	if err != nil {
		return "", fmt.Errorf("failed to read prepare function block: %w", err)
	}

	block = commentLine(block, reCd)
	block = rePrepare.ReplaceAllString(block, "prepare() {\n"+prefix)

	updatedContent := content[:start] + block + content[end:]

	return updatedContent, nil
}
