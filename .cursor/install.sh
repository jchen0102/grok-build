#!/usr/bin/env bash
# Idempotent Cloud Agent bootstrap for the Grok Build Rust workspace.
set -euo pipefail

DOTSLASH_VERSION="v0.5.9"

# The Rust toolchain is pinned by rust-toolchain.toml and installed by the base
# image / rustup on first cargo invocation. Only the protoc launcher and the
# warmed build cache are prepared here.

# dotslash resolves the repo's bin/protoc launcher (protoc 29.3) used by proto
# codegen. Install it to a PATH location if it is not already present.
if ! command -v dotslash >/dev/null 2>&1; then
  arch="$(uname -m)"
  url="https://github.com/facebook/dotslash/releases/download/${DOTSLASH_VERSION}/dotslash-ubuntu-22.04.${arch}.tar.gz"
  curl -LSfs "$url" | sudo tar fxz - -C /usr/local/bin
fi

# Fetch protoc via the dotslash launcher so codegen does not download at build
# time, and fail early with a clear message if resolution breaks.
./bin/protoc --version

# Warm the cargo cache and build the primary binary so the workspace is
# immediately usable. target/ is gitignored.
cargo build -p xai-grok-pager-bin
