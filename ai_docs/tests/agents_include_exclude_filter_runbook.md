# CLI E2E Runbook: Agent Include/Exclude Filters

Validates `targets.<name>.agents.include` and `targets.<name>.agents.exclude`
are applied during `sync agents`, including pruning agents that were synced
before the filter was added.

## Scope

- Global-mode agent filters under `targets.<name>.agents`
- Include filter keeps only matching agents and prunes prior non-matching links
- Exclude filter removes matching agents from the target on re-sync
- Nested agent paths are filtered by flattened target filename

## Environment

Run inside devcontainer via mdproof.
Each step is self-contained because mdproof setup resets config between steps.
Use `-g` to force global mode.

## Steps

### 1. Include filter keeps matching agents and prunes previously synced non-matches

```bash
set -e
BASE=~/.config/skillshare
AGENTS_DIR="$BASE/agents"
SKILLS_DIR="$BASE/skills"
TARGET=~/.claude/agents

rm -rf "$AGENTS_DIR" "$SKILLS_DIR" "$TARGET"
mkdir -p "$AGENTS_DIR" "$SKILLS_DIR" "$TARGET"

cat > "$AGENTS_DIR/team-alpha.md" <<'EOF'
# Team Alpha
EOF
cat > "$AGENTS_DIR/team-beta.md" <<'EOF'
# Team Beta
EOF
cat > "$AGENTS_DIR/personal.md" <<'EOF'
# Personal
EOF

cat > "$BASE/config.yaml" <<'EOF'
source: ~/.config/skillshare/skills
targets:
  claude:
    agents:
      path: ~/.claude/agents
EOF

ss sync agents -g >/dev/null
find "$TARGET" -maxdepth 1 -type l -printf 'before: %f\n' | sort

cat > "$BASE/config.yaml" <<'EOF'
source: ~/.config/skillshare/skills
targets:
  claude:
    agents:
      path: ~/.claude/agents
      include: [team-*]
EOF

ss sync agents -g >/dev/null
find "$TARGET" -maxdepth 1 \( -type l -o -type f \) -printf 'after: %f\n' | sort
test -L "$TARGET/team-alpha.md" && echo "team-alpha linked=yes" || echo "team-alpha linked=no"
test -L "$TARGET/team-beta.md" && echo "team-beta linked=yes" || echo "team-beta linked=no"
test ! -e "$TARGET/personal.md" && echo "personal present=no" || echo "personal present=yes"
```

Expected:
- exit_code: 0
- before: personal.md
- before: team-alpha.md
- before: team-beta.md
- after: team-alpha.md
- after: team-beta.md
- Not after: personal.md
- team-alpha linked=yes
- team-beta linked=yes
- personal present=no

### 2. Exclude filter prunes matching agents that were previously synced

```bash
set -e
BASE=~/.config/skillshare
AGENTS_DIR="$BASE/agents"
SKILLS_DIR="$BASE/skills"
TARGET=~/.claude/agents

rm -rf "$AGENTS_DIR" "$SKILLS_DIR" "$TARGET"
mkdir -p "$AGENTS_DIR" "$SKILLS_DIR" "$TARGET"

cat > "$AGENTS_DIR/stable-reviewer.md" <<'EOF'
# Stable Reviewer
EOF
cat > "$AGENTS_DIR/draft-notes.md" <<'EOF'
# Draft Notes
EOF
cat > "$AGENTS_DIR/draft-checklist.md" <<'EOF'
# Draft Checklist
EOF

cat > "$BASE/config.yaml" <<'EOF'
source: ~/.config/skillshare/skills
targets:
  claude:
    agents:
      path: ~/.claude/agents
EOF

ss sync agents -g >/dev/null
find "$TARGET" -maxdepth 1 -type l -printf 'before: %f\n' | sort

cat > "$BASE/config.yaml" <<'EOF'
source: ~/.config/skillshare/skills
targets:
  claude:
    agents:
      path: ~/.claude/agents
      exclude: [draft-*]
EOF

ss sync agents -g >/dev/null
find "$TARGET" -maxdepth 1 \( -type l -o -type f \) -printf 'after: %f\n' | sort
test -L "$TARGET/stable-reviewer.md" && echo "stable-reviewer linked=yes" || echo "stable-reviewer linked=no"
test ! -e "$TARGET/draft-notes.md" && echo "draft-notes present=no" || echo "draft-notes present=yes"
test ! -e "$TARGET/draft-checklist.md" && echo "draft-checklist present=no" || echo "draft-checklist present=yes"
```

Expected:
- exit_code: 0
- before: draft-checklist.md
- before: draft-notes.md
- before: stable-reviewer.md
- after: stable-reviewer.md
- Not after: draft-notes.md
- Not after: draft-checklist.md
- stable-reviewer linked=yes
- draft-notes present=no
- draft-checklist present=no

### 3. Nested agents are filtered by flattened target filename

```bash
set -e
BASE=~/.config/skillshare
AGENTS_DIR="$BASE/agents"
SKILLS_DIR="$BASE/skills"
TARGET=~/.claude/agents

rm -rf "$AGENTS_DIR" "$SKILLS_DIR" "$TARGET"
mkdir -p "$AGENTS_DIR/team/backend" "$AGENTS_DIR/solo" "$SKILLS_DIR" "$TARGET"

cat > "$AGENTS_DIR/team/backend/reviewer.md" <<'EOF'
# Nested Team Reviewer
EOF
cat > "$AGENTS_DIR/solo/helper.md" <<'EOF'
# Solo Helper
EOF

cat > "$BASE/config.yaml" <<'EOF'
source: ~/.config/skillshare/skills
targets:
  claude:
    agents:
      path: ~/.claude/agents
      include: [team__*]
EOF

ss sync agents -g >/dev/null
find "$TARGET" -maxdepth 1 \( -type l -o -type f \) -printf 'after: %f\n' | sort

test -L "$TARGET/team__backend__reviewer.md" && echo "team__backend__reviewer linked=yes" || echo "team__backend__reviewer linked=no"
test ! -e "$TARGET/solo__helper.md" && echo "solo__helper present=no" || echo "solo__helper present=yes"
```

Expected:
- exit_code: 0
- after: team__backend__reviewer.md
- Not after: solo__helper.md
- team__backend__reviewer linked=yes
- solo__helper present=no

## Pass Criteria

- Include filters prune already-synced non-matching agents on the next `sync agents`
- Exclude filters remove matching agents from the target while keeping non-matching ones
- Nested agent paths are matched against flattened target filenames
