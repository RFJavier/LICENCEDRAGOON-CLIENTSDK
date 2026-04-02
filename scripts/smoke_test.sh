#!/usr/bin/env bash
set -euo pipefail

echo "Running SDK smoke test"
pushd "$(dirname "$0")/../examples" >/dev/null
go mod tidy
go build .
echo "Smoke build OK"
popd >/dev/null
