#!/usr/bin/env bash
set -euo pipefail

status=0
while IFS=: read -r file line rest; do
  if [[ $file == vendor/* || $file == dist/* ]]; then
    continue
  fi
  prev=$((line-1))
  found=0
  while (( prev > 0 )); do
    l=$(sed -n "${prev}p" "$file")
    if [[ $l =~ ^[[:space:]]*$ ]]; then
      prev=$((prev-1))
      continue
    fi
    if [[ $l =~ \[EVT- ]]; then
      found=1
    fi
    break
  done
  if [[ $found -eq 0 ]]; then
    echo "$file:$line missing EVT tag (add // [EVT-...])"
    echo "If no EVT exists yet, update docs/spec.md Appendix C before landing this test." >&2
    status=1
  fi
done < <(rg -n '^[[:space:]]*func[[:space:]]+(Test|Fuzz)[^(]+' --glob '*_test.go' --glob '!*_integration_test.go' --glob '!*vendor/*' --color=never)

exit $status
