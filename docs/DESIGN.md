# a1s design notes

## Product idea

a1s should feel like a cloud operations cockpit in the terminal, not a thin wrapper around `aliyun`.

Core loop:

1. Discover resources.
2. Observe current state and metrics.
3. Manage lifecycle (start/stop/reboot/terminate) with explicit confirmation
   for anything destructive.
4. Run safe diagnostics when explicitly requested.
5. Produce concise operational or FinOps reports.

An AI copilot (natural-language questions over collected evidence, a chat
console) is intentionally deferred to a later release. The current focus is
making direct instance management fast, predictable, and safe without it.

## Safety model

- `--read-only` disables all mutating actions (start/stop/reboot/terminate/run).
- `terminate` always asks for interactive confirmation.
- Command output is evidence; nothing is inferred beyond what a command returned.
- Prefer RAM least privilege and temporary credentials/roles.
- When AI returns, it must never auto-run commands or claim a command ran
  without real output backing it.

## Future GUI/TUI tabs

```text
[ECS] [ACK] [RDS] [Network] [Storage] [Metrics] [Logs] [Cost] [AI]
```

## FinOps UX

The user chooses a display currency independently from Alibaba Cloud settlement currency. Each price card should preserve:

- API currency
- original price
- discount
- trade price
- display currency
- display FX source/value

Future versions can add configurable FX providers. The MVP intentionally keeps conversion deterministic with `A1S_SAR_PER_USD`.
