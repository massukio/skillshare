---
sidebar_position: 8
---

# Declarative Skill Manifest

Define your skill collection as code — install, share, and reproduce setups from a single manifest file.

:::tip When does this matter?
Use the declarative manifest when you want reproducible skill setups across machines, team onboarding with a single command, or open-source project bootstrap.
:::

## What Is a Skill Manifest?

The `skills:` section in `.metadata.json` serves as a **portable declaration** of your skill collection. Instead of manually installing skills one by one, you list them in the metadata and run `skillshare install` to bring everything up.

```json
{
  "skills": [
    {
      "name": "react-best-practices",
      "source": "anthropics/skills/skills/react-best-practices",
      "group": "frontend"
    },
    {
      "name": "_team-skills",
      "source": "my-org/shared-skills",
      "tracked": true,
      "group": "devops"
    },
    {
      "name": "commit",
      "source": "anthropics/skills/skills/commit"
    }
  ]
}
```

:::note Migration notes
- **From config.yaml**: In older versions, `skills:` lived inside `config.yaml`. Skillshare automatically migrates it to `.metadata.json` on first load.
- **From registry.yaml**: In v0.19+, skill records moved from `registry.yaml` to `.metadata.json` as the unified metadata store. The migration is automatic — no manual action required.
:::

## How It Works

### Install from Manifest

Running `skillshare install` with **no arguments** reads the manifest and installs all listed skills:

```bash
# Global mode — installs all skills from ~/.config/skillshare/skills/.metadata.json
skillshare install

# Project mode — installs from .skillshare/.metadata.json
skillshare install -p

# Preview without installing
skillshare install --dry-run
```

Skills that already exist are skipped automatically.

### Automatic Reconciliation

The manifest stays in sync with your actual skill collection:

- **`skillshare install <source>`** — adds the installed skill to `.metadata.json` automatically
- **`skillshare uninstall <name>...`** — removes the entry from `.metadata.json` automatically

You never need to edit the manifest manually (though you can).

## Skill Entry Fields

Each entry in the `skills:` list in `.metadata.json` has these fields:

| Field | Required | Description |
|-------|----------|-------------|
| `name` | Yes | Skill name (directory name in source) |
| `source` | Yes | Install source (GitHub shorthand, HTTPS URL, SSH URL) |
| `tracked` | No | `true` for tracked repositories (preserves `.git`) |
| `group` | No | Subdirectory path (e.g. `frontend` or `frontend/vue`). Corresponds to `--into` during install. |

## Use Cases

### Personal Setup

Maintain your personal skill collection across machines:

```bash
# On machine A — skills are already installed and tracked in registry
skillshare push   # backup config + registry to git

# On machine B — fresh machine
skillshare pull   # restore config + registry from git
skillshare install  # install all skills from manifest
skillshare sync   # distribute to all targets
```

### Team Onboarding

New team members get the same AI context in one command:

```bash
# .skillshare/.metadata.json is committed to the repo
git clone <project-repo>
cd <project-repo>
skillshare install -p   # installs all declared skills
skillshare sync -p      # links to project targets
```

### Open-Source Bootstrap

Project maintainers declare recommended skills:

```yaml
# .skillshare/.metadata.json
{
  "skills":
  - name: react-best-practices
    source: anthropics/skills/skills/react-best-practices
    group: frontend
  - name: commit
    source: anthropics/skills/skills/commit
  - name: project-conventions
    source: ./skills/project-conventions
```

:::info Group field and `--into`
When you install with `--into`, the group is recorded automatically:

```bash
skillshare install anthropics/skills/skills/pdf --into frontend
# .metadata.json will contain: name: pdf, group: frontend
```

Running `skillshare install` (no args) recreates the same directory structure from the manifest.
:::

Contributors clone and run `skillshare install -p` to get project-specific AI context immediately.

## Workflow Summary

```
1. Install skills normally      →  .metadata.json auto-updates
2. Push/pull config via git     →  portable across machines
3. Run `skillshare install`     →  reproduce on new machine
4. Run `skillshare sync`        →  distribute to all targets
```

## Extras Configuration

In addition to skills, `config.yaml` can declare **extras** — non-skill resources (rules, commands, prompts) synced to separate directories. Extras are configured in the `extras:` section of `config.yaml` (both global and project):

```yaml
extras:
  - name: rules
    targets:
      - path: ~/.claude/rules
      - path: ~/.cursor/rules
        mode: copy
```

See [sync extras](/docs/reference/commands/sync#sync-extras) for details.

## Related

- [Install command](/docs/reference/commands/install) — `skillshare install` with and without arguments
- [Push/Pull](/docs/reference/commands/push) — backup and restore config via git
- [Project Skills](./project-skills.md) — project-level manifests
