# a1s UI Upgrade v4

This patch focuses on terminal usability and help correctness.

## Fixed

- The prompt no longer contains a newline, so repainting while typing stays on one row instead of moving the cursor up/down the terminal.
- Linux raw input keeps output post-processing enabled for cleaner Fedora/WSL terminal rendering.
- Up/Down history preserves the unfinished draft when returning to the newest entry.
- Left/Right cursor movement remains supported; Home, End, and Delete are also supported.
- Tab autocomplete remains available for commands and ECS names.

## Help

Inside a1s:

```text
--help
help
help run
run --help
metrics --help
ai --help
bill --help
```

Every main interactive command now has focused usage, examples, and a short explanation.

Outside a1s:

```bash
./bin/a1s --help
```

shows startup flags, account behavior, modes, examples, and interactive controls.

## Build

Copy this patch over the repository and run:

```bash
gofmt -w cmd internal
go test ./...
go build -o bin/a1s ./cmd/a1s
```
