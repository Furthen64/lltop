#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")"

mkdir -p bin
go build -o bin/lltop ./cmd/lltop
echo "Done. See bin/ or run ./launch.sh"
