# a1s v0.5 TUI + Chat Upgrade

This patch moves a1s toward the interaction model used by e1s/g1c/k9s and the conversational terminal workflow used by Hermes Agent.

## What changed

- Alibaba Cloud orange/dark dashboard with compact context + AI status panels.
- K9s-style `:` command aliases: `:ecs`, `:run`, `:metrics`, `:bill`, `:chat`, `:dashboard`, `:help`.
- Existing Linux line editor remains: Up/Down history, Left/Right cursor movement, Home/End/Delete and Tab completion.
- ECS-name completion continues to work for `run`, `use` and `metrics`.
- New `chat` mode inspired by Hermes terminal UX.
- Chat slash commands: `/ecs`, `/use`, `/run`, `/metrics`, `/doctor`, `/providers`, `/models`, `/model`, `/clear`, `/exit`.
- Automatic AI provider detection:
  1. Explicit `A1S_AI_BASE_URL` + `A1S_AI_MODEL` configuration wins.
  2. Otherwise a running local Ollama is detected automatically through `http://127.0.0.1:11434/api/tags`.
  3. The first installed Ollama model is selected automatically; change it with `ai model <number|name>` or `/model` inside chat.
- Unified provider commands: `ai providers`, `ai use ollama`, `ai use openai-compatible`, `ai models`, `ai model`, `ai ask`.
- Cloud-changing actions are never silently executed by AI. In chat, execution requires an explicit `/run ...` command.
- `chat --help`, `ai --help`, `dashboard --help` and the existing per-command help system are supported.

## Install over the current repository

```bash
unzip a1s-ui-upgrade-v5.zip
cp -r a1s-ui-upgrade-v5/cmd ./
cp -r a1s-ui-upgrade-v5/internal ./
gofmt -w cmd internal
go test ./...
go build -o bin/a1s ./cmd/a1s
```

## Try it

```bash
./bin/a1s --currency SAR
```

Then:

```text
:ecs
run test uptime
ai providers
chat
```

Inside chat:

```text
/providers
/models
/model 1
/ecs
/run test uptime
/metrics test 30
/exit
```

## Ollama auto-detection

If Ollama is already running and has local models, a1s selects it automatically. If installed but stopped:

```bash
ollama serve
```

or from a1s:

```text
ollama start
```

Then run `chat` again.

## Design direction

The next architectural step should be a true event-driven full-screen TUI (resource cursor, Enter details, `/` filter/search, live refresh, modal command runner, profile/region pickers). This patch deliberately keeps the current zero-dependency codebase while introducing the navigation and chat interaction model first.
