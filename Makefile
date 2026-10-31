BINDIR   ?= bin
GO       ?= go
GOFMT    ?= gofmt
GOFLAGS  ?=
LDFLAGS  ?= -s -w
PACKAGES ?= ./cmd/... ./internal/...

SKM    := $(BINDIR)/skm
SKMCTL := $(BINDIR)/skmctl

all: build

build:
	@mkdir -p $(BINDIR)
	$(GO) build -trimpath $(GOFLAGS) -ldflags '$(LDFLAGS)' -o $(SKM) ./cmd/skm
	$(GO) build -trimpath $(GOFLAGS) -ldflags '$(LDFLAGS)' -o $(SKMCTL) ./cmd/skmctl

install: build
	install -Dm755 $(SKMCTL) $(DESTDIR)/usr/bin/skmctl
	install -Dm755 $(SKM) $(DESTDIR)/usr/lib/skm/skm

uninstall:
	$(RM) $(DESTDIR)/usr/bin/skmctl
	$(RM) $(DESTDIR)/usr/lib/skm/skm

test:
	$(GO) test ./...

vet:
	$(GO) vet $(PACKAGES)

check: vet test

fmt:
	$(GOFMT) -s -w cmd internal

clean:
	$(RM) -r $(BINDIR)

.PHONY: all build install uninstall test vet check fmt clean
