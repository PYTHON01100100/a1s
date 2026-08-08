# a1s design notes

## Product idea

a1s should feel like a cloud operations cockpit in the terminal, not a thin wrapper around `aliyun`.

Core loop:

1. Discover resources.
2. Observe current state and metrics.
3. Inspect logs/events.
4. Run safe diagnostics when explicitly requested.
5. Feed collected evidence to AI.
6. Produce concise operational or FinOps reports.

## Safety model

- `--read-only` must disable all mutating actions.
- AI never auto-runs commands.
- Command output is evidence; AI text is interpretation.
- Destructive actions should require a confirmation layer in future versions.
- Prefer RAM least privilege and temporary credentials/roles.

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
