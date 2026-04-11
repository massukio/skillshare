# Issue #124: Orchestrator Repo Install

Verify that repos with root + child skills install children as independent skills, root appears in selection, and blob URLs resolve correctly.

Repo structure under test:

```
SKILL.md              (root: name derived from repo dir)
src/main.go
skills/
  writer/SKILL.md     (child: "writer")
  reader/SKILL.md     (child: "reader")
```

## Steps

### Step 1: Create local orchestrator repo

```bash
rm -rf /tmp/orchestrator-repo
mkdir -p /tmp/orchestrator-repo/skills/writer /tmp/orchestrator-repo/skills/reader /tmp/orchestrator-repo/src

git config --global user.email "test@test.com" 2>/dev/null
git config --global user.name "Test" 2>/dev/null

cat > /tmp/orchestrator-repo/SKILL.md << 'SKILL'
---
name: office-cli
description: Office CLI orchestrator
---
# Office CLI
Root skill for the orchestrator pack.
SKILL

cat > /tmp/orchestrator-repo/src/main.go << 'GO'
package main
func main() {}
GO

cat > /tmp/orchestrator-repo/skills/writer/SKILL.md << 'SKILL'
---
name: writer
description: Writer skill
---
# Writer
A child skill for writing.
SKILL

echo "# Writer docs" > /tmp/orchestrator-repo/skills/writer/README.md

cat > /tmp/orchestrator-repo/skills/reader/SKILL.md << 'SKILL'
---
name: reader
description: Reader skill
---
# Reader
A child skill for reading.
SKILL

cd /tmp/orchestrator-repo && git init -q && git add -A && git commit -m "init" -q
echo "REPO_READY"
```

Expected:
- exit_code: 0
- REPO_READY

### Step 2: Install all skills with --json

Install the orchestrator repo using `--json --all` to skip prompts. Verify 3 independent skills are installed. Root name = `orchestrator-repo` (from directory).

```bash
/workspace/bin/skillshare install file:///tmp/orchestrator-repo --json --all -g 2>/dev/null
```

Expected:
- exit_code: 0
- jq: .skills | length == 3
- jq: .skills | sort | . == ["orchestrator-repo","reader","writer"]
- jq: .failed | length == 0

### Step 3: Root dir does NOT contain child skill dirs

Root copy should exclude `skills/writer/` and `skills/reader/` but keep `src/`.

```bash
ROOT_DIR="$HOME/.config/skillshare/skills/orchestrator-repo"
echo "ROOT_EXISTS=$(test -d "$ROOT_DIR" && echo yes || echo no)"
echo "ROOT_SKILL_MD=$(test -f "$ROOT_DIR/SKILL.md" && echo yes || echo no)"
echo "ROOT_SRC=$(test -d "$ROOT_DIR/src" && echo yes || echo no)"
echo "NO_CHILD_WRITER=$(test -d "$ROOT_DIR/skills/writer" && echo LEAKED || echo clean)"
echo "NO_CHILD_READER=$(test -d "$ROOT_DIR/skills/reader" && echo LEAKED || echo clean)"
```

Expected:
- exit_code: 0
- ROOT_EXISTS=yes
- ROOT_SKILL_MD=yes
- ROOT_SRC=yes
- NO_CHILD_WRITER=clean
- NO_CHILD_READER=clean

### Step 4: Children exist as siblings under parent

Children at `orchestrator-repo/writer/` and `orchestrator-repo/reader/`, each with own SKILL.md.

```bash
SKILLS_DIR="$HOME/.config/skillshare/skills"
echo "WRITER_EXISTS=$(test -f "$SKILLS_DIR/orchestrator-repo/writer/SKILL.md" && echo yes || echo no)"
echo "READER_EXISTS=$(test -f "$SKILLS_DIR/orchestrator-repo/reader/SKILL.md" && echo yes || echo no)"
echo "WRITER_README=$(test -f "$SKILLS_DIR/orchestrator-repo/writer/README.md" && echo yes || echo no)"
```

Expected:
- exit_code: 0
- WRITER_EXISTS=yes
- READER_EXISTS=yes
- WRITER_README=yes

### Step 5: All 3 skills appear independently in list

```bash
/workspace/bin/skillshare list --json -g 2>/dev/null
```

Expected:
- exit_code: 0
- jq: [.[] | .name] | sort | . == ["orchestrator-repo","orchestrator-repo__reader","orchestrator-repo__writer"]

### Step 6: Install root skill only via --skill filter

Root skill is individually selectable (issue #124 fix: root included in selection).

```bash
/workspace/bin/skillshare uninstall --all -g --force >/dev/null 2>&1
/workspace/bin/skillshare install file:///tmp/orchestrator-repo --json --skill orchestrator-repo -g 2>/dev/null
```

Expected:
- exit_code: 0
- jq: .skills == ["orchestrator-repo"]

### Step 7: Root-only install has no child content

```bash
SKILLS_DIR="$HOME/.config/skillshare/skills"
echo "ROOT_EXISTS=$(test -f "$SKILLS_DIR/orchestrator-repo/SKILL.md" && echo yes || echo no)"
echo "NO_WRITER=$(test -d "$SKILLS_DIR/orchestrator-repo/writer" && echo LEAKED || echo clean)"
echo "NO_READER=$(test -d "$SKILLS_DIR/orchestrator-repo/reader" && echo LEAKED || echo clean)"
```

Expected:
- exit_code: 0
- ROOT_EXISTS=yes
- NO_WRITER=clean
- NO_READER=clean

### Step 8: Install child-only without root

```bash
/workspace/bin/skillshare uninstall --all -g --force >/dev/null 2>&1
/workspace/bin/skillshare install file:///tmp/orchestrator-repo --json --skill writer -g 2>/dev/null
```

Expected:
- exit_code: 0
- jq: .skills == ["writer"]

### Step 9: Blob URL parsing — trailing SKILL.md stripped (unit tests)

The blob URL fix resolves `github.com/.../blob/main/skills/foo/SKILL.md` to `subdir="skills/foo"`. Run the specific subtests.

```bash
cd /workspace && go test ./internal/install -run "TestParseSource_GitHubShorthand/github_blob_URL" -count=1 -v 2>&1 | tail -10
```

Expected:
- exit_code: 0
- PASS
- regex: blob_URL

## Pass Criteria

- Steps 2-5: Orchestrator install creates 3 independent skills; root excludes child dirs; children are siblings
- Steps 6-7: Root skill is individually selectable and installs cleanly without child leakage
- Step 8: Child skill installs independently without root
- Step 9: Blob URL unit tests pass, confirming trailing SKILL.md is properly resolved
