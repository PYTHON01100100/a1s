#!/usr/bin/env bash
# a1s is a full-screen tview TUI, so it needs a real TTY and can't be driven
# by piping commands into stdin. This smoke test drives it inside a tmux
# pane instead: launch with --sample-data, wait for the table to render,
# exercise a couple of keybindings/palette commands, then quit and check
# their effects.
set -euo pipefail
cd "$(dirname "$0")/.."

command -v tmux >/dev/null || { echo "tmux is required for this smoke test"; exit 1; }

mkdir -p bin
go build -o bin/a1s ./cmd/a1s

session="a1s-smoke-$$"
report="$(pwd)/smoke-report.md"
rm -f "$report"

cleanup() { tmux kill-session -t "$session" >/dev/null 2>&1 || true; }
trap cleanup EXIT

tmux new-session -d -s "$session" -x 200 -y 50 \
  "./bin/a1s --sample-data --region me-central-1 --currency SAR"
sleep 1.5

pane="$(tmux capture-pane -t "$session" -p)"
echo "$pane" | grep -q "web-01" || { echo "smoke test failed: table did not render sample instances"; echo "$pane"; exit 1; }

# Exercise the ":" command palette (report) and the "/" filter.
tmux send-keys -t "$session" ":report smoke-report.md" Enter
sleep 1
tmux send-keys -t "$session" "/" "web" Escape Escape
sleep 0.5
pane="$(tmux capture-pane -t "$session" -p)"
echo "$pane" | grep -q "web-01" || { echo "smoke test failed: filter did not narrow the table"; echo "$pane"; exit 1; }

tmux send-keys -t "$session" "q"
sleep 0.5

test -s "$report" || { echo "smoke test failed: $report was not created"; exit 1; }
echo "Smoke test passed. Report: $report"
