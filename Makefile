BINDIR      ?= bin
COMPLETION_DIR := $(BINDIR)/completions
GO          ?= go
GOFMT       ?= gofmt
GOFLAGS     ?=
GO_LDFLAGS     ?= -s -w
PACKAGES    ?= ./cmd/... ./internal/...
VERSION     ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
VERSION_PKG := github.com/sk8thing/arch-skm/internal/version

VERSION_LDFLAGS := -X $(VERSION_PKG).Version=$(VERSION)

SKM    := $(BINDIR)/skm
SKMCTL := $(BINDIR)/skmctl

all: build

build:
	@mkdir -p $(BINDIR)
	$(GO) build -trimpath $(GOFLAGS) -ldflags '$(GO_LDFLAGS) $(VERSION_LDFLAGS)' -o $(SKM) ./cmd/skm
	$(GO) build -trimpath $(GOFLAGS) -ldflags '$(GO_LDFLAGS) $(VERSION_LDFLAGS)' -o $(SKMCTL) ./cmd/skmctl

completions: build
	@mkdir -p $(COMPLETION_DIR)
	$(SKMCTL) completion bash > $(COMPLETION_DIR)/skmctl
	$(SKMCTL) completion zsh > $(COMPLETION_DIR)/_skmctl
	$(SKMCTL) completion fish > $(COMPLETION_DIR)/skmctl.fish

install: build completions
	install -Dm755 $(SKMCTL) $(DESTDIR)/usr/bin/skmctl
	install -Dm755 $(SKM) $(DESTDIR)/usr/lib/skm/skm
	install -Dm644 $(COMPLETION_DIR)/skmctl $(DESTDIR)/usr/share/bash-completion/completions/skmctl
	install -Dm644 $(COMPLETION_DIR)/_skmctl $(DESTDIR)/usr/share/zsh/site-functions/_skmctl
	install -Dm644 $(COMPLETION_DIR)/skmctl.fish $(DESTDIR)/usr/share/fish/vendor_completions.d/skmctl.fish

uninstall:
	$(RM) $(DESTDIR)/usr/bin/skmctl
	$(RM) $(DESTDIR)/usr/lib/skm/skm
	$(RM) $(DESTDIR)/usr/share/bash-completion/completions/skmctl
	$(RM) $(DESTDIR)/usr/share/zsh/site-functions/_skmctl
	$(RM) $(DESTDIR)/usr/share/fish/vendor_completions.d/skmctl.fish

test:
	$(GO) test $(PACKAGES)

race:
	$(GO) test -race $(PACKAGES)

vet:
	$(GO) vet $(PACKAGES)

check: fmtcheck vet test race

fmt:
	$(GOFMT) -s -w cmd internal

fmtcheck:
	@checks=$$($(GOFMT) -s -l cmd internal); if [ -n "$$checks" ]; then \
		echo "not gofmt-cleaned, run 'make fmt' in:"; echo "$$checks"; exit 1; fi

clean:
	$(RM) -r $(BINDIR)

.PHONY: all build completions install uninstall test race vet check fmtcheck fmt clean
