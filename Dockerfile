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
# ---------------------------------------------------------------------------
FROM base AS cloud
USER root
ARG TARGETARCH

# aws v2 (official installer; arch-aware).
RUN set -eux; \
    case "${TARGETARCH}" in \
        amd64) AWS_ARCH=x86_64 ;; \
        arm64) AWS_ARCH=aarch64 ;; \
        *) echo "unsupported arch ${TARGETARCH}" && exit 1 ;; \
    esac; \
    curl -fsSL "https://awscli.amazonaws.com/awscli-exe-linux-${AWS_ARCH}.zip" -o /tmp/awscliv2.zip; \
    unzip -q /tmp/awscliv2.zip -d /tmp; \
    /tmp/aws/install; \
    rm -rf /tmp/aws /tmp/awscliv2.zip

# Apt-repo-distributed CLIs: gh, kubectl, helm, terraform, tailscale, gcloud,
# docker-ce-cli. Each gets its own keyring + sources.list line. Grouped into
# one RUN to keep the layer count down.
RUN set -eux; \
    install -m 0755 -d /etc/apt/keyrings; \
    \
    # GitHub CLI
    curl -fsSL https://cli.github.com/packages/githubcli-archive-keyring.gpg \
        | tee /etc/apt/keyrings/githubcli-archive-keyring.gpg >/dev/null; \
    chmod go+r /etc/apt/keyrings/githubcli-archive-keyring.gpg; \
    echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main" \
        > /etc/apt/sources.list.d/github-cli.list; \
    \
    # Kubernetes (kubectl)
    curl -fsSL https://pkgs.k8s.io/core:/stable:/v1.31/deb/Release.key \
        | gpg --dearmor -o /etc/apt/keyrings/kubernetes-apt-keyring.gpg; \
    echo "deb [signed-by=/etc/apt/keyrings/kubernetes-apt-keyring.gpg] https://pkgs.k8s.io/core:/stable:/v1.31/deb/ /" \
        > /etc/apt/sources.list.d/kubernetes.list; \
    \
    # Helm
    curl -fsSL https://baltocdn.com/helm/signing.asc \
        | gpg --dearmor -o /etc/apt/keyrings/helm.gpg; \
    echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/helm.gpg] https://baltocdn.com/helm/stable/debian/ all main" \
        > /etc/apt/sources.list.d/helm.list; \
    \
    # HashiCorp (terraform)
    curl -fsSL https://apt.releases.hashicorp.com/gpg \
        | gpg --dearmor -o /etc/apt/keyrings/hashicorp-archive-keyring.gpg; \
    echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/hashicorp-archive-keyring.gpg] https://apt.releases.hashicorp.com noble main" \
        > /etc/apt/sources.list.d/hashicorp.list; \
    \
    # Tailscale
    curl -fsSL https://pkgs.tailscale.com/stable/ubuntu/noble.noarmor.gpg \
        -o /etc/apt/keyrings/tailscale-archive-keyring.gpg; \
    curl -fsSL https://pkgs.tailscale.com/stable/ubuntu/noble.tailscale-keyring.list \
        -o /etc/apt/sources.list.d/tailscale.list; \
    \
    # Google Cloud SDK
    curl -fsSL https://packages.cloud.google.com/apt/doc/apt-key.gpg \
        | gpg --dearmor -o /etc/apt/keyrings/cloud.google.gpg; \
    echo "deb [signed-by=/etc/apt/keyrings/cloud.google.gpg] https://packages.cloud.google.com/apt cloud-sdk main" \
        > /etc/apt/sources.list.d/google-cloud-sdk.list; \
    \
    # Docker (CLI only - the daemon is the host's concern)
    curl -fsSL https://download.docker.com/linux/ubuntu/gpg \
        | gpg --dearmor -o /etc/apt/keyrings/docker.gpg; \
    echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu noble stable" \
        > /etc/apt/sources.list.d/docker.list; \
    \
    apt-get update; \
    apt-get install -y --no-install-recommends \
        gh \
        kubectl \
        helm \
        terraform \
        tailscale \
        google-cloud-cli \
        docker-ce-cli; \
    rm -rf /var/lib/apt/lists/*

# tflint + tfsec ship as single-binary GitHub releases. Pinned versions so
# image rebuilds are reproducible.
ARG TFLINT_VERSION=0.55.1
ARG TFSEC_VERSION=1.28.13
RUN set -eux; \
    case "${TARGETARCH}" in \
        amd64) TFLINT_ARCH=linux_amd64; TFSEC_ARCH=linux-amd64 ;; \
        arm64) TFLINT_ARCH=linux_arm64; TFSEC_ARCH=linux-arm64 ;; \
    esac; \
    curl -fsSL "https://github.com/terraform-linters/tflint/releases/download/v${TFLINT_VERSION}/tflint_${TFLINT_ARCH}.zip" -o /tmp/tflint.zip; \
    unzip -q /tmp/tflint.zip -d /usr/local/bin; \
    rm /tmp/tflint.zip; \
    curl -fsSL "https://github.com/aquasecurity/tfsec/releases/download/v${TFSEC_VERSION}/tfsec-${TFSEC_ARCH}" -o /usr/local/bin/tfsec; \
    chmod 0755 /usr/local/bin/tfsec

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
