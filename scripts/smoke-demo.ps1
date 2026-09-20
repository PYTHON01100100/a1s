$ErrorActionPreference = "Stop"
Set-Location (Join-Path $PSScriptRoot "..")
New-Item -ItemType Directory -Force -Path bin | Out-Null
go build -o bin/a1s.exe ./cmd/a1s
@(
  "ecs",
  "use i-demo-web01",
  "metrics 60",
  "stop eco i-demo-web01",
  "start i-demo-web01",
  "bill 2026-08",
  "run i-demo-web01 systemctl is-active nginx",
  "doctor i-demo-web01",
  "report smoke-report.md",
  "quit"
) | & ./bin/a1s.exe --sample-data --region me-central-1 --currency SAR
if (-not (Test-Path smoke-report.md)) { throw "smoke-report.md was not created" }
Write-Host "Smoke test passed. Report: smoke-report.md"
