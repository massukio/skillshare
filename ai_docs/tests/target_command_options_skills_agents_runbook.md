# CLI E2E Runbook: Target Command Options (Skills + Agents)

Validates `skillshare target` command options across plain list, JSON list,
target info/settings, and both global/project mode for skill and agent
configuration.

**Origin**: v0.19.x — target settings gained explicit agents options alongside
existing skills options, and docs needed regression coverage against code.

## Scope

- `target help` advertises skill and agent settings flags
- `target list --no-tui` and `target list --json` expose agent metadata for
  supported targets while leaving unsupported targets agent-free
- `target <name>` supports `--mode`, `--target-naming`, include/exclude, and
  the agent counterparts `--agent-mode`, `--add/remove-agent-include/exclude`
- Agent filter flags are rejected for unsupported targets and for
  `agent-mode=symlink`
- Project mode mirrors the same skill/agent target settings behavior

## Environment

Run inside devcontainer via mdproof.
Use `-g`/`-p` explicitly to avoid auto-mode ambiguity.

## Steps

### 1. Help output lists skill and agent target settings

```bash
ss target help
```

Expected:
- exit_code: 0
- --mode
- --agent-mode
- --target-naming
- --add-include
- --add-exclude
- --remove-include
- --remove-exclude
- --add-agent-include
- --add-agent-exclude
- --remove-agent-include
- --remove-agent-exclude

### 2. Global plain target list shows skills and agents sections

```bash
set -e
BASE=~/.config/skillshare
mkdir -p "$BASE/skills/team-alpha" "$BASE/agents" "$HOME/custom-tool/skills"

printf '%s\n' \
  '---' \
  'name: team-alpha' \
  'description: Team Alpha skill' \
  'metadata:' \
  '  targets: [claude]' \
  '---' \
  '# Team Alpha' \
  > "$BASE/skills/team-alpha/SKILL.md"

printf '%s\n' \
  '---' \
  'name: reviewer' \
  'description: Review agent' \
  '---' \
  '# Reviewer' \
  > "$BASE/agents/reviewer.md"

ss target add custom-tool "$HOME/custom-tool/skills" -g
ss target list --no-tui -g
```

Expected:
- exit_code: 0
- claude
- custom-tool
- Skills:
- Agents:
- /.claude/agents

### 3. Global JSON target list includes agent metadata only for supported targets

```bash
ss target list --json -g
```

Expected:
- exit_code: 0
- jq: .targets | length >= 2
- jq: (.targets[] | select(.name == "claude").agentPath | type) == "string"
- jq: (.targets[] | select(.name == "claude").agentMode) == "merge"
- jq: (.targets[] | select(.name == "claude").agentExpectedCount) >= 0
- jq: (.targets[] | select(.name == "custom-tool").agentPath) == null

### 4. Global skill settings flags update target mode, naming, and filters

```bash
set -e
ss target claude --mode copy -g
ss target claude --target-naming standard -g
ss target claude --add-include "team-*" -g
ss target claude --add-exclude "_legacy*" -g
ss target claude -g
```

Expected:
- exit_code: 0
- Changed claude mode: merge -> copy
- Changed claude target naming: flat -> standard
- added include: team-*
- added exclude: _legacy*
- Mode:    copy
- Naming:  standard
- Include: team-*
- Exclude: _legacy*

### 5. Global agent settings flags update agent mode and agent filters

```bash
set -e
ss target claude --agent-mode copy -g
ss target claude --add-agent-include "team-*" -g
ss target claude --add-agent-exclude "draft-*" -g
ss target claude -g
```

Expected:
- exit_code: 0
- Changed claude agent mode: merge -> copy
- added agent include: team-*
- added agent exclude: draft-*
- Agents:
- Mode:    copy
- Include: team-*
- Exclude: draft-*

### 6. Agent filter guard rails reject symlink mode and unsupported targets

```bash
set -e
ss target claude --agent-mode symlink -g
ss target claude -g

set +e
SYMLINK_ERR=$(ss target claude --add-agent-include "retry-*" -g 2>&1)
SYMLINK_STATUS=$?
CUSTOM_ERR=$(ss target custom-tool --add-agent-include "retry-*" -g 2>&1)
CUSTOM_STATUS=$?
set -e

printf 'REJECTED_SYMLINK=%d\n' "$SYMLINK_STATUS"
printf '%s\n' "$SYMLINK_ERR"
printf 'REJECTED_CUSTOM=%d\n' "$CUSTOM_STATUS"
printf '%s\n' "$CUSTOM_ERR"
```

Expected:
- exit_code: 0
- Changed claude agent mode: copy -> symlink
- Filters: ignored in symlink mode
- REJECTED_SYMLINK=1
- ignored in symlink mode
- REJECTED_CUSTOM=1
- target 'custom-tool' does not have an agents path

### 7. Global remove flags clear both skill and agent filters

```bash
set -e
ss target claude --agent-mode copy -g
ss target claude --remove-include "team-*" -g
ss target claude --remove-exclude "_legacy*" -g
ss target claude --remove-agent-include "team-*" -g
ss target claude --remove-agent-exclude "draft-*" -g
ss target claude -g
```

Expected:
- exit_code: 0
- Changed claude agent mode: symlink -> copy
- removed include: team-*
- removed exclude: _legacy*
- removed agent include: team-*
- removed agent exclude: draft-*
- Include: (none)
- Exclude: (none)

### 8. Project mode mirrors skill and agent target settings

```bash
set -e
PROJECT=/tmp/target-options-project
rm -rf "$PROJECT"
mkdir -p "$PROJECT/.skillshare/skills" "$PROJECT/.skillshare/agents"

cat > "$PROJECT/.skillshare/config.yaml" <<'EOF'
targets:
  - claude
EOF

cd "$PROJECT"
ss target claude --mode copy -p
ss target claude --target-naming standard -p
ss target claude --add-include "proj-*" -p
ss target claude --add-exclude "draft-*" -p
ss target claude --agent-mode copy -p
ss target claude --add-agent-include "proj-*" -p
ss target claude --add-agent-exclude "draft-*" -p
ss target claude -p
```

Expected:
- exit_code: 0
- Changed claude mode: merge -> copy
- Changed claude target naming: flat -> standard
- added include: proj-*
- added exclude: draft-*
- Changed claude agent mode: merge -> copy
- added agent include: proj-*
- added agent exclude: draft-*
- Mode:    copy
- Naming:  standard
- Include: proj-*
- Exclude: draft-*
- Agents:

### 9. Project JSON target list includes agent metadata

```bash
cd /tmp/target-options-project
ss target list --json -p
```

Expected:
- exit_code: 0
- jq: .targets | length == 1
- jq: (.targets[0].name) == "claude"
- jq: (.targets[0].agentMode) == "copy"
- jq: (.targets[0].agentInclude) == ["proj-*"]
- jq: (.targets[0].agentExclude) == ["draft-*"]

## Pass Criteria

- The `target` command exposes both skill and agent settings in help and list
  output
- Global and project target settings accept and persist skill/agent mode and
  include/exclude changes
- Agent-only guard rails behave correctly for unsupported targets and
  symlink-mode agent targets
