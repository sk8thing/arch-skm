package patch

import (
	"fmt"
	"regexp"
)

const preparePrefix = `  # ` + SigningMarker + `
  #
  ######################################################
  # this is added at the start of prepare()            #
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

  cd $srcdir
`

const prepareSuffix = `
  # ` + SigningMarker + `
  #
  ######################################################
  # this is added at the end of prepare()              #
  # for kernels that generate their config in prepare  #
  ######################################################

  msg2 "Rebuilding local signing key..."

  local certs_local=certs-local
  if [[ ! -d "$startdir/certs-local" ]]; then
    error "Missing $startdir/certs-local; run 'skmctl update <kernel>' first"
    exit 1
  fi

  rm -rf $certs_local
  cp -a $startdir/certs-local $certs_local
  cd $certs_local

  msg2 "Updating kernel config with new key..."
  ./skm gen-keys '../.config'

  cd ..
`

var rePrepare = regexp.MustCompile(`(?m)^prepare\(\)\s*\{\n`)

func patchPrepareFunction(content string, configAtStart bool) (string, error) {
	block, start, end, err := getFunctionBlock("prepare", content)
	if err != nil {
		return "", fmt.Errorf("failed to read prepare function block: %w", err)
	}

	if configAtStart {
		block = rePrepare.ReplaceAllLiteralString(block, "prepare() {\n"+preparePrefix)
	} else {
		block = block[:len(block)-1] + prepareSuffix + "\n}"
	}

	updatedContent := content[:start] + block + content[end:]

	return updatedContent, nil
}
