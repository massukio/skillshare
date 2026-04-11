# CLI E2E Runbook: Collect Skills and Agents

Validates `ss collect` for both resource kinds across global and project modes.
Each step is self-contained because mdproof setup provides a fresh initialized
environment and `/tmp` is shared across runs.

## Scope

- Global `collect` for skills with `--dry-run --json`
- Global `collect` for skills with real writes to source
- Global `collect agents` with `--dry-run --json`
- Global `collect agents` with real writes to source
- Project `collect -p` JSON output for skills
- Project `collect -p agents --dry-run --json` does not write
- Project `collect -p agents --json` writes to `.skillshare/agents`

## Environment

- Run inside devcontainer via mdproof
- Global steps use `-g` and explicitly target `claude`
- Project steps create fresh projects under `/tmp`
- Assertions rely on `--json` output plus silent file-system checks

## Steps

### Step 1: Global collect skills dry-run previews without writing

```bash
set -e
rm -rf ~/.claude/skills/collect-dry-skill ~/.config/skillshare/skills/collect-dry-skill
mkdir -p ~/.claude/skills/collect-dry-skill
cat > ~/.claude/skills/collect-dry-skill/SKILL.md <<'EOF'
# Collect Dry Skill
EOF

OUTPUT=$(ss collect claude --json --dry-run -g)
test ! -e ~/.config/skillshare/skills/collect-dry-skill
printf '%s\n' "$OUTPUT"
```

Expected:
- exit_code: 0
- jq: .dry_run == true
- jq: .pulled == ["collect-dry-skill"]
- jq: (.skipped | length) == 0
- jq: (.failed | length) == 0

### Step 2: Global collect skills copies a local skill to source

```bash
set -e
rm -rf \
  ~/.claude/skills/collect-dry-skill \
  ~/.claude/skills/collect-live-skill \
  ~/.config/skillshare/skills/collect-dry-skill \
  ~/.config/skillshare/skills/collect-live-skill
mkdir -p ~/.claude/skills/collect-live-skill
cat > ~/.claude/skills/collect-live-skill/SKILL.md <<'EOF'
# Collect Live Skill
Collected from target.
EOF

OUTPUT=$(ss collect claude --json -g)
test -f ~/.config/skillshare/skills/collect-live-skill/SKILL.md
grep -q "Collected from target." ~/.config/skillshare/skills/collect-live-skill/SKILL.md
printf '%s\n' "$OUTPUT"
```

Expected:
- exit_code: 0
- jq: .dry_run == false
- jq: .pulled == ["collect-live-skill"]
- jq: (.skipped | length) == 0
- jq: (.failed | length) == 0

### Step 3: Global collect agents dry-run previews without writing

```bash
set -e
rm -f ~/.claude/agents/collect-dry-agent.md ~/.config/skillshare/agents/collect-dry-agent.md
mkdir -p ~/.claude/agents
cat > ~/.claude/agents/collect-dry-agent.md <<'EOF'
---
name: collect-dry-agent
description: Dry-run agent
---
# Collect Dry Agent
EOF

OUTPUT=$(ss collect agents claude --json --dry-run -g)
test ! -e ~/.config/skillshare/agents/collect-dry-agent.md
printf '%s\n' "$OUTPUT"
```

Expected:
- exit_code: 0
- jq: .dry_run == true
- jq: .pulled == ["collect-dry-agent.md"]
- jq: (.skipped | length) == 0
- jq: (.failed | length) == 0

### Step 4: Global collect agents copies a local agent to source

```bash
set -e
rm -f \
  ~/.claude/agents/collect-dry-agent.md \
  ~/.claude/agents/collect-live-agent.md \
  ~/.config/skillshare/agents/collect-dry-agent.md \
  ~/.config/skillshare/agents/collect-live-agent.md
mkdir -p ~/.claude/agents
cat > ~/.claude/agents/collect-live-agent.md <<'EOF'
---
name: collect-live-agent
description: Live collect agent
---
# Collect Live Agent
Collected from target.
EOF

OUTPUT=$(ss collect agents claude --json -g)
test -f ~/.config/skillshare/agents/collect-live-agent.md
grep -q "Collected from target." ~/.config/skillshare/agents/collect-live-agent.md
printf '%s\n' "$OUTPUT"
```

Expected:
- exit_code: 0
- jq: .dry_run == false
- jq: .pulled == ["collect-live-agent.md"]
- jq: (.skipped | length) == 0
- jq: (.failed | length) == 0

### Step 5: Project collect skills outputs JSON and writes into .skillshare/skills

```bash
set -e
rm -rf /tmp/collect-project-skills
mkdir -p /tmp/collect-project-skills
cd /tmp/collect-project-skills
ss init -p --targets claude >/dev/null 2>&1

mkdir -p .claude/skills/project-collect-skill
cat > .claude/skills/project-collect-skill/SKILL.md <<'EOF'
# Project Collect Skill
Collected into project source.
EOF

OUTPUT=$(ss collect claude --json -p)
test -f .skillshare/skills/project-collect-skill/SKILL.md
grep -q "Collected into project source." .skillshare/skills/project-collect-skill/SKILL.md
printf '%s\n' "$OUTPUT"
```

Expected:
- exit_code: 0
- jq: .dry_run == false
- jq: .pulled == ["project-collect-skill"]
- jq: (.skipped | length) == 0
- jq: (.failed | length) == 0

### Step 6: Project collect agents dry-run previews without writing

```bash
set -e
rm -rf /tmp/collect-project-agents-dry
mkdir -p /tmp/collect-project-agents-dry
cd /tmp/collect-project-agents-dry
ss init -p --targets claude >/dev/null 2>&1

mkdir -p .claude/agents
cat > .claude/agents/project-dry-agent.md <<'EOF'
---
name: project-dry-agent
description: Project dry-run agent
---
# Project Dry Agent
EOF

OUTPUT=$(ss collect agents claude --json --dry-run -p)
test ! -e .skillshare/agents/project-dry-agent.md
printf '%s\n' "$OUTPUT"
```

Expected:
- exit_code: 0
- jq: .dry_run == true
- jq: .pulled == ["project-dry-agent.md"]
- jq: (.skipped | length) == 0
- jq: (.failed | length) == 0

### Step 7: Project collect agents outputs JSON and writes into .skillshare/agents

```bash
set -e
rm -rf /tmp/collect-project-agents-live
mkdir -p /tmp/collect-project-agents-live
cd /tmp/collect-project-agents-live
ss init -p --targets claude >/dev/null 2>&1

mkdir -p .claude/agents
cat > .claude/agents/project-live-agent.md <<'EOF'
---
name: project-live-agent
description: Project live agent
---
# Project Live Agent
Collected into project agent source.
EOF

OUTPUT=$(ss collect agents claude --json -p)
test -f .skillshare/agents/project-live-agent.md
grep -q "Collected into project agent source." .skillshare/agents/project-live-agent.md
printf '%s\n' "$OUTPUT"
```

Expected:
- exit_code: 0
- jq: .dry_run == false
- jq: .pulled == ["project-live-agent.md"]
- jq: (.skipped | length) == 0
- jq: (.failed | length) == 0

## Pass Criteria

- All 7 steps exit successfully
- Dry-run steps report `dry_run: true` and do not write to source
- Global collect writes to `~/.config/skillshare/skills` and `~/.config/skillshare/agents`
- Project collect writes to `.skillshare/skills` and `.skillshare/agents`
- All collect commands emit valid JSON without UI noise on stdout
