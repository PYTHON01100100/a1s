# Contributing to a1s

Thanks for considering a contribution — bug reports, features, and doc fixes
are all welcome.

## Development setup

Requires Go 1.23+.

```bash
git clone https://github.com/PYTHON01100100/a1s.git
cd a1s
go build ./...
go test ./...
```

Try it without an Alibaba Cloud account:

```bash
go run ./cmd/a1s --sample-data
```

`--sample-data` runs against local, clearly-fake ECS instances, so lifecycle
actions (`start`, `stop`, `stop eco`, `reboot`, `terminate`), filtering, and
diagnostics can all be exercised safely.

## Project layout

```text
cmd/a1s/            entry point, flag parsing
internal/aliyun/    shells out to the aliyun CLI; owns all cloud calls
internal/config/    env/aliyun-cli profile detection
internal/currency/  display currency conversion
internal/model/     shared data types
internal/report/    Markdown report generation
internal/ui/        the interactive terminal UI (commands, rendering, line editor)
```

## Before opening a PR

- `go build ./...`, `go vet ./...`, and `go test ./...` should all pass.
- `gofmt -l .` should report no diffs for files you touched (ignore
  pre-existing line-ending noise in files you didn't change).
- Keep changes focused — unrelated formatting or refactors make a PR harder
  to review.
- If you add a command or flag, update `README.md` and the in-app `help` /
  `--help` text together so they don't drift.
- Add or update a test when you touch `internal/aliyun` (mock the `Runner`
  interface — see `internal/aliyun/client_test.go`) or pure logic like
  filtering/theme switching in `internal/ui`.

## Design principles to keep in mind

- **a1s never handles Alibaba Cloud credentials.** All cloud calls shell out
  to the official `aliyun` CLI; profile/region are only ever read, never
  stored or transmitted by a1s itself.
- **Destructive actions confirm.** Anything that deletes a resource
  (`terminate`) must ask for explicit confirmation, and `--read-only` must
  block every mutating action.
- **No AI in this release, by design.** An AI copilot is on the roadmap but
  intentionally deferred — please keep unrelated AI/chat features out of PRs
  targeting this phase of the project. See `docs/DESIGN.md`.

## Reporting bugs / requesting features

Open a GitHub issue with:

- What you ran (the exact command/flags) and what you expected.
- What happened instead — include the error text if there was one.
- Your OS and whether you were using `--sample-data`, `--demo`, or a real
  account.

## Code of conduct

Be respectful and constructive. Assume good faith, disagree on substance,
and keep discussion focused on the project.
