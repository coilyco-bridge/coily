# syntax=docker/dockerfile:1.7

# coily multi-stage image. Three published targets:
#
#   :base   coily + Claude Code + git/python/node + lockdown applied.
#           The minimum agent runtime.
#   :cloud  base + aws/gh/kubectl/helm/terraform/tflint/tfsec/tailscale/
#           gcloud/docker-cli. The ops-session image.
#   :full   cloud + every package manager and runtime coily wraps under
#           `coily pkg` (minus brew - macOS-host concern), plus go, ruby,
#           just, task. The polyglot dev image.
#
# Each stage is `--target`-able and shares layers with the ones below it.
# Build via `make docker-base|docker-cloud|docker-full`.

ARG UBUNTU_VERSION=24.04
ARG GO_VERSION=1.25
ARG NODE_MAJOR=22

# ---------------------------------------------------------------------------
# build: compile coily once, copy the binary into every runtime stage.
# ---------------------------------------------------------------------------
FROM golang:${GO_VERSION}-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG VERSION=docker
RUN CGO_ENABLED=0 go build \
    -tags prod \
    -ldflags "-s -w -X main.Version=${VERSION}" \
    -o /out/coily \
    ./cmd/coily

# ---------------------------------------------------------------------------
# base: Ubuntu 24.04 + minimal agent runtime + Claude Code + lockdown.
# ---------------------------------------------------------------------------
FROM ubuntu:${UBUNTU_VERSION} AS base
ARG NODE_MAJOR
ARG TARGETARCH

ENV DEBIAN_FRONTEND=noninteractive \
    LANG=C.UTF-8 \
    LC_ALL=C.UTF-8

# System packages: read-only utilities the lockdown allow-list expects to
# find on PATH, plus the network primitives Claude Code needs to function.
RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        ca-certificates \
        curl \
        wget \
        gnupg \
        git \
        openssh-client \
        rsync \
        jq \
        ripgrep \
        tree \
        less \
        unzip \
        zip \
        xz-utils \
        make \
        python3 \
        python3-pip \
        python3-venv \
        pipx \
    && rm -rf /var/lib/apt/lists/*

# Node via NodeSource. Claude Code is `npm i -g @anthropic-ai/claude-code`
# and several `coily pkg` pass-throughs (pnpm/yarn/bun) live on top of npm.
RUN curl -fsSL "https://deb.nodesource.com/setup_${NODE_MAJOR}.x" | bash - \
    && apt-get install -y --no-install-recommends nodejs \
    && rm -rf /var/lib/apt/lists/* \
    && npm install -g @anthropic-ai/claude-code

# Non-root user. coily's binary stays root-owned 0755 so an agent running
# as `coily` cannot overwrite it - same property as /usr/local/bin on the
# host install path (see SECURITY.md).
ARG USER_UID=1000
ARG USER_GID=1000
# Ubuntu 24.04 ships a default `ubuntu` user/group at UID/GID 1000.
# Drop it before creating coily so the standard host UID maps cleanly to
# our non-root user for bind-mounted workspaces.
RUN if id -u ubuntu >/dev/null 2>&1; then userdel -r ubuntu; fi \
    && if getent group ubuntu >/dev/null 2>&1; then groupdel ubuntu; fi \
    && groupadd --gid ${USER_GID} coily \
    && useradd --uid ${USER_UID} --gid ${USER_GID} --create-home --shell /bin/bash coily \
    && mkdir -p /workspace \
    && chown coily:coily /workspace

COPY --from=build --chown=root:root --chmod=0755 /out/coily /usr/local/bin/coily

# Apply the lockdown defaults to the user's home so a bare `claude` invocation
# already has the deny-list in place. A bind-mounted ~/.claude from the host
# overrides this, which is the user's prerogative.
USER coily
RUN coily lockdown --apply --path /home/coily \
    && coily --version

WORKDIR /workspace
ENV PATH=/home/coily/.local/bin:${PATH}

# Default to an interactive Claude Code session. Override with any other
# command (`docker run ... ghcr.io/coilysiren/coily:base bash`).
CMD ["claude"]

# ---------------------------------------------------------------------------
# cloud: base + the infra CLIs coily wraps for ops work.
#
# Each tool is fetched as a pinned binary release rather than installed via
# a third-party apt repo. That avoids "is hashicorp/tailscale/k8s/helm's
# noble apt repo up today" as a class of CI failure, and makes the image
# content reproducible by pinned version rather than by whatever the repo
# serves at build time. Bump versions explicitly when a refresh is wanted.
# ---------------------------------------------------------------------------
FROM base AS cloud
USER root
ARG TARGETARCH

ARG GH_VERSION=2.62.0
ARG KUBECTL_VERSION=1.31.3
ARG HELM_VERSION=3.16.3
ARG TERRAFORM_VERSION=1.10.0
ARG TAILSCALE_VERSION=1.78.1
ARG DOCKER_VERSION=27.4.0
ARG TFLINT_VERSION=0.55.1
ARG TFSEC_VERSION=1.28.13
ARG GCLOUD_VERSION=502.0.0

# AWS CLI v2 (official installer; bundles its own python).
RUN set -eux; \
    case "${TARGETARCH}" in \
        amd64) AWS_ARCH=x86_64 ;; \
        arm64) AWS_ARCH=aarch64 ;; \
        *) echo "unsupported arch ${TARGETARCH}" && exit 1 ;; \
    esac; \
    cd /tmp; \
    curl -fsSL "https://awscli.amazonaws.com/awscli-exe-linux-${AWS_ARCH}.zip" -o aws.zip; \
    unzip -q aws.zip; \
    ./aws/install; \
    rm -rf aws aws.zip

# Single-file binaries: gh, kubectl, helm, terraform, tailscale (+ tailscaled),
# docker CLI, tflint, tfsec. Grouped into one RUN to share /tmp staging; each
# command is `set -eux`-traced so a failure points to the specific tool.
RUN set -eux; \
    case "${TARGETARCH}" in \
        amd64) ARCH=amd64; STATIC_ARCH=x86_64 ;; \
        arm64) ARCH=arm64; STATIC_ARCH=aarch64 ;; \
    esac; \
    cd /tmp; \
    \
    # gh
    curl -fsSL "https://github.com/cli/cli/releases/download/v${GH_VERSION}/gh_${GH_VERSION}_linux_${ARCH}.tar.gz" \
        | tar -xz; \
    install -m 0755 "gh_${GH_VERSION}_linux_${ARCH}/bin/gh" /usr/local/bin/gh; \
    rm -rf "gh_${GH_VERSION}_linux_${ARCH}"; \
    \
    # kubectl
    curl -fsSL "https://dl.k8s.io/release/v${KUBECTL_VERSION}/bin/linux/${ARCH}/kubectl" -o /usr/local/bin/kubectl; \
    chmod 0755 /usr/local/bin/kubectl; \
    \
    # helm
    curl -fsSL "https://get.helm.sh/helm-v${HELM_VERSION}-linux-${ARCH}.tar.gz" | tar -xz; \
    install -m 0755 "linux-${ARCH}/helm" /usr/local/bin/helm; \
    rm -rf "linux-${ARCH}"; \
    \
    # terraform
    curl -fsSL "https://releases.hashicorp.com/terraform/${TERRAFORM_VERSION}/terraform_${TERRAFORM_VERSION}_linux_${ARCH}.zip" -o tf.zip; \
    unzip -q tf.zip; \
    install -m 0755 terraform /usr/local/bin/terraform; \
    rm -f terraform tf.zip; \
    \
    # tailscale CLI + daemon. The daemon (tailscaled) only does anything
    # useful with --cap-add=NET_ADMIN at runtime; ship it so the option
    # exists.
    curl -fsSL "https://pkgs.tailscale.com/stable/tailscale_${TAILSCALE_VERSION}_${ARCH}.tgz" | tar -xz; \
    install -m 0755 "tailscale_${TAILSCALE_VERSION}_${ARCH}/tailscale" /usr/local/bin/tailscale; \
    install -m 0755 "tailscale_${TAILSCALE_VERSION}_${ARCH}/tailscaled" /usr/local/sbin/tailscaled; \
    rm -rf "tailscale_${TAILSCALE_VERSION}_${ARCH}"; \
    \
    # docker CLI only (no daemon; the daemon belongs to the host).
    curl -fsSL "https://download.docker.com/linux/static/stable/${STATIC_ARCH}/docker-${DOCKER_VERSION}.tgz" | tar -xz; \
    install -m 0755 docker/docker /usr/local/bin/docker; \
    rm -rf docker; \
    \
    # tflint
    curl -fsSL "https://github.com/terraform-linters/tflint/releases/download/v${TFLINT_VERSION}/tflint_linux_${ARCH}.zip" -o tflint.zip; \
    unzip -q tflint.zip; \
    install -m 0755 tflint /usr/local/bin/tflint; \
    rm -f tflint tflint.zip; \
    \
    # tfsec
    curl -fsSL "https://github.com/aquasecurity/tfsec/releases/download/v${TFSEC_VERSION}/tfsec-linux-${ARCH}" \
        -o /usr/local/bin/tfsec; \
    chmod 0755 /usr/local/bin/tfsec

# gcloud is the one tool with a directory-style install (libs + bundled
# python + bin). Stage it under /opt and symlink the entrypoints.
RUN set -eux; \
    case "${TARGETARCH}" in \
        amd64) GC_ARCH=x86_64 ;; \
        arm64) GC_ARCH=arm ;; \
    esac; \
    curl -fsSL "https://dl.google.com/dl/cloudsdk/channels/rapid/downloads/google-cloud-cli-${GCLOUD_VERSION}-linux-${GC_ARCH}.tar.gz" \
        | tar -xz -C /opt; \
    /opt/google-cloud-sdk/install.sh \
        --quiet \
        --usage-reporting false \
        --path-update false \
        --command-completion false; \
    ln -sf /opt/google-cloud-sdk/bin/gcloud /usr/local/bin/gcloud; \
    ln -sf /opt/google-cloud-sdk/bin/gsutil /usr/local/bin/gsutil; \
    ln -sf /opt/google-cloud-sdk/bin/bq /usr/local/bin/bq

USER coily

# ---------------------------------------------------------------------------
# full: cloud + every pkg manager + every runtime coily wraps.
# ---------------------------------------------------------------------------
FROM cloud AS full
USER root
ARG TARGETARCH

# Language runtimes: go, ruby. python3 / node already in :base.
ARG GO_VERSION=1.25.0
RUN set -eux; \
    case "${TARGETARCH}" in \
        amd64) GO_ARCH=amd64 ;; \
        arm64) GO_ARCH=arm64 ;; \
    esac; \
    curl -fsSL "https://go.dev/dl/go${GO_VERSION}.linux-${GO_ARCH}.tar.gz" \
        | tar -C /usr/local -xz; \
    ln -s /usr/local/go/bin/go /usr/local/bin/go; \
    ln -s /usr/local/go/bin/gofmt /usr/local/bin/gofmt

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        ruby-full \
        build-essential \
    && rm -rf /var/lib/apt/lists/*

# Node-side pkg managers via npm (already on PATH).
RUN npm install -g pnpm yarn bun

# Rust toolchain via rustup. Installed system-wide so cargo is on PATH for
# all users (including non-root coily).
ENV RUSTUP_HOME=/usr/local/rustup \
    CARGO_HOME=/usr/local/cargo \
    PATH=/usr/local/cargo/bin:${PATH}
RUN curl --proto '=https' --tlsv1.2 -fsSL https://sh.rustup.rs \
        | sh -s -- -y --no-modify-path --profile minimal --default-toolchain stable \
    && chown -R coily:coily /usr/local/cargo /usr/local/rustup

# just + task: small Go/Rust task runners that show up in repo configs.
ARG JUST_VERSION=1.36.0
ARG TASK_VERSION=3.40.1
RUN set -eux; \
    case "${TARGETARCH}" in \
        amd64) JUST_ARCH=x86_64-unknown-linux-musl; TASK_ARCH=linux_amd64 ;; \
        arm64) JUST_ARCH=aarch64-unknown-linux-musl; TASK_ARCH=linux_arm64 ;; \
    esac; \
    curl -fsSL "https://github.com/casey/just/releases/download/${JUST_VERSION}/just-${JUST_VERSION}-${JUST_ARCH}.tar.gz" \
        | tar -xz -C /usr/local/bin just; \
    curl -fsSL "https://github.com/go-task/task/releases/download/v${TASK_VERSION}/task_${TASK_ARCH}.tar.gz" \
        | tar -xz -C /usr/local/bin task

USER coily

# uv, poetry via pipx into the user environment. pipx puts them on
# /home/coily/.local/bin which is already on PATH from :base.
RUN pipx install uv \
    && pipx install poetry
