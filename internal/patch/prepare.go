package patch

import (
	"fmt"
	"regexp"
)

const prefix = `  # ` + SigningMarker + `
  #
  ######################################################
  # this is added at the start of prepare() & replaces #
  # 'cd $_srcname'                                     #
  ######################################################

  msg2 "Rebuilding local signing key..."

  local certs_local=$srcdir/certs-local
  if [[ ! -d "$startdir/certs-local" ]]; then
    error "Missing $startdir/certs-local; run 'skmctl update <kernel>' first"
    exit 1
  fi

  rm -rf $certs_local
  cp -a $startdir/certs-local $certs_local
  cd $certs_local

  msg2 "Updating kernel config with new key..."
  ./skm gen-keys '../config*'

  cd ../$_srcname
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
	block = rePrepare.ReplaceAllLiteralString(block, "prepare() {\n"+prefix)

	updatedContent := content[:start] + block + content[end:]

	return updatedContent, nil
}
