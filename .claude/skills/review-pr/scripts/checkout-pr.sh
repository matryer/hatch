#!/usr/bin/env bash
set -euo pipefail
if [[ $# -ne 1 ]]; then
  echo "usage: checkout-pr.sh <pr-number>" >&2
  exit 2
fi
gh pr checkout "$1"
