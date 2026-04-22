# CLI E2E Runbook: Local Install Source Preservation

Verifies that installing skills from a local path does NOT delete the
original source files — via both CLI and the Web UI API endpoints.

**Origin**: [#139](https://github.com/runkids/skillshare/issues/139) — UI install
deletes local source files when discovery cleanup runs on user's real directory.

## Scope

- CLI `ss install <local-path>` preserves source files after install
- API `POST /api/discover` with local path preserves source files
- API `POST /api/install/batch` with local path preserves source files
- Full UI flow (discover → batch install) preserves source files
- Multi-skill local repos preserve source files through discovery

## Environment

Run inside devcontainer with `ssenv` isolation.
Requires `curl` and `jq` for API testing.

## Steps

### 1. Setup: create local skill source directory

```bash
rm -rf /tmp/ss-local-src /tmp/ss-local-multi
mkdir -p /tmp/ss-local-src/my-precious-skill
cat > /tmp/ss-local-src/my-precious-skill/SKILL.md << 'EOF'
---
name: my-precious-skill
description: Test skill that must not be deleted
---
# My Precious Skill
This file must survive installation.
EOF
echo "important data" > /tmp/ss-local-src/my-precious-skill/README.md
ls -la /tmp/ss-local-src/my-precious-skill/
```

Expected:
- exit_code: 0
- SKILL.md
- README.md

### 2. Setup: create multi-skill local repo

```bash
rm -rf /tmp/ss-local-multi
mkdir -p /tmp/ss-local-multi/alpha /tmp/ss-local-multi/beta
cat > /tmp/ss-local-multi/alpha/SKILL.md << 'EOF'
---
name: alpha
description: Alpha skill
---
# Alpha
EOF
cat > /tmp/ss-local-multi/beta/SKILL.md << 'EOF'
---
name: beta
description: Beta skill
---
# Beta
EOF
echo "alpha data" > /tmp/ss-local-multi/alpha/data.txt
echo "beta data" > /tmp/ss-local-multi/beta/data.txt
find /tmp/ss-local-multi -type f | sort
```

Expected:
- exit_code: 0
- /tmp/ss-local-multi/alpha/SKILL.md
- /tmp/ss-local-multi/beta/SKILL.md
- /tmp/ss-local-multi/alpha/data.txt
- /tmp/ss-local-multi/beta/data.txt

### 3. CLI install: local path preserves source files

```bash
ss install /tmp/ss-local-src/my-precious-skill --json -g
```

Expected:
- exit_code: 0
- jq: .skillName == "my-precious-skill"
- jq: .action == "copied"

### 4. Verify: CLI install source files still exist

```bash
test -f /tmp/ss-local-src/my-precious-skill/SKILL.md && echo "SKILL.md: EXISTS"
test -f /tmp/ss-local-src/my-precious-skill/README.md && echo "README.md: EXISTS"
cat /tmp/ss-local-src/my-precious-skill/README.md
```

Expected:
- exit_code: 0
- SKILL.md: EXISTS
- README.md: EXISTS
- important data

### 5. Cleanup: uninstall CLI-installed skill

```bash
ss uninstall my-precious-skill --json -g
```

Expected:
- exit_code: 0

### 6. Start UI server for API tests

```bash
ss ui -g --no-open --port 19421 &
UI_PID=$!
echo "UI_PID=$UI_PID"
sleep 2
curl -sf http://127.0.0.1:19421/api/overview | jq -r '.version'
```

Expected:
- exit_code: 0
- regex: (dev|v\d+)

### 7. API discover: local path returns skills without deleting source

```bash
curl -sf -X POST http://127.0.0.1:19421/api/discover \
  -H 'Content-Type: application/json' \
  -d '{"source": "/tmp/ss-local-src/my-precious-skill"}'
```

Expected:
- exit_code: 0
- jq: .skills | length == 1
- jq: .skills[0].name == "my-precious-skill"

### 8. Verify: source files survive after /api/discover

```bash
test -f /tmp/ss-local-src/my-precious-skill/SKILL.md && echo "SKILL.md: SURVIVED"
test -f /tmp/ss-local-src/my-precious-skill/README.md && echo "README.md: SURVIVED"
test -d /tmp/ss-local-src/my-precious-skill && echo "DIR: SURVIVED"
```

Expected:
- exit_code: 0
- SKILL.md: SURVIVED
- README.md: SURVIVED
- DIR: SURVIVED

### 9. API install: local path preserves source after install

```bash
curl -sf -X POST http://127.0.0.1:19421/api/install \
  -H 'Content-Type: application/json' \
  -d '{"source": "/tmp/ss-local-src/my-precious-skill", "force": true}'
```

Expected:
- exit_code: 0
- jq: .skillName == "my-precious-skill"

### 10. Verify: source files survive after /api/install

```bash
test -f /tmp/ss-local-src/my-precious-skill/SKILL.md && echo "SKILL.md: SURVIVED"
test -f /tmp/ss-local-src/my-precious-skill/README.md && echo "README.md: SURVIVED"
cat /tmp/ss-local-src/my-precious-skill/README.md
```

Expected:
- exit_code: 0
- SKILL.md: SURVIVED
- README.md: SURVIVED
- important data

### 11. Cleanup: uninstall before batch test

```bash
ss uninstall my-precious-skill --json -g >/dev/null 2>&1 || true
echo "cleaned"
```

Expected:
- exit_code: 0
- cleaned

### 12. API discover multi-skill: local repo preserves source

```bash
curl -sf -X POST http://127.0.0.1:19421/api/discover \
  -H 'Content-Type: application/json' \
  -d '{"source": "/tmp/ss-local-multi"}'
```

Expected:
- exit_code: 0
- jq: .skills | length == 2

### 13. Verify: multi-skill source files survive after discover

```bash
test -f /tmp/ss-local-multi/alpha/SKILL.md && echo "alpha/SKILL.md: SURVIVED"
test -f /tmp/ss-local-multi/beta/SKILL.md && echo "beta/SKILL.md: SURVIVED"
test -f /tmp/ss-local-multi/alpha/data.txt && echo "alpha/data.txt: SURVIVED"
test -f /tmp/ss-local-multi/beta/data.txt && echo "beta/data.txt: SURVIVED"
```

Expected:
- exit_code: 0
- alpha/SKILL.md: SURVIVED
- beta/SKILL.md: SURVIVED
- alpha/data.txt: SURVIVED
- beta/data.txt: SURVIVED

### 14. API batch install: full UI flow preserves source

```bash
curl -sf -X POST http://127.0.0.1:19421/api/install/batch \
  -H 'Content-Type: application/json' \
  -d '{
    "source": "/tmp/ss-local-multi",
    "skills": [{"name": "alpha", "path": "alpha"}, {"name": "beta", "path": "beta"}],
    "force": true
  }'
```

Expected:
- exit_code: 0
- jq: .results | length == 2
- jq: [.results[].error] | all(. == null or . == "")

### 15. Verify: multi-skill source files survive after batch install

```bash
test -d /tmp/ss-local-multi && echo "DIR: SURVIVED"
test -f /tmp/ss-local-multi/alpha/SKILL.md && echo "alpha/SKILL.md: SURVIVED"
test -f /tmp/ss-local-multi/beta/SKILL.md && echo "beta/SKILL.md: SURVIVED"
test -f /tmp/ss-local-multi/alpha/data.txt && echo "alpha/data.txt: SURVIVED"
test -f /tmp/ss-local-multi/beta/data.txt && echo "beta/data.txt: SURVIVED"
cat /tmp/ss-local-multi/alpha/data.txt
```

Expected:
- exit_code: 0
- DIR: SURVIVED
- alpha/SKILL.md: SURVIVED
- beta/SKILL.md: SURVIVED
- alpha/data.txt: SURVIVED
- beta/data.txt: SURVIVED
- alpha data

### 16. Teardown: stop UI server and clean up

```bash
kill $(lsof -ti:19421) 2>/dev/null || true
ss uninstall alpha --force -g 2>/dev/null || true
ss uninstall beta --force -g 2>/dev/null || true
ss uninstall my-precious-skill --force -g 2>/dev/null || true
rm -rf /tmp/ss-local-src /tmp/ss-local-multi
echo "teardown complete"
```

Expected:
- exit_code: 0
- teardown complete

## Pass Criteria

- All steps exit with code 0
- Source files exist after every discover and install operation
- No source files are deleted at any point during the test
- Both single-skill and multi-skill local repos survive the full UI flow
