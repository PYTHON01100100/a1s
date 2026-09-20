# a1s

[![Go](https://img.shields.io/badge/go-1.23%2B-00ADD8?logo=go)](go.mod)
[![License: MIT](https://img.shields.io/badge/license-MIT-orange.svg)](LICENSE)
[![PRs welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](CONTRIBUTING.md)

**A fast, keyboard-driven terminal UI for managing Alibaba Cloud ECS instances.**

`a1s` lets you browse, filter, and operate on your Alibaba Cloud ECS fleet
without leaving the terminal or clicking through the web console. It talks to
your account exclusively through the official `aliyun` CLI — `a1s` never
stores, transmits, or even sees your AccessKey credentials.

> 🚧 Screenshots and a demo recording are coming soon.

## ✨ Features

- 🖥️ **Real-time ECS inventory** — name, instance ID, status, type, OS,
  internal/external IPs, billing mode, VPC, vSwitch, and zone, in one table.
- 🔍 **Filter and search** instances by name, status, type, zone, or a
  specific field (`filter status=running`, `filter zone=me-central-1a`).
- ⚡ **Full instance lifecycle** — `start`, `stop`, `reboot`, `terminate`.
  - `stop` supports both Alibaba Cloud stop modes: a **normal** stop (keeps
    the instance billed and ready for a fast restart) and an **economic**
    stop (`stop eco`), which pauses vCPU/memory billing while the instance
    stays stopped — the same behavior as the console's economical mode.
  - `terminate` always asks for confirmation before deleting anything.
- 🔄 **Live auto-refresh** (`watch [seconds]`) for a dashboard-style view
  that updates on an interval until you stop it.
- 🌍 **Multi-profile / multi-region** — auto-detects your current
  `aliyun-cli` profile and region, and lets you switch (`profiles`,
  `profile <name>`, `region <region-id>`) without restarting.
- 🔐 **Guided setup, credential-free by design** — `configure` walks you
  through naming a profile (`uat`, `client1`, `client2`, ...), picking a
  region from a reference list (or typing any region ID), then enters the
  official `aliyun configure` for the AccessKey ID/Secret — where pasting
  works exactly like any other terminal prompt. `a1s` itself never reads,
  stores, or logs your keys.
- 💻 **Remote command execution** through Alibaba Cloud Cloud Assistant
  (`run <ecs> <command>`), plus safe one-word diagnostics (`uptime`, `disk`,
  `memory`, `failed`, `ports`, `quick`).
- 💰 **FinOps basics** — `bill`, `doctor` (health summary), and `report`
  (writes a Markdown operations report). Display currency (`currency
  USD|SAR`) is switchable live, independent of your account's settlement
  currency.
- 🎨 **Professional, Alibaba Cloud–branded UI** — the default theme uses
  Alibaba Cloud's orange, with a low-color `mono` theme available
  (`theme alibaba` / `theme mono`). Status colors (running/stopped/warning)
  never change between themes, so meaning always stays consistent.
- 🛑 **Safe by default** — `--read-only` disables every mutating action
  (start/stop/reboot/terminate/run), and destructive actions always confirm.

## 📦 Installation

Requires Go 1.23+.

```bash
go install github.com/PYTHON01100100/a1s/cmd/a1s@latest
```

Or build from source:

```bash
git clone https://github.com/PYTHON01100100/a1s.git
cd a1s
go build -o bin/a1s ./cmd/a1s
```

## 🚀 Usage

```bash
a1s
```

That's it. `a1s` auto-detects your current `aliyun-cli` profile and region
and shows exactly what your account returns — no flags required.

If no Alibaba Cloud account is configured yet, `a1s` still starts and shows
you how to fix it, right inside the app (see [Getting an account
configured](#getting-an-account-configured) below).

<details>
<summary>Other flags</summary>

| Flag | Description |
|---|---|
| `--region <id>` | Override the Alibaba Cloud region, e.g. `me-central-1` |
| `--profile <name>` | Override the `aliyun-cli` profile to use |
| `--currency USD\|SAR` | Initial display currency (changeable live with `currency`) |
| `--read-only` | Disable every mutating action (start/stop/reboot/terminate/run) |
| `--demo` | UI only — no cloud calls, no data |
| `--sample-data` | Local, clearly-fake instances — no account or CLI required |
| `--version` | Print the version and exit |

</details>

### Inside the app

```text
ecs                       list instances
use 1                     select an instance by row, name, or ID
stop eco web-01           economic stop (billing paused while stopped)
stop web-02               normal stop
reboot cache-01
terminate db-01           asks for confirmation before deleting
filter status=running
watch 10                  live-refresh every 10s (Ctrl+C to stop)
run web-01 uptime
configure                 add a profile: name it, pick a region, then keys
regions                   show a reference list of region IDs
profiles / profile <name> / region <region-id>
currency SAR              change the display currency, any time
theme mono / theme alibaba
bill 2026-08
doctor
report ops-report.md
help                      full command reference
```

Every command also has its own `--help`, e.g. `stop --help`.

## Getting an account configured

`a1s` needs the [`aliyun` CLI](https://github.com/aliyun/aliyun-cli) to talk
to your account. If it isn't installed or configured yet, `a1s` starts
anyway and tells you so right in the UI, with a way to fix it without
leaving the app:

```text
configure              # name a profile, pick a region from a list, then add your keys
dashboard              # reload once your account is ready
```

`configure` also works for multiple accounts — run `configure uat`,
`configure client1`, `configure client2`, etc. to add or update named
profiles, then switch between them any time with `profile <name>`.

Prefer not to use real credentials yet? Run `a1s --sample-data` to try
everything — lifecycle actions included — against local, clearly-fake
instances.

## Safety model

- `--read-only` disables every mutating action.
- `terminate` always requires interactive `y/N` confirmation.
- `a1s` shells out to the official `aliyun` CLI for every cloud call; it
  never stores, transmits, or logs your AccessKey ID/Secret.
- Command output is evidence — nothing is inferred or fabricated beyond what
  a command actually returned.

See [docs/DESIGN.md](docs/DESIGN.md) for the longer-term design notes,
including the roadmap.

## Roadmap

An AI copilot (natural-language questions over your infrastructure, a chat
console) is planned but intentionally **not** included yet. This release is
focused on making direct ECS management itself fast, predictable, and safe.

## Inspiration

`a1s` follows the terminal-UI-for-cloud-resources pattern set by:

- [**k9s**](https://github.com/derailed/k9s) — the terminal UI for Kubernetes
  that popularized this whole category of tool.
- [**g1c**](https://github.com/nlamirault/g1c) — a terminal UI for managing
  Google Cloud VM instances.
- [**e1s**](https://github.com/keidarcy/e1s) — a terminal UI for AWS ECS.

`a1s` brings the same idea to Alibaba Cloud ECS.

## Contributing

Contributions are welcome — see [CONTRIBUTING.md](CONTRIBUTING.md) for how
to set up your environment, run the test suite, and submit a pull request.

## License

[MIT](LICENSE)
