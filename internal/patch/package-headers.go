package patch

import (
	"fmt"
	"regexp"
	"strings"
)

const suffix = `
  # ` + SigningMarker + `
  ##################################################
  # this is added at the end of _package-headers() #
  ##################################################
  # This is run in the kernel source / build directory
  #
  # some AUR kernels may also need to set $builddir to match
  # the Package Maintainer's variable for the build dir
  # inside the package:
  #
  # local builddir="$pkgdir/usr/lib/modules/${_kernver}/build"

  msg2 "Local Signing certs for out-of-tree modules..."

  certs_local_src="../certs-local"
  certs_local_dst="${builddir}/certs-local"

  # install certificates
  ${certs_local_src}/skm install-certs $certs_local_dst

  # install dkms tools
  dkms_src="$certs_local_src/dkms"
  dkms_dst="${pkgdir}/etc/dkms"
  mkdir -p $dkms_dst

  rsync -a $dkms_src/{kernel-sign.conf,kernel-sign.sh} $dkms_dst/`

var reBuilddirVar = regexp.MustCompile(`(?m)^\s*(?:local\s+)?(_?builddir)=`)

func patchPackageHeadersFunction(content string, configAtStart bool) (string, error) {
	block, start, end, err := getFunctionBlock("_package-headers", content)
	if err != nil {
		return "", fmt.Errorf("failed to read _package-headers function block: %w", err)
	}

	builddirVar := "builddir"
	if match := reBuilddirVar.FindStringSubmatch(block); match != nil {
		builddirVar = match[1]
	}

	certsSrc := "../certs-local"
	if !configAtStart {
		certsSrc = "./certs-local"
	}

	blockSuffix := strings.NewReplacer(
		"${builddir}", "${"+builddirVar+"}",
		"../certs-local", certsSrc,
	).Replace(suffix)

	updatedBlock := block[:len(block)-1] + blockSuffix + "\n}"
	updatedContent := content[:start] + updatedBlock + content[end:]

	return updatedContent, nil
}
