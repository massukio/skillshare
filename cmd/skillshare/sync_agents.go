package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"skillshare/internal/config"
	"skillshare/internal/resource"
	"skillshare/internal/sync"
	"skillshare/internal/ui"
)

// agentSyncStats aggregates per-target agent sync results.
type agentSyncStats struct {
	linked, skipped, updated, pruned int
}

// syncAgentsGlobal discovers agents and syncs them to all agent-capable targets.
// Returns total stats and any error.
func syncAgentsGlobal(cfg *config.Config, dryRun, force, jsonOutput bool, start time.Time) (agentSyncStats, error) {
	agentsSource := cfg.EffectiveAgentsSource()

	// Check agent source exists
	if _, err := os.Stat(agentsSource); err != nil {
		if os.IsNotExist(err) {
			if !jsonOutput {
				ui.Info("No agents source directory (%s)", agentsSource)
			}
			return agentSyncStats{}, nil
		}
		return agentSyncStats{}, fmt.Errorf("cannot access agents source: %w", err)
	}

	// Discover agents
	agents, err := resource.AgentKind{}.Discover(agentsSource)
	if err != nil {
		return agentSyncStats{}, fmt.Errorf("cannot discover agents: %w", err)
	}

	if len(agents) == 0 {
		if !jsonOutput {
			ui.Info("No agents found in %s", agentsSource)
		}
		return agentSyncStats{}, nil
	}

	if !jsonOutput {
		ui.Header("Syncing agents")
		if dryRun {
			ui.Warning("Dry run mode - no changes will be made")
		}
	}

	// Resolve agent-capable targets: user config agents sub-key + built-in defaults
	builtinAgents := config.DefaultAgentTargets()
	var totals agentSyncStats
	var syncErr error

	for name := range cfg.Targets {
		agentPath := resolveAgentTargetPath(cfg.Targets[name], builtinAgents, name)
		if agentPath == "" {
			continue // target has no agent path
		}

		tc := cfg.Targets[name]
		ac := tc.AgentsConfig()
		mode := ac.Mode
		if mode == "" {
			mode = "merge" // default agent sync mode
		}

		result, err := sync.SyncAgents(agents, agentsSource, agentPath, mode, dryRun, force)
		if err != nil {
			if !jsonOutput {
				ui.Error("%s: agent sync failed: %v", name, err)
			}
			syncErr = fmt.Errorf("some agent targets failed to sync")
			continue
		}

		// Prune orphan agent links/copies
		var pruned []string
		switch mode {
		case "copy":
			pruned, _ = sync.PruneOrphanAgentCopies(agentPath, agents, dryRun)
		case "merge":
			pruned, _ = sync.PruneOrphanAgentLinks(agentPath, agents, dryRun)
		}

		stats := agentSyncStats{
			linked:  len(result.Linked),
			skipped: len(result.Skipped),
			updated: len(result.Updated),
			pruned:  len(pruned),
		}
		totals.linked += stats.linked
		totals.skipped += stats.skipped
		totals.updated += stats.updated
		totals.pruned += stats.pruned

		if !jsonOutput {
			reportAgentSyncResult(name, mode, stats, dryRun)
		}
	}

	if !jsonOutput {
		fmt.Println()
		ui.Info("Agent sync: %d linked, %d local, %d updated, %d pruned (%s)",
			totals.linked, totals.skipped, totals.updated, totals.pruned,
			formatDuration(start))
	}

	return totals, syncErr
}

// resolveAgentTargetPath returns the effective agent path for a target,
// checking user config first, then built-in defaults. Returns "" if none.
func resolveAgentTargetPath(tc config.TargetConfig, builtinAgents map[string]config.TargetConfig, name string) string {
	if ac := tc.AgentsConfig(); ac.Path != "" {
		return config.ExpandPath(ac.Path)
	}
	if builtin, ok := builtinAgents[name]; ok {
		return config.ExpandPath(builtin.Path)
	}
	return ""
}

// syncAgentsProject syncs agents for project mode using .skillshare/agents/ as source
// and project-level target agent paths.
func syncAgentsProject(projectRoot string, dryRun, force, jsonOutput bool, start time.Time) error {
	agentsSource := filepath.Join(projectRoot, ".skillshare", "agents")

	if _, err := os.Stat(agentsSource); err != nil {
		if os.IsNotExist(err) {
			if !jsonOutput {
				ui.Info("No project agents directory (%s)", agentsSource)
			}
			return nil
		}
		return fmt.Errorf("cannot access project agents: %w", err)
	}

	agents, err := resource.AgentKind{}.Discover(agentsSource)
	if err != nil {
		return fmt.Errorf("cannot discover project agents: %w", err)
	}

	if len(agents) == 0 {
		if !jsonOutput {
			ui.Info("No project agents found")
		}
		return nil
	}

	if !jsonOutput {
		ui.Header("Syncing project agents")
		if dryRun {
			ui.Warning("Dry run mode - no changes will be made")
		}
	}

	builtinAgents := config.ProjectAgentTargets()
	var totals agentSyncStats
	var syncErr error

	// Load project config for target list
	projCfg, loadErr := config.LoadProject(projectRoot)
	if loadErr != nil {
		return fmt.Errorf("cannot load project config: %w", loadErr)
	}

	for _, entry := range projCfg.Targets {
		var agentPath string
		ac := entry.AgentsConfig()
		if ac.Path != "" {
			agentPath = ac.Path
			if !filepath.IsAbs(agentPath) {
				agentPath = filepath.Join(projectRoot, agentPath)
			}
		} else if builtin, ok := builtinAgents[entry.Name]; ok {
			agentPath = config.ExpandPath(builtin.Path)
		}
		if agentPath == "" {
			continue
		}

		mode := ac.Mode
		if mode == "" {
			mode = "merge"
		}

		result, syncResultErr := sync.SyncAgents(agents, agentsSource, agentPath, mode, dryRun, force)
		if syncResultErr != nil {
			if !jsonOutput {
				ui.Error("%s: agent sync failed: %v", entry.Name, syncResultErr)
			}
			syncErr = fmt.Errorf("some agent targets failed to sync")
			continue
		}

		var pruned []string
		switch mode {
		case "copy":
			pruned, _ = sync.PruneOrphanAgentCopies(agentPath, agents, dryRun)
		case "merge":
			pruned, _ = sync.PruneOrphanAgentLinks(agentPath, agents, dryRun)
		}

		stats := agentSyncStats{
			linked:  len(result.Linked),
			skipped: len(result.Skipped),
			updated: len(result.Updated),
			pruned:  len(pruned),
		}
		totals.linked += stats.linked
		totals.skipped += stats.skipped
		totals.updated += stats.updated
		totals.pruned += stats.pruned

		if !jsonOutput {
			reportAgentSyncResult(entry.Name, mode, stats, dryRun)
		}
	}

	if !jsonOutput {
		fmt.Println()
		ui.Info("Project agent sync: %d linked, %d local, %d updated, %d pruned (%s)",
			totals.linked, totals.skipped, totals.updated, totals.pruned,
			formatDuration(start))
	}

	return syncErr
}

// reportAgentSyncResult prints per-target agent sync status.
func reportAgentSyncResult(name, mode string, stats agentSyncStats, dryRun bool) {
	if stats.linked > 0 || stats.updated > 0 || stats.pruned > 0 {
		ui.Success("%s: agents %s (%d linked, %d local, %d updated, %d pruned)",
			name, mode, stats.linked, stats.skipped, stats.updated, stats.pruned)
	} else if stats.skipped > 0 {
		ui.Success("%s: agents %s (%d local preserved)", name, mode, stats.skipped)
	} else {
		ui.Success("%s: agents %s (up to date)", name, mode)
	}
}
