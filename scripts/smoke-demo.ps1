# a1s is a full-screen tview TUI, so it needs a real TTY and can't be
# driven by piping commands into stdin — there's no tmux-equivalent smoke
# test on Windows here yet. This script only verifies the binary builds and
# starts; drive the interactive UI itself manually, or use
# scripts/smoke-demo.sh on Linux/macOS (with tmux) for a scripted pass.
$ErrorActionPreference = "Stop"
Set-Location (Join-Path $PSScriptRoot "..")
New-Item -ItemType Directory -Force -Path bin | Out-Null
go build -o bin/a1s.exe ./cmd/a1s
& ./bin/a1s.exe --version
Write-Host "Build smoke check passed. Run 'bin/a1s.exe --sample-data' to try the UI interactively."
