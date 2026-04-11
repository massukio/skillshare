# CLI E2E Runbook: Collect Multiple Targets

Validates that `ss collect` requires an explicit target name or `--all` when
multiple eligible targets are configured. Covers both skills and agents in
global and project modes.

## Scope

- Global `collect` plain-text warning for skills
- Global `collect --json` error envelope for skills
- Global `collect agents` plain-text warning
- Global `collect agents --json` error envelope
- Project `collect -p` plain-text warning for skills
- Project `collect -p --json` error envelope for skills
- Project `collect -p agents` plain-text warning
- Project `collect -p agents --json` error envelope

## Environment

- Run inside devcontainer via mdproof
- Global steps use mdproof's default initialized HOME
- Project steps create a fresh project under `/tmp`
- JSON error steps intentionally swallow the non-zero exit code and assert on
  the returned error object

## Steps

### Step 1: Global collect skills warns when multiple targets exist

```bash
set -e
rm -rf /tmp/collect-multi-global-skills
mkdir -p \
  /tmp/collect-multi-global-skills/.config \
  /tmp/collect-multi-global-skills/.local/share \
  /tmp/collect-multi-global-skills/.local/state \
  /tmp/collect-multi-global-skills/.cache \
  /tmp/collect-multi-global-skills/.claude \
  /tmp/collect-multi-global-skills/.cursor
HOME=/tmp/collect-multi-global-skills /workspace/bin/skillshare init -g --force --all-targets --no-git --no-skill >/dev/null 2>&1

OUTPUT=$(HOME=/tmp/collect-multi-global-skills /workspace/bin/skillshare collect -g 2>&1)
printf '%s\n' "$OUTPUT"
```

Expected:
- exit_code: 0
- Multiple targets found. Specify a target name or use --all
- claude
- cursor
- universal

### Step 2: Global collect skills --json returns an error envelope

```bash
set -e
rm -rf /tmp/collect-multi-global-skills-json
mkdir -p \
  /tmp/collect-multi-global-skills-json/.config \
  /tmp/collect-multi-global-skills-json/.local/share \
  /tmp/collect-multi-global-skills-json/.local/state \
  /tmp/collect-multi-global-skills-json/.cache \
  /tmp/collect-multi-global-skills-json/.claude \
  /tmp/collect-multi-global-skills-json/.cursor
HOME=/tmp/collect-multi-global-skills-json /workspace/bin/skillshare init -g --force --all-targets --no-git --no-skill >/dev/null 2>&1

OUTPUT=$(HOME=/tmp/collect-multi-global-skills-json /workspace/bin/skillshare collect --json -g 2>&1 || true)
printf '%s\n' "$OUTPUT"
```

Expected:
- exit_code: 0
- jq: .error == "multiple targets found; specify a target name or use --all"

### Step 3: Global collect agents warns when multiple agent targets exist

```bash
set -e
rm -rf /tmp/collect-multi-global-agents
mkdir -p \
  /tmp/collect-multi-global-agents/.config \
  /tmp/collect-multi-global-agents/.local/share \
  /tmp/collect-multi-global-agents/.local/state \
  /tmp/collect-multi-global-agents/.cache \
  /tmp/collect-multi-global-agents/.claude \
  /tmp/collect-multi-global-agents/.cursor
HOME=/tmp/collect-multi-global-agents /workspace/bin/skillshare init -g --force --all-targets --no-git --no-skill >/dev/null 2>&1

OUTPUT=$(HOME=/tmp/collect-multi-global-agents /workspace/bin/skillshare collect agents -g 2>&1)
printf '%s\n' "$OUTPUT"
```

Expected:
- exit_code: 0
- Multiple targets found. Specify a target name or use --all
- claude
- cursor

### Step 4: Global collect agents --json returns an error envelope

```bash
set -e
rm -rf /tmp/collect-multi-global-agents-json
mkdir -p \
  /tmp/collect-multi-global-agents-json/.config \
  /tmp/collect-multi-global-agents-json/.local/share \
  /tmp/collect-multi-global-agents-json/.local/state \
  /tmp/collect-multi-global-agents-json/.cache \
  /tmp/collect-multi-global-agents-json/.claude \
  /tmp/collect-multi-global-agents-json/.cursor
HOME=/tmp/collect-multi-global-agents-json /workspace/bin/skillshare init -g --force --all-targets --no-git --no-skill >/dev/null 2>&1

OUTPUT=$(HOME=/tmp/collect-multi-global-agents-json /workspace/bin/skillshare collect agents --json -g 2>&1 || true)
printf '%s\n' "$OUTPUT"
```

Expected:
- exit_code: 0
- jq: .error == "multiple targets found; specify a target name or use --all"

### Step 5: Project collect skills warns when multiple targets exist

```bash
set -e
rm -rf /tmp/collect-multi-project-skills
mkdir -p /tmp/collect-multi-project-skills
cd /tmp/collect-multi-project-skills
ss init -p --targets claude,cursor >/dev/null 2>&1

OUTPUT=$(ss collect -p 2>&1)
printf '%s\n' "$OUTPUT"
```

Expected:
- exit_code: 0
- Multiple targets found. Specify a target name or use --all
- claude
- cursor

### Step 6: Project collect skills --json returns an error envelope

```bash
set -e
rm -rf /tmp/collect-multi-project-skills-json
mkdir -p /tmp/collect-multi-project-skills-json
cd /tmp/collect-multi-project-skills-json
ss init -p --targets claude,cursor >/dev/null 2>&1

OUTPUT=$(ss collect --json -p 2>&1 || true)
printf '%s\n' "$OUTPUT"
```

Expected:
- exit_code: 0
- jq: .error == "multiple targets found; specify a target name or use --all"

### Step 7: Project collect agents warns when multiple agent targets exist

```bash
set -e
rm -rf /tmp/collect-multi-project-agents
mkdir -p /tmp/collect-multi-project-agents
cd /tmp/collect-multi-project-agents
ss init -p --targets claude,cursor >/dev/null 2>&1

OUTPUT=$(ss collect agents -p 2>&1)
printf '%s\n' "$OUTPUT"
```

Expected:
- exit_code: 0
- Multiple targets found. Specify a target name or use --all
- claude
- cursor

### Step 8: Project collect agents --json returns an error envelope

```bash
set -e
rm -rf /tmp/collect-multi-project-agents-json
mkdir -p /tmp/collect-multi-project-agents-json
cd /tmp/collect-multi-project-agents-json
ss init -p --targets claude,cursor >/dev/null 2>&1

OUTPUT=$(ss collect agents --json -p 2>&1 || true)
printf '%s\n' "$OUTPUT"
```

Expected:
- exit_code: 0
- jq: .error == "multiple targets found; specify a target name or use --all"

## Pass Criteria

- All 8 steps pass
- Plain-text collect warns instead of silently collecting from multiple targets
- JSON collect returns a structured error envelope with the exact target-selection error
