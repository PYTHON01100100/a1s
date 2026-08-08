#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
mkdir -p bin
go build -o bin/a1s ./cmd/a1s
printf '%s\n' \
  'ecs' \
  'metrics i-demo-web01 60' \
  'price ecs.g8i.large Hour' \
  'bill 2026-08' \
  'run i-demo-web01 systemctl is-active nginx' \
  'doctor i-demo-web01' \
  'report smoke-report.md' \
  'quit' | ./bin/a1s --demo --region me-central-1 --currency SAR

test -s smoke-report.md
echo "Smoke test passed. Report: smoke-report.md"
