#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${ROOT_DIR}"

if [[ "${1:-}" == "vote" ]]; then
  TOPIC="${2:-}"
  if [[ -z "${TOPIC}" ]]; then
    echo "usage: ./scripts/debug_grpc.sh vote <topic>" >&2
    exit 1
  fi
  go run ./cmd/grpcclient -action vote -topic "${TOPIC}"
  exit 0
fi

if [[ "${1:-}" == "results" || -z "${1:-}" ]]; then
  go run ./cmd/grpcclient -action results
  exit 0
fi

echo "usage:" >&2
echo "  ./scripts/debug_grpc.sh results" >&2
echo "  ./scripts/debug_grpc.sh vote <topic>" >&2
exit 1
