package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"skillshare/internal/install"
)

// ReconcileProjectSkills scans the project source directory recursively for
// remotely-installed skills (those with install metadata or tracked repos)
// and ensures they are present in the MetadataStore.
// It also updates .skillshare/.gitignore for each tracked skill.
func ReconcileProjectSkills(projectRoot string, projectCfg *ProjectConfig, store *install.MetadataStore, sourcePath string) error {
	if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
		return nil
	}

	var gitignoreEntries []string
	onFound := func(fullPath string) {
		gitignoreEntries = append(gitignoreEntries, filepath.Join("skills", fullPath))
	}

	result, err := reconcileSkillsWalk(sourcePath, store, onFound)
	if err != nil {
		return fmt.Errorf("failed to scan project skills: %w", err)
	}

	if pruneStaleEntries(store, result.live) {
		result.changed = true
	}

	if len(gitignoreEntries) > 0 {
		if err := install.UpdateGitIgnoreBatch(filepath.Join(projectRoot, ".skillshare"), gitignoreEntries); err != nil {
			return fmt.Errorf("failed to update .skillshare/.gitignore: %w", err)
		}
	}

	if result.changed {
		if err := store.Save(sourcePath); err != nil {
			return err
		}
	}

	return nil
}

// ReconcileProjectAgents scans the project agents source directory for
// installed agents and ensures they are present in the MetadataStore.
// Also updates .skillshare/.gitignore for each agent.
func ReconcileProjectAgents(projectRoot string, store *install.MetadataStore, agentsSourcePath string) error {
	if _, err := os.Stat(agentsSourcePath); os.IsNotExist(err) {
		return nil
	}

	entries, err := os.ReadDir(agentsSourcePath)
	if err != nil {
		return nil
	}

	changed := false
	var gitignoreEntries []string

	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(name), ".md") {
			continue
		}

		agentName := strings.TrimSuffix(name, ".md")

		existing := store.Get(agentName)
		if existing == nil || existing.Source == "" {
			continue
		}

		if existing.Kind != "agent" {
			existing.Kind = "agent"
			changed = true
		}

		gitignoreEntries = append(gitignoreEntries, filepath.Join("agents", name))
	}

	if len(gitignoreEntries) > 0 {
		if err := install.UpdateGitIgnoreBatch(filepath.Join(projectRoot, ".skillshare"), gitignoreEntries); err != nil {
			return fmt.Errorf("failed to update .skillshare/.gitignore for agents: %w", err)
		}
	}

	if changed {
		if err := store.Save(agentsSourcePath); err != nil {
			return err
		}
	}

	return nil
}
