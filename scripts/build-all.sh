#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DIST_DIR="${ROOT_DIR}/dist"

if ! command -v go >/dev/null 2>&1; then
  echo "[ERROR] go is not installed or not in PATH" >&2
  exit 1
fi

TAG="${1:-local}"
mkdir -p "${DIST_DIR}"

build_one() {
  local goos="$1"
  local goarch="$2"
  local pkg_target="$3"
  local arch_label="$4"
  local bin_ext="$5"

  local output="${DIST_DIR}/rajidou-${TAG}-${pkg_target}-${arch_label}${bin_ext}"
  echo "[INFO] building ${goos}/${goarch} -> ${output}"

  GOOS="${goos}" GOARCH="${goarch}" CGO_ENABLED=0 \
    go build \
      -trimpath \
      -buildvcs=false \
      -ldflags "-s -w -buildid=" \
      -o "${output}" \
      "${ROOT_DIR}/cmd/rajidou"
}

build_one linux amd64 linux x64 ""
build_one linux arm64 linux arm64 ""
build_one darwin amd64 macos x64 ""
build_one darwin arm64 macos arm64 ""
build_one windows amd64 win x64 ".exe"
build_one windows arm64 win arm64 ".exe"

echo "[OK] all binaries are in ${DIST_DIR}"
ls -lh "${DIST_DIR}"
