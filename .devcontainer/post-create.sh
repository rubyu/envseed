#!/usr/bin/env bash
set -euo pipefail

sudo /usr/local/bin/setup-firewall.sh

go env -w GOPATH=/home/developer/go GOMODCACHE=/home/developer/go/pkg/mod
