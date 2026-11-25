#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
go run ./bindings/testdata/generate > bindings/testdata/golden.json
