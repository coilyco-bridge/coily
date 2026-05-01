SHELL := /usr/bin/env bash

BIN_NAME        := coily
DEV_BIN_NAME    := coily-dev
INSTALL_PREFIX  := /usr/local
SERVER_HOST     ?= kai-server
SERVER_USER     ?= kai
SERVER_ARCH     ?= arm64

# Windows install target. C:\Program Files\coily is the admin-write-required
# equivalent of a root-owned /usr/local/bin on unix: an agent running as the
# user cannot overwrite the binary without UAC elevation. MSYS-style path so
# Git Bash can `mkdir -p` / `cp` without path mangling.
WINDOWS_INSTALL_DIR ?= /c/Program Files/coily
WINDOWS_ARCH        ?= amd64

VERSION := $(shell git rev-parse --short HEAD 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.Version=$(VERSION)

GO := go
# gotest is a drop-in for `go test` that colorizes PASS/FAIL lines. Install
# with `go install github.com/rakyll/gotest@latest`. CI overrides this back
# to `go test` since color codes are noise in Actions logs.
GO_TEST ?= gotest

.PHONY: help
help:
	@echo "Targets:"
	@echo "  make dev              build ./bin/$(DEV_BIN_NAME) (dev name, not on PATH)"
	@echo "  make build            build ./bin/$(BIN_NAME) (prod tags, for install)"
	@echo "  make install          sudo-install ./bin/$(BIN_NAME) to $(INSTALL_PREFIX)/bin"
	@echo "  make install-windows  build + install bin/$(BIN_NAME).exe to $(WINDOWS_INSTALL_DIR) (run from elevated shell)"
	@echo "  make deploy-server    cross-compile + scp + sudo-install on $(SERVER_HOST)"
	@echo "  make test             $(GO_TEST) ./..."
	@echo "  make vet              go vet ./..."
	@echo "  make clean            remove build outputs"

.PHONY: dev
dev:
	@mkdir -p bin
	$(GO) build -tags dev -ldflags "$(LDFLAGS)" -o bin/$(DEV_BIN_NAME) ./cmd/coily
	@echo "built bin/$(DEV_BIN_NAME) — invoke via ./bin/$(DEV_BIN_NAME) from the repo root only"

.PHONY: build
build:
	@mkdir -p bin
	$(GO) build -tags prod -ldflags "$(LDFLAGS)" -o bin/$(BIN_NAME) ./cmd/coily

.PHONY: install
install:
	@# Block on Windows (Git Bash reports MINGW*/MSYS*/CYGWIN*). The unix
	@# install path needs `sudo install` + /usr/local/bin, which silently
	@# no-ops or lands somewhere unreachable on Windows. The Windows-native
	@# target writes to C:\Program Files\coily and updates User PATH.
	@case "$$(uname -s)" in \
		MINGW*|MSYS*|CYGWIN*) \
			echo "ERROR: 'make install' is the macOS/Linux path. On Windows use 'make install-windows' (run from an elevated Git Bash)."; \
			exit 1 ;; \
	esac
	@$(MAKE) build
	sudo install -o root -g wheel -m 0755 bin/$(BIN_NAME) $(INSTALL_PREFIX)/bin/$(BIN_NAME)
	@echo "installed $(INSTALL_PREFIX)/bin/$(BIN_NAME) (version $(VERSION))"

.PHONY: install-windows
install-windows:
	@# Windows analog of `make install`. C:\Program Files\coily is admin-write
	@# required, same ACL story as /usr/local/bin being root-owned on unix -
	@# see SECURITY.md for the reasoning. Run this from an elevated
	@# shell (Run as Administrator) or mkdir below will fail.
	@mkdir -p bin
	GOOS=windows GOARCH=$(WINDOWS_ARCH) $(GO) build -tags prod -ldflags "$(LDFLAGS)" -o bin/$(BIN_NAME).exe ./cmd/coily
	@mkdir -p "$(WINDOWS_INSTALL_DIR)" 2>/dev/null || { echo "ERROR: could not create $(WINDOWS_INSTALL_DIR)."; echo "  Run 'make install-windows' from an elevated shell (Start > type 'Git Bash' > Ctrl+Shift+Enter, or a PowerShell 'Run as Administrator')."; exit 1; }
	cp bin/$(BIN_NAME).exe "$(WINDOWS_INSTALL_DIR)/$(BIN_NAME).exe"
	@echo "installed $(WINDOWS_INSTALL_DIR)/$(BIN_NAME).exe (version $(VERSION))"
	@echo "if coily is not on PATH yet: add 'C:\\Program Files\\coily' to your user or system PATH (once)."

.PHONY: deploy-server
deploy-server:
	@mkdir -p bin
	GOOS=linux GOARCH=$(SERVER_ARCH) $(GO) build -tags prod -ldflags "$(LDFLAGS)" -o bin/$(BIN_NAME)-linux-$(SERVER_ARCH) ./cmd/coily
	scp bin/$(BIN_NAME)-linux-$(SERVER_ARCH) $(SERVER_USER)@$(SERVER_HOST):/tmp/$(BIN_NAME)
	ssh $(SERVER_USER)@$(SERVER_HOST) 'sudo install -o root -g root -m 0755 /tmp/$(BIN_NAME) /usr/local/bin/$(BIN_NAME) && rm /tmp/$(BIN_NAME)'
	@echo "deployed $(VERSION) to $(SERVER_HOST)"

.PHONY: test
test:
	$(GO_TEST) ./...

.PHONY: test-integration
test-integration:
	@# Layer 2 integration tests. Shell out to live aws/gh/kubectl via
	@# whoami-style verbs. Requires these binaries to be on PATH.
	$(GO_TEST) -tags integration ./test/integration/...

.PHONY: vet
vet:
	$(GO) vet ./...

.PHONY: lint
lint:
	golangci-lint run ./...

.PHONY: lint-fix
lint-fix:
	golangci-lint run --fix ./...

.PHONY: cover
cover:
	$(GO_TEST) -coverprofile=coverage.out ./...
	$(GO) tool cover -func=coverage.out | tail -20
	@echo "HTML report: go tool cover -html=coverage.out"

.PHONY: clean
clean:
	rm -rf bin dist
