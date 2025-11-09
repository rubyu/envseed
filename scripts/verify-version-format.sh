#!/usr/bin/env bash
set -euo pipefail

BIN="dist/envseed"
if [[ ! -x "$BIN" ]]; then
  echo "binary not found: $BIN" >&2
  exit 1
fi

uname_s=$(uname -s)
case "$uname_s" in
  Linux)  host_os=linux ;;
  Darwin) host_os=darwin ;;
  *)      host_os=$(printf '%s' "$uname_s" | tr '[:upper:]' '[:lower:]') ;;
esac
uname_m=$(uname -m)
case "$uname_m" in
  x86_64|amd64) host_arch=amd64 ;;
  aarch64|arm64) host_arch=arm64 ;;
  i386|i686) host_arch=386 ;;
  *) host_arch=$uname_m ;;
esac
build_os="${GOOS:-$host_os}"
build_arch="${GOARCH:-$host_arch}"

regex='^v[0-9]+\.[0-9]+\.[0-9]+(-dev)?\+[0-9]{8}\.[0-9a-f]{12}(\.dirty)?$'

if [[ "$host_os/$host_arch" == "$build_os/$build_arch" ]]; then
  # Executable on this host: run and also verify embedded string
  ver=$("$BIN" --version | tr -d '\r')
  if ! echo "$ver" | grep -Eq "$regex"; then
    echo "invalid version format: $ver" >&2
    exit 1
  fi
  if ! strings "$BIN" | grep -Fqx "$ver"; then
    echo "embedded version string not found: $ver" >&2
    exit 1
  fi
else
  # Cross binary: static verification only
  if [[ -n "${VERSION_STR:-}" ]]; then
    if ! strings "$BIN" | grep -Fqx "$VERSION_STR"; then
      echo "embedded version string not found (expected VERSION_STR): $VERSION_STR" >&2
      exit 1
    fi
  else
    if ! strings "$BIN" | grep -Eaq "$regex"; then
      echo "no valid version string embedded in binary" >&2
      exit 1
    fi
  fi
fi
