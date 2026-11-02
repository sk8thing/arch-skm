# Arch-SKM

**Automated Linux kernel building with Signed Kernel Modules** (in-tree & out-of-tree) for Arch Linux.

A Go rewrite of [Arch-SKM](https://github.com/itoffshore/Arch-SKM) / [Gene's Arch-SKM](https://github.com/gene-git/Arch-SKM), providing the same local module signing workflow with small improvements.

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

[Signed kernel modules](https://www.kernel.org/doc/html/v5.13/admin-guide/module-signing.html) allow the kernel to verify the integrity of loadable modules at runtime. This project provides a complete workflow:

1. Clone the official Arch Linux kernel packaging repo
2. Patch the PKGBUILD to enable local module signing
3. Generate ECDSA key pairs & inject them into the kernel config
4. Build the kernel with `makepkg`
5. Install the signed kernel and its packages

---

## Architecture

The project provides two binaries:

| Binary | Purpose |
|---|---|
| `skmctl` | High-level workflow: update → build → install |
| `skm` | Low-level signing toolkit: gen-keys, sign, install-certs |

### `skmctl` — Kernel Build Workflow

```
skmctl update linux-hardened   # Clone, patch, and prepare build dir
skmctl build linux-hardened    # Build with makepkg
skmctl install linux-hardened  # Install with pacman
```

#### `skmctl update [kernel]`

Clones or updates the kernel packaging repo, patches the PKGBUILD for local module signing, and seeds a `certs-local/` directory with signing keys and DKMS scripts.

```bash
# Clone and prepare linux-hardened kernel
skmctl update linux-hardened

# Force re-clone (removes existing build directory)
skmctl update linux-hardened --force
```

#### `skmctl build [kernel] [-- <makepkg args>...]`

Runs `makepkg` for a prepared kernel build directory. Extra arguments after `--` are passed through to `makepkg`.

```bash
# Build linux-hardened kernel
skmctl build linux-hardened

# Clean build
skmctl build linux-hardened -C

# Pass extra makepkg flags
skmctl build linux-hardened -- -j$(nproc)
```

| Flag | Default | Description |
|---|---|---|
| `-s, --syncdeps` | `true` | Install missing dependencies with pacman |
| `-C, --cleanbuild` | `false` | Remove existing source directory before building |
| `--noconfirm` | `false` | Skip confirmation prompts |

#### `skmctl install [kernel]`

Installs the built kernel packages with `pacman -U`. Automatically overwrites `/etc/dkms/kernel-sign.conf` and `/etc/dkms/kernel-sign.sh` so DKMS can sign future out-of-tree modules.

```bash
# Install linux-hardened kernel
skmctl install linux-hardened

# Skip confirmation prompt
skmctl install linux-hardened --noconfirm
```

### `skm` — Signing Toolkit

```bash
skm gen-keys './config*'     # Generate key pair for kernel configs
skm sign ./module.ko         # Sign a kernel module
skm install-certs ./certs-local  # Install keys and helper binary
```

#### `skm gen-keys [config]`

Generates an ECDSA key pair, creates a self-signed certificate, and updates all matching kernel config files (`CONFIG_SYSTEM_TRUSTED_KEYS`). Keys are stored in `certs-local/keys/` alongside the binary.

```bash
# Generate keys for a single config
skm gen-keys ./config.x86_64

# Generate keys for multiple configs (wildcard)
skm gen-keys './config*'
```

#### `skm sign [modules...]`

Signs out-of-tree kernel modules using the local private key. Supports individual `.ko` files, compressed modules (`.zst`, `.gz`), or an entire directory.

```bash
# Sign a single module
skm sign ./module.ko

# Sign multiple modules
skm sign ./module1.ko ./module2.ko.zst ./module3.ko.gz

# Sign all modules in a directory
skm sign --dir ./modules

# Limit concurrency (default: CPU count)
skm sign --dir ./modules --jobs 4
```

#### `skm install-certs [destination]`

Copies the signing keys and the `skm` binary itself into a `certs-local/` directory, ready to be included in a kernel build.

```bash
# Install keys into certs-local
skm install-certs ./certs-local
```

---

## Configuration

Configuration is read from `$XDG_CONFIG_HOME/skm/config`, falling back to `$HOME/skm/config`.

```ini
BUILD_DIR = ~/build
MAKEPKG_DIR = /tmp/makepkg
```

If no config file is found, the defaults above are used.

---

## How It Works

### In-tree modules

During a standard kernel build, the build tools create a fresh ECDSA key pair and sign every in-tree module. The public key is embedded into the kernel image.

### Out-of-tree modules

Out-of-tree modules (DKMS, AUR packages, manual builds) are signed separately using the same key pair:

1. `skm gen-keys` generates a key pair and updates the kernel's `CONFIG_SYSTEM_TRUSTED_KEYS` to point at the public cert.
2. `skm install-certs` places keys and DKMS scripts into `certs-local/`.
3. The patched PKGBUILD includes `certs-local/` in the kernel build, embedding the public key.
4. After installation, `certs-local/dkms/kernel-sign.conf` is linked into `/etc/dkms/`, so DKMS automatically signs new out-of-tree modules.

### PKGBUILD Patching

The `skmctl update` command patches the PKGBUILD's `prepare()` and `_package_headers()` functions to:

- Run `skm gen-keys` to generate fresh keys on each build
- Copy `certs-local/` into the kernel build tree
- Install the signing public key into `/usr/lib/modules/$(uname -r)/certs-local/` via `_package_headers()`

---

## Building from Source

```bash
make
```

This produces two binaries in `bin/`:

| Binary | Path |
|---|---|
| `skmctl` | `bin/skmctl` |
| `skm` | `bin/skm` |

### Install

```bash
make install
```

Installs `skmctl` to `/usr/bin/skmctl` and `skm` to `/usr/lib/skm/skm`.

### Uninstall

```bash
make uninstall
```

---

## Kernel Configuration

The following kernel config options should be set (the PKGBUILD patch handles `CONFIG_SYSTEM_TRUSTED_KEYS` automatically):

```
CONFIG_MODULE_SIG=y              # Module signature verification
CONFIG_MODULE_SIG_FORCE=n        # Do not force signature enforcement at compile time
CONFIG_MODULE_SIG_HASH=sha512    # Hash algorithm
CONFIG_MODULES_COMPRESS=zstd     # Compress modules on installation
CONFIG_MODULE_SIG_ALL=y          # Automatically sign all modules
```

### Enforcing Module Signatures

Once you're confident everything works, enable runtime enforcement via the kernel command line:

```
module.sig_enforce=1
```

---

## DKMS Setup

After installing a kernel built with this tool, DKMS will automatically sign out-of-tree modules. The patched PKGBUILD installs:

```
/etc/dkms/kernel-sign.conf   # DKMS configuration
/etc/dkms/kernel-sign.sh     # Signing script
```

For each DKMS package (e.g., `nvidia`, `zfs`), a symlink has to be created:

```bash
sudo ln -s /etc/dkms/kernel-sign.conf /etc/dkms/<module>
```

---

## License

See [LICENSE](LICENSE) for details.

This project is a Go port of [Arch-SKM](https://github.com/itoffshore/Arch-SKM) and [Gene's Arch-SKM](https://github.com/gene-git/Arch-SKM).
