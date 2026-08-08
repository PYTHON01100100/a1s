# a1s — AI-powered Alibaba Cloud terminal operations

`a1s` is an experimental terminal UI for Alibaba Cloud inspired by **g1c**, **e1s**, and the official **aliyun-cli**.

The MVP focuses on one workflow:

> Browse → Observe → Run → Verify → Analyze → Report

It reuses the credentials/profile behavior of `aliyun-cli` rather than storing Alibaba Cloud secrets itself.

## MVP features

- ECS inventory table
- Region/profile selection at startup
- Read-only mode
- Cloud Assistant `RunCommand` execution with output polling
- Quick ECS health commands (`uptime`, disk, memory, failed services, listening ports)
- CloudMonitor CPU metrics with terminal bars
- `doctor` health summary
- AI copilot over an OpenAI-compatible `/chat/completions` endpoint
- Markdown operations reports
- Current-month billing summary
- ECS `DescribePrice` quote lookup
- User-selectable display currency: USD or SAR
- Configurable USD→SAR display FX (default: `3.75`)
- Full local `--demo` mode that requires no Alibaba Cloud account

## Important pricing note

`a1s` does **not** change the settlement currency on your Alibaba Cloud account. It reads the currency returned by Alibaba Cloud APIs and can convert it for presentation in the terminal/report.

For international Alibaba Cloud accounts, ECS `DescribePrice` commonly returns USD. `a1s` can display the quote in SAR using `A1S_SAR_PER_USD`.

## Architecture

```text
                      ┌────────────────────┐
                      │       a1s TUI      │
                      └─────────┬──────────┘
                                │
            ┌───────────────────┼────────────────────┐
            │                   │                    │
            ▼                   ▼                    ▼
      aliyun-cli          AI Provider          Report Engine
            │          (OpenAI compatible)           │
            │                   │                    ▼
            │                   │                Markdown
            ▼                   │
   Alibaba Cloud OpenAPI        │
            │                   │
   ┌────────┼────────┬──────────┼──────────┐
   ▼        ▼        ▼          ▼          ▼
  ECS   CloudMonitor Billing   Pricing   Cloud Assistant
```

## Requirements

- Go 1.23+
- `aliyun` CLI for live mode
- Alibaba Cloud credentials/profile configured in `aliyun-cli`
- Cloud Assistant Agent on ECS for remote commands
- Optional OpenAI-compatible LLM endpoint for AI features

## Build

```bash
git clone <your-repository>
cd a1s
go test ./...
go build -o bin/a1s ./cmd/a1s
```

Windows PowerShell:

```powershell
go test ./...
go build -o bin/a1s.exe ./cmd/a1s
```

## Test without Alibaba Cloud

This is the fastest test:

```bash
go run ./cmd/a1s --demo --region me-central-1 --currency SAR
```

Then try:

```text
ecs
metrics i-demo-web01 60
price ecs.g8i.large Hour
bill 2026-08
run i-demo-web01 systemctl is-active nginx
quick i-demo-web01
doctor i-demo-web01
report demo-report.md
currency USD
price ecs.g8i.large Hour
quit
```

A scripted smoke test is included:

```bash
./scripts/smoke-demo.sh
```

## Configure Alibaba Cloud CLI

Install `aliyun-cli`, then:

```bash
aliyun configure
```

Or use a named profile:

```bash
aliyun configure --profile prod
```

Run `a1s`:

```bash
go run ./cmd/a1s --region me-central-1 --profile prod --currency SAR
```

Read-only production check:

```bash
go run ./cmd/a1s --region me-central-1 --profile prod --currency SAR --read-only
```

## Main commands

```text
ecs
metrics <instance-id> [minutes]
run <instance-id> <shell command>
quick <instance-id>
doctor [instance-id]
bill [YYYY-MM]
price <instance-type> [Hour|Month|Year]
report [file.md]
ai <question>
currency USD|SAR
help
quit
```

## ECS remote command execution

Example:

```text
run i-xxxxxxxx systemctl is-active nginx
```

`a1s` calls Alibaba Cloud ECS Cloud Assistant `RunCommand`, receives the `InvokeId`, then polls `DescribeInvocationResults` until the invocation finishes or times out.

Use `--read-only` to block execution:

```bash
a1s --read-only
```

## Metrics

Example:

```text
metrics i-xxxxxxxx 60
```

The MVP queries:

- Namespace: `acs_ecs_dashboard`
- Metric: `CPUUtilization`
- Period: 60 seconds

Future versions should add dynamic metric discovery and service-specific metric panels.

## Pricing and currency

Query an ECS price:

```text
price ecs.g8i.large Hour
```

Switch display currency:

```text
currency SAR
currency USD
```

Default display conversion:

```text
1 USD = 3.75 SAR
```

Override it:

```bash
export A1S_SAR_PER_USD=3.75
```

PowerShell:

```powershell
$env:A1S_SAR_PER_USD="3.75"
```

The Alibaba Cloud API amount/currency is retained as the authoritative source. Conversion is display-only.

## AI setup

Any OpenAI-compatible endpoint can be used.

```bash
export A1S_AI_BASE_URL="https://your-endpoint/v1"
export A1S_AI_API_KEY="..."
export A1S_AI_MODEL="your-model"
```

This also works with compatible local gateways such as vLLM or compatible hosted model services.

Then:

```text
ai why could this ECS fleet be unhealthy?
doctor i-xxxxxxxx
report weekly.md
```

The AI layer is intentionally advisory. It does not automatically execute generated commands.

## Recommended live test order

1. Start with `--read-only`.
2. Run `ecs` and verify inventory.
3. Run `metrics <instance-id>`.
4. Run `price <instance-type> Hour`.
5. Run `bill` if the RAM identity has billing permissions.
6. Remove `--read-only` only on a safe test ECS.
7. Run a harmless command such as `uptime`.
8. Verify the returned stdout and exit code.
9. Enable AI only after the non-AI data path is working.

## RAM permissions for an MVP test

Use least privilege. The exact policy depends on your environment, but the tool needs read access for the resources you inspect and Cloud Assistant permissions if you enable execution.

Typical API actions used by this MVP:

```text
ecs:DescribeInstances
ecs:DescribePrice
ecs:RunCommand
ecs:DescribeInvocationResults
cms:DescribeMetricList
bssopenapi:QueryAccountBill
```

Do not grant mutation permissions if you only need read-only monitoring.

## Roadmap

### v0.2 — richer TUI

- arrow-key navigation
- searchable/filterable tables
- resource details side panel
- region/profile switcher
- themes
- auto-refresh
- copy resource ID
- open resource in Alibaba Cloud Console

### v0.3 — more Alibaba Cloud resources

- VPC / vSwitch / route tables
- EIP / NAT Gateway / VPN Gateway / CEN
- ALB / CLB / NLB and backend health
- ACK clusters / node pools / pods
- RDS
- OSS / NAS
- ACR
- Auto Scaling

### v0.4 — observability

- dynamic CloudMonitor metric catalog
- CPU / memory / disk / network dashboards
- SLS logs
- ActionTrail activity
- CloudMonitor alarms
- Security Center findings
- correlation timeline

### v0.5 — AI operations

- AI incident summary
- AI metric anomaly explanation
- AI log analysis
- AI-generated daily/weekly reports
- safe suggested commands with explicit approval
- resource relationship reasoning
- cost optimization suggestions

### v0.6 — FinOps

- billing by product/resource
- subscription vs pay-as-you-go inventory
- expiry view
- region-aware price comparison
- USD/SAR display preferences
- cost trend charts
- idle/unattached resource findings

## Project structure

```text
cmd/a1s/                 executable
internal/aliyun/         aliyun-cli/OpenAPI adapter
internal/ai/             OpenAI-compatible AI client
internal/config/         env/flag configuration
internal/currency/       display currency conversion
internal/model/          data models
internal/report/         doctor + Markdown report
internal/ui/             terminal UI
scripts/                 test helpers
docs/                    design notes
```

## References / inspiration

- g1c: https://terminaltrove.com/g1c/
- e1s: https://github.com/keidarcy/e1s
- Alibaba Cloud CLI: https://github.com/aliyun/aliyun-cli
- ECS RunCommand documentation
- ECS DescribeInvocationResults documentation
- CloudMonitor DescribeMetricList documentation
- ECS DescribePrice documentation

## Status

This ZIP is an **MVP/prototype**, not a production-ready operations platform. The included demo mode and unit tests are intended to let you validate the interaction model before expanding the codebase.
