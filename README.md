# Arch-SKM — WORK IN PROGRESS 🛠️

**Automated Linux kernel building with Signed Kernel Modules** (in-tree & out-of-tree) for Arch Linux.

A Go rewrite of [Arch-SKM](https://github.com/itoffshore/Arch-SKM) / [Gene's Arch-SKM](https://github.com/gene-git/Arch-SKM), providing the same local module signing workflow with a few improvements.

### Supported Kernels

* [linux](https://gitlab.archlinux.org/archlinux/packaging/packages/linux.git)
* [linux-lts](https://gitlab.archlinux.org/archlinux/packaging/packages/linux-lts.git)
* [linux-hardened](https://gitlab.archlinux.org/archlinux/packaging/packages/linux-hardened.git)
* [linux-zen](https://gitlab.archlinux.org/archlinux/packaging/packages/linux-zen.git)
* [linux-rt](https://gitlab.archlinux.org/archlinux/packaging/packages/linux-rt.git)
* [linux-libre](https://aur.archlinux.org/packages/linux-libre)
* [linux-xanmod](https://aur.archlinux.org/packages/linux-xanmod)

---

## Overview

[Signed kernel modules](https://wiki.archlinux.org/title/Signed_kernel_modules) allow the kernel to verify the integrity of loadable modules at runtime. This project provides a complete workflow:

1. Clone the official Arch Linux kernel packaging repo
2. Patch the PKGBUILD to enable local module signing
3. Generate ECDSA key pairs & inject them into the kernel config
4. Build the kernel with `makepkg`
5. Install the signed kernel and its packages

### Architecture

The project provides two binaries:

| Binary | Purpose |
|---|---|
| `skmctl` | High-level workflow: update → build → install |
| `skm` | Low-level toolkit: gen-keys, sign, install-certs |

#### `skmctl` — Kernel Build Workflow

| Command | Function |
|---|---|
| `skmctl update [kernel]` | Clone kernel, patch it, and prepare build directory |
| `skmctl build [kernel]` | Build with makepkg |
| `skmctl install [kernel]` | Install using pacman |


---

## Installation

### Building from Source

```bash
make
```

This produces two binaries in `bin/`:

| Binary | Path |
|---|---|
| `skmctl` | `bin/skmctl` |
| `skm` | `bin/skm` |

Shell completions for bash, zsh, and fish are generated separately:

```bash
make completions
```

They are written to `bin/completions/` as `skmctl` (bash), `_skmctl` (zsh), and `skmctl.fish` (fish).

```bash
sudo make install
```

Installs `skmctl` to `/usr/bin/skmctl`, `skm` to `/usr/lib/skm/skm`, and the completions to `/usr/share/bash-completion/completions/skmctl`, `/usr/share/zsh/site-functions/_skmctl`, and `/usr/share/fish/vendor_completions.d/skmctl.fish`. The target runs `make completions` itself, so generating them beforehand is optional.

```bash
sudo make uninstall
```

Removes the binaries and the installed completion files.

---

## Configuration

Configuration is read from `$XDG_CONFIG_HOME/skm/skm.conf`, falling back to `$HOME/.config/skm/skm.conf`.

```ini
BUILD_DIR = ~/build
MAKEPKG_DIR = /tmp/makepkg
```

If no config file is found, the defaults above are used.

---

## DKMS Setup

DKMS signs out-of-tree modules through a `POST_BUILD` hook. A kernel built with this tool ships that hook in its package:

| File | Purpose |
|---|---|
| `/etc/dkms/kernel-sign.conf` | Enables the hook for one DKMS package (`POST_BUILD`) |
| `/etc/dkms/kernel-sign.sh` | Signs the built modules with the kernel's out-of-tree key |

`skmctl install [kernel]` overwrites both files from the kernel package, so the hook always matches the key embedded in that kernel.
DKMS only reads `/etc/dkms/<package>.conf`, so signing must be opted into per package by symlinking that file under the package name.

> **Prerequisite:** The `dkms` package must be installed for `skmctl dkms status` and `skmctl dkms rebuild` to work.

### Subcommands

| Command | Function |
|---|---|
| `skmctl dkms status` | List DKMS modules, their signing state, and dangling configurations |
| `skmctl dkms install [modules…]` | Enable signing for DKMS packages |
| `skmctl dkms remove [modules…]` | Disable signing for DKMS packages |
| `skmctl dkms rebuild [modules…]` | Rebuild DKMS modules for all kernels |

### Checking signing status

```console
$ skmctl dkms status
nvidia: installed
vboxhost: configured
acpi_call: dangling
```

Each line shows a unique DKMS package name and one of three states:

| State | Meaning |
|---|---|
| `installed` | The module is installed but signing is not enabled |
| `configured` | Signing is enabled (the `/etc/dkms/<package>.conf` symlink exists) |
| `dangling` | A signing symlink exists but `dkms status` has no matching module (the package was probably uninstalled) |


### Enabling signing

```bash
skmctl dkms install nvidia
```

This creates `/etc/dkms/nvidia.conf` as a symlink to `/etc/dkms/kernel-sign.conf`. Multiple packages can be configured at once:

```bash
skmctl dkms install nvidia vboxhost
```

The command accepts any DKMS package name, so signing can be configured before the package itself is installed. If a symlink already exists and points to the correct target, the command reports `already configured` and skips it.

To disable signing for a package, remove its configuration:

```bash
skmctl dkms remove nvidia
```

This also works for `dangling` entries, removing symlinks left behind by packages that are no longer installed.

### Rebuilding modules

After enabling signing for a package, rebuild it against all installed kernels:

```bash
skmctl dkms rebuild nvidia
```

To rebuild all installed modules:

```bash
skmctl dkms rebuild --all
```

### Verifying the signature

```bash
modinfo -F signer nvidia
```

A module signed with this kernel's key reports:

```
Local Out of tree kernel module signing key
```

An empty or different value means the module was not signed with that key.

### Enforce signature verification

Once you're confident everything works, runtime enforcement can be enabled via the kernel command line:

```
module.sig_enforce=1
```
