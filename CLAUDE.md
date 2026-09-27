# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

`a1s` is a Go, full-screen terminal UI (k9s/e1s/ec2s-style, built on
[tview](https://github.com/rivo/tview)/[tcell](https://github.com/gdamore/tcell))
for managing Alibaba Cloud ECS instances, themed in Alibaba Cloud's orange.
It shells out to the official `aliyun` CLI for every cloud call and never
stores, transmits, or sees AccessKey credentials itself — that design
constraint shapes most of `internal/aliyun` and `internal/config`.

## Commands

```bash
go build ./...                        # build everything
go build -o bin/a1s ./cmd/a1s         # build the binary
go vet ./...
go test ./...                         # run all tests
go test ./internal/aliyun/...         # test a single package
go test ./internal/ui/ -run TestFilterInstancesFieldMatch  # run a single test
gofmt -l .                            # check formatting (should report nothing for touched files)

go run ./cmd/a1s --sample-data        # try it with fake local instances, no account/CLI needed
./scripts/smoke-demo.sh               # tmux-driven scripted smoke run against --sample-data (Linux/macOS)
./scripts/smoke-demo.ps1              # Windows: build + --version check only (no tmux equivalent yet)
```

There is no linter config beyond `go vet`/`gofmt`. CI expectations (per
CONTRIBUTING.md) are: `go build ./...`, `go vet ./...`, `go test ./...`, and
`gofmt -l .` all clean for touched files.

`a1s` is a full-screen TUI, so it needs a real TTY — it can't be exercised
by piping commands into stdin. To drive it non-interactively (for a manual
check or a new smoke test), run it inside a `tmux` pane and use
`tmux send-keys` / `tmux capture-pane -p`; see `scripts/smoke-demo.sh`.

## Architecture

```
cmd/a1s/            entry point: flag parsing, wires config -> aliyun.Client -> ui.App
internal/aliyun/    shells out to the `aliyun` CLI; owns all cloud calls
internal/config/    env var + aliyun-cli profile/region detection (~/.aliyun/config.json)
internal/currency/  USD/SAR display conversion (a display preference only)
internal/model/     shared plain-data types (ECSInstance, MetricPoint, BillSummary, ...)
internal/report/    Markdown report / doctor-summary generation
internal/ui/        the tview terminal UI: table/header/footer widgets, overlays, and actions
```

### Cloud access boundary (`internal/aliyun`)

- `Runner` is the seam for testability: `ExecRunner` shells out to the real
  `aliyun` binary; tests inject a `fakeRunner` that records calls and returns
  canned JSON (see `client_test.go`). Any new cloud call should go through
  `runner.Run(ctx, args...)`, not exec directly.
- `Client` has three data modes controlled by `config.Config`:
  - normal (live `aliyun` CLI calls),
  - `SampleData` (deterministic fake instances held in `Client.sample`,
    mutated in place by lifecycle calls so the demo stays self-consistent),
  - `Demo` (UI-only, returns empty results, no cloud calls at all).
  Every public method on `Client` branches on these before touching the
  network — keep that pattern when adding new methods.
- `cfg.ReadOnly` is enforced centrally via `requireWritable()`/checks inside
  each mutating method (Start/Stop/Reboot/Delete/RunCommand), not in the UI
  layer. `terminate`/`DeleteInstance` always requires interactive
  confirmation at the UI layer regardless of read-only state.
- Alibaba API responses are generic `map[string]any` JSON dug out with the
  `dig*`/`str`/`f64` helpers at the bottom of `client.go` rather than typed
  structs, because CLI output shapes vary by API version.

### Config resolution (`internal/config`)

`FromEnv()` builds the effective `Config` by layering: environment vars
(`ALIBABA_CLOUD_PROFILE`, `ALIBABA_CLOUD_REGION_ID`, `A1S_CURRENCY`,
`A1S_SAR_PER_USD`) over auto-detected `aliyun-cli` profile/region from
`~/.aliyun/config.json` (also checked at `$USERPROFILE` and WSL's
`/mnt/c/Users/*/.aliyun/config.json` for cross-platform convenience). CLI
flags in `cmd/a1s/main.go` override both. a1s only ever *reads* this file
for profile/region metadata — never credentials.

### UI layer (`internal/ui`)

The layout is a fixed `tview.Flex`: an info panel (`header.go`) on top, the
ECS table (`table.go`) filling the middle, a chip-style status bar
(`footer.go`) pinned to the bottom. `app.go` wires it together (`App.New`)
and owns the event loop (`App.Run`), which also kicks off the initial fetch
and a 20s auto-refresh ticker (`autoRefreshInterval`).

- **`App` owns the `*aliyun.Client` directly** (unlike ec2s's caller-supplied
  `Actions` struct) and calls its methods itself from goroutines — a1s talks
  to one profile/region at a time, so there's no multi-account aggregation
  to abstract away. Any cloud call must run off the UI goroutine and report
  back via `a.tapp.QueueUpdateDraw(...)`, never touch tview widgets directly.
- **Never call `a.tapp.QueueUpdateDraw` synchronously before `a.tapp.Run()`
  has started its event loop** (i.e. not directly inside `App.Run`, only
  from a goroutine it spawns) — that channel send blocks forever until
  `Run()`'s loop is draining it, which deadlocks the whole program before
  the screen ever draws. This is why `Run()` spawns the initial `refresh()`
  with `go`, not a direct call.
- **Keybindings** are dispatched in `app.go`'s `handleKey`, gated on
  `a.pages.GetFrontPage() == pageMain` so overlays get every keystroke
  otherwise. Lifecycle keys (`s`/`S`/`x`/`R`/`D`) live in `actions.go`,
  guarded by `requireCloudData()` and (for anything but start) a
  `showConfirm` modal; `stop` (`S`) maps to `aliyun.StopNormal`, `x` (eco
  stop) to `aliyun.StopEco`.
- **The `:` command palette** (`palette.go`) is the extension point for
  anything that isn't a single-key lifecycle action or the run-command
  overlay — `bill`, `doctor`, `report`, `query`, `metrics`, `run`, the quick
  diagnostics, `currency`, `theme`, `profile`, `region`, `regions`,
  `profiles`, `configure`. Add new long-tail commands as a new `case` in
  `runPalette` plus a `paletteX` handler; instant/local ones call
  `a.Notify`, anything producing a block of text opens the shared result
  pane via `a.showPaletteResult` (fill it from a goroutine with
  `a.tapp.QueueUpdateDraw`, guarding on `a.resultView != view` in case the
  user closed it first).
- **`:configure`** hands the real terminal back with `a.tapp.Suspend(...)`
  so the official, interactive `aliyun configure` can run with inherited
  stdio (paste works normally) — `a1s` never reads/stores/logs the keys.
- Colors are theme-swappable tag/hex-string vars (`orange`, `orange2`,
  `accent`) in `styles.go`, rebuilt by `setTheme()` for `alibaba`/`mono` and
  re-applied to tview via `applyTheme()`. Status colors (`colorGreen`/
  `colorYellow`/`colorRed`, via `stateColor()`) are intentionally NOT part
  of the swappable theme — they must stay constant across themes.
- `filterInstances` (in `filter.go`) backs both the `/` live filter box and
  the palette's `query` command; it supports plain substring search and
  `field=value` matching (status, zone, type, vpc, etc.) via `fieldValue`.
- `format.go` holds the ECS-domain formatting helpers shared by the table,
  info panel, and palette output (`billingLabel`, `billingExpiry`,
  `instanceKind`/`isGPUFamily`, `pairLabelCIDR`, `zoneLabel`, `bar`,
  `indentOutput`) — reuse these rather than re-deriving billing/GPU/zone
  logic in a new view.

## Notable constraints (see docs/DESIGN.md, CONTRIBUTING.md)

- No AI/chat features in this phase of the project by design — an AI
  copilot is a deferred roadmap item; keep unrelated AI features out of
  changes here unless explicitly requested.
- `ref-design-ui-ignore/` is a reference/inspiration snapshot from a
  different (AWS-focused) project — ec2s, whose tview UI a1s's current
  `internal/ui` is modeled on — not part of the `a1s` module; it's
  untracked and excluded from normal work in this repo.
