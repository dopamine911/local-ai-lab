#!/usr/bin/env bash
# Build the lab CLI in a pinned Go container (nothing to install on the machine), test it, and optionally install it.
#   cli/build.sh            vet + test + build -> cli/bin/lab
#   cli/build.sh install    the same, then copy to ~/.local/bin/lab
#   cli/build.sh deps       (after changing a version below) update go.mod/go.sum, then build
# Every build also regenerates the command reference in docs/cli/ from the commands' help (lab docs) and says so
# when it changed: commit it with the code.
# Good result: "tests ok", then `lab version` prints a commit id.
# Pinned: Go 1.27.3 image by digest; libraries exactly in go.mod/go.sum (latest at 2026-11-02, setup/VERSIONS.md).
set -euo pipefail
CLI=$(cd "$(dirname "$0")" && pwd)
IMG=golang:1.27.3@sha256:2920bb124f1a2e12a1d39f0cab756ae0068441ee1da7655dec7f5e374944095b
VER=$(git -C "$CLI" describe --always --dirty 2>/dev/null || echo dev)
CACHE=$HOME/.cache/lab-go; mkdir -p "$CACHE"
go() { docker run --rm -u "$(id -u):$(id -g)" -e HOME=/tmp -e GOCACHE=/cache/build -e GOMODCACHE=/cache/mod \
         -e GOTOOLCHAIN=local -e CGO_ENABLED=0 -e GOFLAGS=-trimpath -v "$CLI":/src -v "$CACHE":/cache -w /src "$IMG" go "$@"; }
if [ "${1:-}" = deps ]; then
  go get github.com/spf13/cobra@v1.10.2 charm.land/bubbletea/v2@v2.1.1 charm.land/lipgloss/v2@v2.0.6 go.yaml.in/yaml/v3@v3.0.6
  go mod tidy
fi
go vet ./...
go test ./... && echo "tests ok"
go build -ldflags "-s -w -X main.version=$VER" -o bin/lab .
echo "built cli/bin/lab ($VER)"
DOCS="$CLI/../docs/cli"
"$CLI/bin/lab" docs "$DOCS"
if [ -n "$(git -C "$CLI/.." status --porcelain -- docs/cli 2>/dev/null)" ]; then
  echo "docs/cli changed (the help text changed): commit it with the code"
else
  echo "docs/cli up to date"
fi
if [ "${1:-}" = install ]; then
  install -D -m 755 "$CLI/bin/lab" "$HOME/.local/bin/lab"
  echo "installed ~/.local/bin/lab: $("$HOME/.local/bin/lab" version | head -1)"
fi
