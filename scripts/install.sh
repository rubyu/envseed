#!/usr/bin/env bash
set -euo pipefail

REPO="rubyu/envseed"
BIN_NAME="envseed"

usage() {
  cat <<EOF
Install ${BIN_NAME} from GitHub Releases

Usage:
  install.sh [-v <version>] [-b <bin-dir>]

Options:
  -v, --version   Version tag to install (e.g., v0.1.0). If omitted, installs the latest release.
  -b, --bin-dir   Installation directory (default: ~/.local/bin if writable; otherwise /usr/local/bin if writable).

Environment:
  VERSION         Same as --version
  BIN_DIR         Same as --bin-dir

This script expects release assets named like:
  ${BIN_NAME}_<version>_<os>_<arch>.tar.gz
where <os> is linux|darwin and <arch> is amd64|arm64.
EOF
}

VERSION="${VERSION:-}"
BIN_DIR="${BIN_DIR:-}"

while [[ $# -gt 0 ]]; do
  case "$1" in
    -h|--help)
      usage; exit 0 ;;
    -v|--version)
      VERSION="$2"; shift 2 ;;
    -b|--bin-dir)
      BIN_DIR="$2"; shift 2 ;;
    *)
      echo "Unknown option: $1" >&2
      usage
      exit 1 ;;
  esac
done

detect_os() {
  case "$(uname -s)" in
    Linux)  echo linux ;;
    Darwin) echo darwin ;;
    *) echo "unsupported OS: $(uname -s)" >&2; exit 1 ;;
  esac
}

detect_arch() {
  case "$(uname -m)" in
    x86_64|amd64) echo amd64 ;;
    arm64|aarch64) echo arm64 ;;
    *) echo "unsupported ARCH: $(uname -m)" >&2; exit 1 ;;
  esac
}

latest_version() {
  # Avoid jq; use sed/grep
  curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
    | sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]\+\)".*/\1/p' | head -n1
}

choose_bin_dir() {
  if [[ -n "${BIN_DIR}" ]]; then
    echo "${BIN_DIR}"
    return
  fi
  local default1="$HOME/.local/bin"
  local default2="/usr/local/bin"
  if mkdir -p "$default1" 2>/dev/null && [[ -w "$default1" ]]; then
    echo "$default1"
  elif [[ -w "$default2" ]]; then
    echo "$default2"
  else
    echo "Error: no writable bin dir. Use --bin-dir" >&2
    exit 1
  fi
}

OS=$(detect_os)
ARCH=$(detect_arch)

if [[ -z "${VERSION}" ]]; then
  VERSION=$(latest_version)
  if [[ -z "${VERSION}" ]]; then
    echo "Failed to detect latest version" >&2
    exit 1
  fi
fi

DEST_DIR=$(choose_bin_dir)
TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

ASSET="${BIN_NAME}_${VERSION}_${OS}_${ARCH}.tar.gz"
URL="https://github.com/${REPO}/releases/download/${VERSION}/${ASSET}"

echo "Installing ${BIN_NAME} ${VERSION} for ${OS}/${ARCH} -> ${DEST_DIR}"
echo "Downloading ${URL}"
curl -fL "$URL" -o "${TMPDIR}/${ASSET}"

tar -xzf "${TMPDIR}/${ASSET}" -C "$TMPDIR"

if [[ ! -f "${TMPDIR}/${BIN_NAME}" ]]; then
  # Try to find the binary if archive contains a directory
  CAND=$(find "$TMPDIR" -type f -name "${BIN_NAME}" | head -n1 || true)
  if [[ -z "$CAND" ]]; then
    echo "Binary ${BIN_NAME} not found in archive" >&2
    exit 1
  fi
  mv "$CAND" "${TMPDIR}/${BIN_NAME}"
fi

install -m 0755 "${TMPDIR}/${BIN_NAME}" "${DEST_DIR}/${BIN_NAME}"
echo "Installed to ${DEST_DIR}/${BIN_NAME}"
echo "Version: $(${DEST_DIR}/${BIN_NAME} --version 2>/dev/null || echo 'installed')"

