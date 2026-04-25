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

.PHONY: help
help:
	@echo "Targets:"
	@echo "  make dev              build ./bin/$(DEV_BIN_NAME) (dev name, not on PATH)"
	@echo "  make build            build ./bin/$(BIN_NAME) (prod tags, for install)"
	@echo "  make install          sudo-install ./bin/$(BIN_NAME) to $(INSTALL_PREFIX)/bin"
	@echo "  make install-windows  build + install bin/$(BIN_NAME).exe to $(WINDOWS_INSTALL_DIR) (run from elevated shell)"
	@echo "  make deploy-server    cross-compile + scp + sudo-install on $(SERVER_HOST)"
	@echo "  make scope-aws        run subcli-scope against aws"
	@echo "  make scope-gh         run subcli-scope against gh"
	@echo "  make scope-kubectl    run subcli-scope against kubectl"
	@echo "  make scope-tailscale  run subcli-scope against tailscale"
	@echo "  make scope-docker     run subcli-scope against docker"
	@echo "  make scope-all        run subcli-scope against all scoped CLIs"
	@echo "  make test             go test ./..."
	@echo "  make vet              go vet ./..."
	@echo "  make clean            remove build outputs"

.PHONY: _sync-config
_sync-config:
	@# //go:embed can only reach files in the same package; mirror the canonical
	@# repo-root config.yaml into pkg/config/ before build. Both are gitignored.
	@# Per-user / per-repo overrides live at ~/.coily/config.yaml and
	@# ./.coily/config.yaml respectively, layered on top of this embedded base.
	@if [ ! -f config.yaml ]; then cp config.example.yaml config.yaml; fi
	@cp config.yaml pkg/config/config.yaml

.PHONY: dev
dev: _sync-config
	@mkdir -p bin
	$(GO) build -tags dev -ldflags "$(LDFLAGS)" -o bin/$(DEV_BIN_NAME) ./cmd/coily
	@echo "built bin/$(DEV_BIN_NAME) — invoke via ./bin/$(DEV_BIN_NAME) from the repo root only"

.PHONY: build
build: _sync-config
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
install-windows: _sync-config
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
deploy-server: _sync-config
	@mkdir -p bin
	GOOS=linux GOARCH=$(SERVER_ARCH) $(GO) build -tags prod -ldflags "$(LDFLAGS)" -o bin/$(BIN_NAME)-linux-$(SERVER_ARCH) ./cmd/coily
	scp bin/$(BIN_NAME)-linux-$(SERVER_ARCH) $(SERVER_USER)@$(SERVER_HOST):/tmp/$(BIN_NAME)
	ssh $(SERVER_USER)@$(SERVER_HOST) 'sudo install -o root -g root -m 0755 /tmp/$(BIN_NAME) /usr/local/bin/$(BIN_NAME) && rm /tmp/$(BIN_NAME)'
	@echo "deployed $(VERSION) to $(SERVER_HOST)"

.PHONY: scope-aws
scope-aws:
	$(GO) run ./cmd/subcli-scope aws

.PHONY: scope-gh
scope-gh:
	$(GO) run ./cmd/subcli-scope gh

.PHONY: scope-kubectl
scope-kubectl:
	$(GO) run ./cmd/subcli-scope kubectl

.PHONY: scope-tailscale
scope-tailscale:
	$(GO) run ./cmd/subcli-scope tailscale

.PHONY: scope-docker
scope-docker:
	$(GO) run ./cmd/subcli-scope docker

.PHONY: scope-all
scope-all: scope-aws scope-gh scope-kubectl scope-tailscale scope-docker

.PHONY: gen-passthrough
gen-passthrough:
	@# Regenerate pkg/ops/{aws,gh,kubectl}/generated.go from configs/commands/*.yaml.
	@# Run after update-fixtures or whenever the command manifests change.
	$(GO) run ./cmd/gen-passthrough all
	gofmt -w pkg/ops/

.PHONY: update-fixtures
update-fixtures:
	@# Recapture help-text fixtures from live aws/gh/kubectl into cmd/subcli-scope/testdata/fixtures/.
	@# Refresh the goldens AND the per-tool classification snapshots
	@# (cmd/subcli-scope/testdata/<tool>.classified.txt), then regenerate
	@# the pass-through and the skill. Review the resulting diff before
	@# committing - in particular the .classified.txt diff is the place a
	@# new MUTATING verb mis-labeled READONLY will show up.
	$(GO) run ./cmd/subcli-scope -capture cmd/subcli-scope/testdata/fixtures gh
	$(GO) run ./cmd/subcli-scope -capture cmd/subcli-scope/testdata/fixtures kubectl
	$(GO) run ./cmd/subcli-scope -capture cmd/subcli-scope/testdata/fixtures aws
	$(GO) run ./cmd/subcli-scope -capture cmd/subcli-scope/testdata/fixtures tailscale
	$(GO) run ./cmd/subcli-scope -capture cmd/subcli-scope/testdata/fixtures docker
	$(GO) test ./cmd/subcli-scope -update
	$(MAKE) gen-passthrough
	$(MAKE) skill
	@echo "fixtures + goldens + classified snapshots + pass-through + skill refreshed. diff and commit if you approve the changes."

.PHONY: skill
skill: dev
	@# Regenerate skill/SKILL.md and skill/reference/*.md from configs/commands/*.yaml.
	@# Uses the dev binary because skill-gen is a dev-only subcommand (not in prod).
	./bin/$(DEV_BIN_NAME) skill-gen

.PHONY: install-skill
install-skill: skill
	@# Thin wrapper over the dev-only `coily install-skill` subcommand.
	./bin/$(DEV_BIN_NAME) install-skill --force

.PHONY: test
test:
	$(GO) test ./...

.PHONY: test-integration
test-integration:
	@# Layer 2 integration tests. Shell out to live aws/gh/kubectl via
	@# whoami-style verbs. Requires these binaries to be on PATH.
	$(GO) test -tags integration ./test/integration/...

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
	$(GO) test -coverprofile=coverage.out ./...
	$(GO) tool cover -func=coverage.out | tail -20
	@echo "HTML report: go tool cover -html=coverage.out"

.PHONY: clean
clean:
	rm -rf bin dist
