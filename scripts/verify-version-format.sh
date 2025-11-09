#!/usr/bin/env bash
set -euo pipefail

BIN="dist/envseed"
if [[ ! -x "$BIN" ]]; then
  echo "binary not found: $BIN" >&2
  exit 1
fi

host_os=$(go env GOOS 2>/dev/null || uname | tr '[:upper:]' '[:lower:]')
host_arch=$(go env GOARCH 2>/dev/null || uname -m)
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
