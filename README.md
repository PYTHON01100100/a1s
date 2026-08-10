# a1s UI Upgrade v3

Drop these files into the matching paths in the a1s repository, then rebuild.

## What changed

- `run <ecs-name> <command>` targets ECS by **instance name**, ID, or row number.
- `use <row|name|instance-id>` supports names too.
- ECS inventory now shows: internal/private IP, external/public IP, instance type, OS, billing/charge type, VPC ID + name, vSwitch ID + name, zone.
- VPC/vSwitch names are resolved from Alibaba Cloud VPC APIs after `DescribeInstances`.
- Linux interactive line editor: Up/Down command history, Left/Right cursor movement, Tab autocomplete.
- Autocomplete suggests a1s commands and ECS names for `run`, `use`, and `metrics`.
- AI is now grouped under `ai`:
  - `ai providers`
  - `ai use ollama`
  - `ai models`
  - `ai model <number|name>`
  - `ai ask <question>`
- Ollama model selection only accepts models already installed on the machine.

## Install

```bash
cp -r cmd internal /path/to/a1s/
cd /path/to/a1s
gofmt -w cmd internal
go test ./...
go build -o bin/a1s ./cmd/a1s
./bin/a1s --currency SAR
```

## Examples

```text
run test pwd
run test uptime
use test
run df -h

ai providers
ai use ollama
ai models
ai model 1
ai ask summarize my ECS environment
```

Alibaba Cloud permissions used for the richer inventory include `ecs:DescribeInstances`, `vpc:DescribeVpcs`, and `vpc:DescribeVSwitches` in addition to the permissions already used by a1s.
