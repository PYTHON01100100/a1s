# a1s

[![Go](https://img.shields.io/badge/go-1.24%2B-00ADD8?logo=go)](go.mod)
[![License: MIT](https://img.shields.io/badge/license-MIT-orange.svg)](LICENSE)
[![PRs welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](CONTRIBUTING.md)

**A fast, keyboard-driven, full-screen terminal UI for managing Alibaba Cloud ECS instances.**

`a1s` lets you browse, filter, and operate on your Alibaba Cloud ECS fleet
without leaving the terminal or clicking through the web console. It talks to
your account exclusively through the official `aliyun` CLI — `a1s` never
stores, transmits, or even sees your AccessKey credentials.

`a1s` follows the k9s/e1s/ec2s school of terminal UI: a live, full-screen
instances table with an info panel and status bar, single-key actions,
`/` to filter, and a `:` command palette for everything else — themed in
Alibaba Cloud's own orange instead of those tools' teal/fuchsia.

> 🚧 Screenshots and a demo recording are coming soon.

## ✨ Features

- 🖥️ **Full-screen, live ECS table** — name, instance ID, status, type,
  compute kind (CPU/GPU), OS, internal/external IPs, VPC, vSwitch, zone, and
  billing, all in one auto-refreshing view (also `Ctrl-R` to refresh now).
- ℹ️ **Info panel** — every field for the selected instance (IDs, CIDRs, ENI,
  billing/expiry) plus the active keybindings, updated as you move the
  selection.
- 🔍 **Filter** instances live as you type (`/`), by plain text or a specific
  field (`status=running`, `zone=me-central-1a`).
- ⚡ **Full instance lifecycle**, one key each: `s` start, `S` stop, `x`
  economic stop (pauses vCPU/memory billing while stopped — the console's
  economical mode), `R` reboot, `D` terminate. Anything destructive or
  disruptive asks you to confirm first; starting never does.
- 💻 **Remote command execution** through Alibaba Cloud Cloud Assistant, no
  SSH required — `E` runs any shell command on the selected instance, with
  its output in a scrollable pane.
- 🧰 **`:` command palette** (k9s-style) for everything else: `bill`,
  `doctor`, `report`, `query`, `metrics`, `run`, the `uptime`/`disk`/`memory`/
  `failed`/`ports`/`quick` diagnostics, `currency`, `theme`, `profile`,
  `region`, `regions`, `profiles`, and `configure`.
- 🌍 **Multi-profile / multi-region** — auto-detects your current
  `aliyun-cli` profile and region, and lets you switch with `Ctrl-P` (or the
  `:profile <name>` / `:region <id>` palette commands) without restarting.
- 🔐 **Guided setup, credential-free by design** — `:configure [name]` hands
  the terminal to the official, interactive `aliyun configure` for the
  AccessKey ID/Secret — pasting works exactly like any other terminal
  prompt, and `a1s` itself never reads, stores, or logs your keys.
- 💰 **FinOps basics** — `:bill`, `:doctor` (health summary), and `:report`
  (writes a Markdown operations report). Display currency (`:currency
  USD|SAR`) is switchable live, independent of your account's settlement
  currency.
- 🎨 **Professional, Alibaba Cloud–branded UI** — the default theme uses
  Alibaba Cloud's orange, with a low-color `mono` theme available
  (`:theme alibaba` / `:theme mono`). Status colors (running/stopped/warning)
  never change between themes, so meaning always stays consistent.
- 🛑 **Safe by default** — `--read-only` disables every mutating action
  (start/stop/reboot/terminate/run), and destructive actions always confirm.

## 📦 Installation

Requires Go 1.24+.

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
/                filter the table (plain text, or field=value e.g. status=running)
g / G            jump to top / bottom
s                start the selected instance
S                stop normally
x                economic stop (billing paused while stopped)
R                reboot
D                terminate (asks for confirmation, irreversible)
E                run a shell command on the selected instance (no SSH)
Ctrl-P           switch aliyun-cli profile/region
Ctrl-R           refresh now (also auto-refreshes on its own)
?                help
q / Ctrl-C       quit

: command palette, e.g.:
  :bill 2026-08
  :doctor
  :report ops-report.md
  :query vpc=vpc-xxxx
  :metrics 30
  :run uptime
  :quick                    # uptime + disk + memory + failed + ports
  :currency SAR
  :theme mono
  :profile uat
  :region me-central-1
  :regions
  :profiles
  :configure uat            # add/update a profile: name it, pick a region, then keys
```

## Getting an account configured

`a1s` needs the [`aliyun` CLI](https://github.com/aliyun/aliyun-cli) to talk
to your account. If it isn't installed or configured yet, `a1s` starts
anyway and tells you so right in the UI, with a way to fix it without
leaving the app:

```text
:configure              # name a profile, pick a region from a list, then add your keys
Ctrl-R                  # reload once your account is ready
```

`:configure` also works for multiple accounts — run `:configure uat`,
`:configure client1`, `:configure client2`, etc. to add or update named
profiles, then switch between them any time with `Ctrl-P` or
`:profile <name>`.

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
