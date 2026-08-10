# a1s UI Upgrade v2

Drop these files into the matching paths in your `a1s` repository, then rebuild.

## What changed

- `--demo` is now UI-only: no Alibaba Cloud calls and no fake ECS/billing/metrics are displayed.
- `--sample-data` explicitly enables the old local sample-data behavior.
- Better `--help` output.
- Quick ECS shortcuts after selecting an instance: `uptime`, `disk`, `memory`, `failed`, `ports`, and `quick`.
- Ollama integration:
  - `ollama status`
  - `ollama models`
  - `ollama start`
  - `ollama use <model>`
  - `ollama ask <question>`
  - after `ollama use <model>`, normal `ai <question>` uses Ollama through its OpenAI-compatible `/v1/chat/completions` API.
- Keeps aliyun-cli profile/region auto-detection from the previous upgrade.

## Build

```bash
go test ./...
go build -o bin/a1s ./cmd/a1s
```

## Real Alibaba Cloud account

Do **not** use `--demo`:

```bash
aliyun configure list
./bin/a1s --currency SAR
```

Only resources returned by the configured aliyun-cli account/region are displayed.

## UI-only test

```bash
./bin/a1s --demo --currency SAR
```

No resources or billing data are invented in this mode.

## Explicit sample-data test

```bash
./bin/a1s --sample-data --currency SAR
```

Use this only when you deliberately want fake local data for screenshots or smoke tests.

## Ollama

If Ollama is installed:

```text
ollama status
ollama start
ollama models
ollama use qwen3:8b
ai explain the health of my selected ECS
```

`ollama use` configures the existing a1s OpenAI-compatible AI client to use `http://127.0.0.1:11434/v1` for the current session.
