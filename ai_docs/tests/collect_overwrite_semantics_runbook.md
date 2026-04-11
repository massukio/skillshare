# CLI E2E Runbook: Collect Overwrite Semantics

Validates source-conflict behavior for `ss collect`: default collect skips
existing source content, `--force` overwrites, and `--json` implies force.
Covers skills and agents in global and project modes.

## Scope

- Global skills skip without force
- Global skills `--json` overwrites existing source
- Global agents skip without force
- Global agents `--force --json` overwrites existing source
- Project skills skip without force
- Project skills `--force --json` overwrites existing source
- Project agents skip without force
- Project agents `--json` overwrites existing source

## Environment

- Run inside devcontainer via mdproof
- Global steps use a fresh mdproof HOME
- Project steps create fresh projects under `/tmp`
- Skip steps use `printf 'y\n'` to confirm the collect action, then assert the
  source content remains unchanged
- Overwrite steps keep stdout as pure JSON and verify file content silently

## Steps

### Step 1: Global collect skills skips an existing source skill by default

```bash
set -e
mkdir -p ~/.config/skillshare/skills/dupe-skill ~/.claude/skills/dupe-skill
cat > ~/.config/skillshare/skills/dupe-skill/SKILL.md <<'EOF'
# Source Skill
Source version survives.
EOF
cat > ~/.claude/skills/dupe-skill/SKILL.md <<'EOF'
# Target Skill
Target version should not overwrite.
EOF

OUTPUT=$(printf 'y\n' | ss collect claude -g 2>&1)
grep -q "Source version survives." ~/.config/skillshare/skills/dupe-skill/SKILL.md
printf '%s\n' "$OUTPUT"
```

Expected:
- exit_code: 0
- already exists in source
- 1 skipped

### Step 2: Global collect skills --json overwrites the existing source skill

```bash
set -e
mkdir -p ~/.config/skillshare/skills/dupe-skill ~/.claude/skills/dupe-skill
cat > ~/.config/skillshare/skills/dupe-skill/SKILL.md <<'EOF'
# Source Skill
Old source version.
EOF
cat > ~/.claude/skills/dupe-skill/SKILL.md <<'EOF'
# Target Skill
Target version wins via json.
EOF

OUTPUT=$(ss collect claude --json -g)
grep -q "Target version wins via json." ~/.config/skillshare/skills/dupe-skill/SKILL.md
printf '%s\n' "$OUTPUT"
```

Expected:
- exit_code: 0
- jq: .pulled == ["dupe-skill"]
- jq: (.skipped | length) == 0
- jq: (.failed | length) == 0

### Step 3: Global collect agents skips an existing source agent by default

```bash
set -e
mkdir -p ~/.config/skillshare/agents ~/.claude/agents
cat > ~/.config/skillshare/agents/dupe-agent.md <<'EOF'
# Source Agent
Source agent survives.
EOF
cat > ~/.claude/agents/dupe-agent.md <<'EOF'
# Target Agent
Target agent should not overwrite.
EOF

OUTPUT=$(printf 'y\n' | ss collect agents claude -g 2>&1)
grep -q "Source agent survives." ~/.config/skillshare/agents/dupe-agent.md
printf '%s\n' "$OUTPUT"
```

Expected:
- exit_code: 0
- already exists in source
- 1 skipped

### Step 4: Global collect agents --force --json overwrites the existing source agent

```bash
set -e
mkdir -p ~/.config/skillshare/agents ~/.claude/agents
cat > ~/.config/skillshare/agents/dupe-agent.md <<'EOF'
# Source Agent
Old source agent.
EOF
cat > ~/.claude/agents/dupe-agent.md <<'EOF'
# Target Agent
Target agent force wins.
EOF

OUTPUT=$(ss collect agents claude --force --json -g)
grep -q "Target agent force wins." ~/.config/skillshare/agents/dupe-agent.md
printf '%s\n' "$OUTPUT"
```

Expected:
- exit_code: 0
- jq: .pulled == ["dupe-agent.md"]
- jq: (.skipped | length) == 0
- jq: (.failed | length) == 0

### Step 5: Project collect skills skips an existing source skill by default

```bash
set -e
rm -rf /tmp/collect-overwrite-project-skills-skip
mkdir -p /tmp/collect-overwrite-project-skills-skip
cd /tmp/collect-overwrite-project-skills-skip
ss init -p --targets claude >/dev/null 2>&1

mkdir -p .skillshare/skills/dupe-skill .claude/skills/dupe-skill
cat > .skillshare/skills/dupe-skill/SKILL.md <<'EOF'
# Source Skill
Project source survives.
EOF
cat > .claude/skills/dupe-skill/SKILL.md <<'EOF'
# Target Skill
Project target should not overwrite.
EOF

OUTPUT=$(printf 'y\n' | ss collect claude -p 2>&1)
grep -q "Project source survives." .skillshare/skills/dupe-skill/SKILL.md
printf '%s\n' "$OUTPUT"
```

Expected:
- exit_code: 0
- already exists in source
- 1 skipped

### Step 6: Project collect skills --force --json overwrites the existing source skill

```bash
set -e
rm -rf /tmp/collect-overwrite-project-skills-force
mkdir -p /tmp/collect-overwrite-project-skills-force
cd /tmp/collect-overwrite-project-skills-force
ss init -p --targets claude >/dev/null 2>&1

mkdir -p .skillshare/skills/dupe-skill .claude/skills/dupe-skill
cat > .skillshare/skills/dupe-skill/SKILL.md <<'EOF'
# Source Skill
Old project source.
EOF
cat > .claude/skills/dupe-skill/SKILL.md <<'EOF'
# Target Skill
Project force overwrite wins.
EOF

OUTPUT=$(ss collect claude --force --json -p)
grep -q "Project force overwrite wins." .skillshare/skills/dupe-skill/SKILL.md
printf '%s\n' "$OUTPUT"
```

Expected:
- exit_code: 0
- jq: .pulled == ["dupe-skill"]
- jq: (.skipped | length) == 0
- jq: (.failed | length) == 0

### Step 7: Project collect agents skips an existing source agent by default

```bash
set -e
rm -rf /tmp/collect-overwrite-project-agents-skip
mkdir -p /tmp/collect-overwrite-project-agents-skip
cd /tmp/collect-overwrite-project-agents-skip
ss init -p --targets claude >/dev/null 2>&1

mkdir -p .skillshare/agents .claude/agents
cat > .skillshare/agents/dupe-agent.md <<'EOF'
# Source Agent
Project source agent survives.
EOF
cat > .claude/agents/dupe-agent.md <<'EOF'
# Target Agent
Project target agent should not overwrite.
EOF

OUTPUT=$(printf 'y\n' | ss collect agents claude -p 2>&1)
grep -q "Project source agent survives." .skillshare/agents/dupe-agent.md
printf '%s\n' "$OUTPUT"
```

Expected:
- exit_code: 0
- already exists in source
- 1 skipped

### Step 8: Project collect agents --json overwrites the existing source agent

```bash
set -e
rm -rf /tmp/collect-overwrite-project-agents-json
mkdir -p /tmp/collect-overwrite-project-agents-json
cd /tmp/collect-overwrite-project-agents-json
ss init -p --targets claude >/dev/null 2>&1

mkdir -p .skillshare/agents .claude/agents
cat > .skillshare/agents/dupe-agent.md <<'EOF'
# Source Agent
Old project source agent.
EOF
cat > .claude/agents/dupe-agent.md <<'EOF'
# Target Agent
Project json overwrite wins.
EOF

OUTPUT=$(ss collect agents claude --json -p)
grep -q "Project json overwrite wins." .skillshare/agents/dupe-agent.md
printf '%s\n' "$OUTPUT"
```

Expected:
- exit_code: 0
- jq: .pulled == ["dupe-agent.md"]
- jq: (.skipped | length) == 0
- jq: (.failed | length) == 0

## Pass Criteria

- All 8 steps pass
- Default collect preserves existing source content for both skills and agents
- Explicit `--force` and implicit force via `--json` overwrite existing source content
