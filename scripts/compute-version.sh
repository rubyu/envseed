#!/usr/bin/env bash
set -euo pipefail

TAG="$(git describe --tags --exact-match 2>/dev/null || true)"
DATE="$(date -u +%Y%m%d)"
SHA="$(git rev-parse --short=12 HEAD | tr 'A-Z' 'a-z')"
DIRTY=""
if ! git diff --quiet --ignore-submodules HEAD; then DIRTY=".dirty"; fi

BRANCH="${GITHUB_REF_NAME:-}"
if [[ -z "${BRANCH}" || "${BRANCH}" == "HEAD" ]]; then
  BRANCH="$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo HEAD)"
fi

if [[ -n "${TAG}" ]]; then
  # Stable release: tag MUST be on main; no .dirty
  if ! git branch -r --contains HEAD | grep -q "origin/main"; then
    echo "tag ${TAG} is not on main" >&2; exit 1
  fi
  if [[ -n "${DIRTY}" ]]; then
    echo "working tree dirty on release tag" >&2; exit 1
  fi
  echo "${TAG}+${DATE}.${SHA}"
  exit 0
fi

LAST="$(git tag --list 'v[0-9]*' --sort=-version:refname | head -n1 || true)"
if [[ -z "${LAST}" ]]; then LAST="v0.0.0"; fi
M="$(echo "${LAST}" | sed -E 's/^v([0-9]+)\..*$/\1/')"
m="$(echo "${LAST}" | sed -E 's/^v[0-9]+\.([0-9]+)\..*$/\1/')"
p="$(echo "${LAST}" | sed -E 's/^v[0-9]+\.[0-9]+\.([0-9]+).*$/\1/')"

if [[ "${BRANCH}" == "main" ]] || git branch -r --contains HEAD | grep -q "origin/main"; then
  p=$((p+1))
  echo "v${M}.${m}.${p}-dev+${DATE}.${SHA}${DIRTY}"
else
  m=$((m+1))
  echo "v${M}.${m}.0-dev+${DATE}.${SHA}${DIRTY}"
fi
