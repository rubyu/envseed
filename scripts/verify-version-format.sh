#!/usr/bin/env bash
set -euo pipefail

BIN="dist/envseed"
if [[ ! -x "$BIN" ]]; then
  echo "binary not found: $BIN" >&2
  exit 1
fi

ver=$("$BIN" --version | tr -d '\r')
if echo "$ver" | grep -Eq '^v[0-9]+\.[0-9]+\.[0-9]+(-dev)?\+[0-9]{8}\.[0-9a-f]{12}(\.dirty)?$'; then
  exit 0
fi

echo "invalid version format: $ver" >&2
exit 1

