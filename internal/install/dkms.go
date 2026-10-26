package install

import (
	"os"
	"path/filepath"
)

const kernelSignScript = `#!/bin/bash
SIGN=/usr/lib/modules/$kernelver/build/certs-local/skm

if [ -f $SIGN ] ;then
    $SIGN sign -d ../$kernelver/$arch/module/
else
   echo "kernel $kernelver doesn't have out of tree module signing tools"
fi
`

const kernelSignConf = `# 
# Installed as /etc/dkms/kernel-sign.conf
#
# link this to any module to be signed
#
# e.g. ln -s /etc/dkms/kernel-sign.conf /etc/dkms/virtualbox.conf

POST_BUILD=../../../../../../etc/dkms/kernel-sign.sh
`

func copyDkmsFiles(path string) error {
	dkmsPath := filepath.Join(path, "dkms")
	err := os.MkdirAll(dkmsPath, 0755)
	if err != nil {
		return err
	}

	err = os.WriteFile(filepath.Join(dkmsPath, "kernel-sign.sh"), []byte(kernelSignScript), 0755)
	if err != nil {
		return err
	}

	err = os.WriteFile(filepath.Join(dkmsPath, "kernel-sign.conf"), []byte(kernelSignConf), 0644)
	if err != nil {
		return err
	}

	return nil
}
