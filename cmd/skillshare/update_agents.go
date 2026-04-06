package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"skillshare/internal/check"
	"skillshare/internal/config"
	"skillshare/internal/install"
	"skillshare/internal/oplog"
	"skillshare/internal/ui"
)

// cmdUpdateAgents handles "skillshare update agents [name|--all]".
func cmdUpdateAgents(args []string, cfg *config.Config, start time.Time) error {
	opts, showHelp, parseErr := parseUpdateAgentArgs(args)
	if showHelp {
		printUpdateHelp()
		return nil
	}
	if parseErr != nil {
		return parseErr
	}

	agentsDir := cfg.EffectiveAgentsSource()
	if _, err := os.Stat(agentsDir); err != nil {
		if os.IsNotExist(err) {
			ui.Info("No agents source directory (%s)", agentsDir)
			return nil
		}
		return fmt.Errorf("cannot access agents source: %w", err)
	}

	// Discover agents and check status
	results := check.CheckAgents(agentsDir)
	if len(results) == 0 {
		ui.Info("No agents found")
		return nil
	}

	// Filter by name if specified
	if len(opts.names) > 0 {
		results = filterAgentCheckResults(results, opts.names)
		if len(results) == 0 {
			return fmt.Errorf("no matching agents found: %s", strings.Join(opts.names, ", "))
		}
	}

	// Only check agents that have remote sources
	var tracked []check.AgentCheckResult
	for _, r := range results {
		if r.Source != "" {
			tracked = append(tracked, r)
		}
	}

	if len(tracked) == 0 {
		ui.Info("No tracked agents to update (all are local)")
		return nil
	}

	// Enrich with remote status
	if !opts.jsonOutput {
		sp := ui.StartSpinner(fmt.Sprintf("Checking %d agent(s) for updates...", len(tracked)))
		check.EnrichAgentResultsWithRemote(tracked, func() { sp.Success("Check complete") })
	} else {
		check.EnrichAgentResultsWithRemote(tracked, nil)
	}

	// Find agents with updates available
	var updatable []check.AgentCheckResult
	for _, r := range tracked {
		if r.Status == "update_available" {
			updatable = append(updatable, r)
		}
	}

	if len(updatable) == 0 {
		if !opts.jsonOutput {
			ui.Success("All agents are up to date")
		}
		if opts.jsonOutput {
			return updateAgentsOutputJSON(nil, opts.dryRun, start, nil)
		}
		return nil
	}

	if !opts.jsonOutput {
		ui.Header("Updating agents")
		if opts.dryRun {
			ui.Warning("Dry run mode - no changes will be made")
		}
	}

	// Update each agent by re-installing from its source
	var updated, failed int
	for _, r := range updatable {
		if opts.dryRun {
			if !opts.jsonOutput {
				ui.Info("  %s: update available from %s", r.Name, r.Source)
			}
			continue
		}

		err := reinstallAgent(agentsDir, r)
		if err != nil {
			if !opts.jsonOutput {
				ui.Error("  %s: update failed: %v", r.Name, err)
			}
			failed++
		} else {
			if !opts.jsonOutput {
				ui.Success("  %s: updated", r.Name)
			}
			updated++
		}
	}

	if !opts.jsonOutput && !opts.dryRun {
		fmt.Println()
		ui.Info("Agent update: %d updated, %d failed", updated, failed)
	}

	logUpdateAgentOp(config.ConfigPath(), len(updatable), updated, failed, opts.dryRun, start)

	if opts.jsonOutput {
		return updateAgentsOutputJSON(updatable, opts.dryRun, start, nil)
	}

	if failed > 0 {
		return fmt.Errorf("%d agent(s) failed to update", failed)
	}
	return nil
}

// reinstallAgent re-installs an agent from its recorded source.
func reinstallAgent(agentsDir string, r check.AgentCheckResult) error {
	metaFile := filepath.Join(agentsDir, r.Name+".skillshare-meta.json")

	// Read current metadata
	metaData, err := os.ReadFile(metaFile)
	if err != nil {
		return fmt.Errorf("cannot read metadata: %w", err)
	}
	var meta install.SkillMeta
	if err := json.Unmarshal(metaData, &meta); err != nil {
		return fmt.Errorf("invalid metadata: %w", err)
	}

	if meta.Source == "" {
		return fmt.Errorf("no source in metadata")
	}

	// Parse and re-install from source
	source, parseErr := install.ParseSource(meta.Source)
	if parseErr != nil {
		return fmt.Errorf("invalid source: %w", parseErr)
	}
	if meta.Branch != "" {
		source.Branch = meta.Branch
	}

	installOpts := install.InstallOptions{
		Kind:       "agent",
		AgentNames: []string{r.Name},
		Force:      true,
		Update:     true,
	}

	_, installErr := install.Install(source, agentsDir, installOpts)
	return installErr
}

// updateAgentArgs holds parsed arguments for agent update.
type updateAgentArgs struct {
	names      []string
	all        bool
	dryRun     bool
	jsonOutput bool
}

func parseUpdateAgentArgs(args []string) (*updateAgentArgs, bool, error) {
	opts := &updateAgentArgs{}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--all":
			opts.all = true
		case arg == "--dry-run" || arg == "-n":
			opts.dryRun = true
		case arg == "--json":
			opts.jsonOutput = true
		case arg == "--group" || arg == "-G":
			return nil, false, fmt.Errorf("--group is not supported for agents")
		case arg == "--help" || arg == "-h":
			return nil, true, nil
		case strings.HasPrefix(arg, "-"):
			return nil, false, fmt.Errorf("unknown option: %s", arg)
		default:
			opts.names = append(opts.names, arg)
		}
	}

	if !opts.all && len(opts.names) == 0 {
		return nil, false, fmt.Errorf("specify agent name(s) or --all")
	}
	if opts.all && len(opts.names) > 0 {
		return nil, false, fmt.Errorf("--all cannot be used with agent names")
	}

	return opts, false, nil
}

func filterAgentCheckResults(results []check.AgentCheckResult, names []string) []check.AgentCheckResult {
	nameSet := make(map[string]bool, len(names))
	for _, n := range names {
		nameSet[n] = true
	}
	var filtered []check.AgentCheckResult
	for _, r := range results {
		if nameSet[r.Name] {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

func logUpdateAgentOp(cfgPath string, total, updated, failed int, dryRun bool, start time.Time) {
	status := "ok"
	if failed > 0 && updated > 0 {
		status = "partial"
	} else if failed > 0 {
		status = "error"
	}
	e := oplog.NewEntry("update", status, time.Since(start))
	e.Args = map[string]any{
		"resource_kind":  "agent",
		"agents_total":   total,
		"agents_updated": updated,
		"agents_failed":  failed,
		"dry_run":        dryRun,
	}
	oplog.WriteWithLimit(cfgPath, oplog.OpsFile, e, logMaxEntries()) //nolint:errcheck
}

func updateAgentsOutputJSON(updatable []check.AgentCheckResult, dryRun bool, start time.Time, err error) error {
	type agentUpdateJSON struct {
		Name   string `json:"name"`
		Source string `json:"source,omitempty"`
		Status string `json:"status"`
	}
	var items []agentUpdateJSON
	for _, r := range updatable {
		items = append(items, agentUpdateJSON{
			Name:   r.Name,
			Source: r.Source,
			Status: r.Status,
		})
	}
	output := struct {
		Agents   []agentUpdateJSON `json:"agents"`
		DryRun   bool              `json:"dry_run"`
		Duration string            `json:"duration"`
	}{
		Agents:   items,
		DryRun:   dryRun,
		Duration: formatDuration(start),
	}
	return writeJSONResult(&output, err)
}

// cmdUpdateAgentsProject handles "skillshare update -p agents [name|--all]".
func cmdUpdateAgentsProject(args []string, projectRoot string, start time.Time) error {
	agentsDir := filepath.Join(projectRoot, ".skillshare", "agents")
	if _, err := os.Stat(agentsDir); err != nil {
		if os.IsNotExist(err) {
			ui.Info("No project agents directory (%s)", agentsDir)
			return nil
		}
		return fmt.Errorf("cannot access project agents: %w", err)
	}

	opts, showHelp, parseErr := parseUpdateAgentArgs(args)
	if showHelp {
		printUpdateHelp()
		return nil
	}
	if parseErr != nil {
		return parseErr
	}

	results := check.CheckAgents(agentsDir)
	if len(results) == 0 {
		ui.Info("No project agents found")
		return nil
	}

	if len(opts.names) > 0 {
		results = filterAgentCheckResults(results, opts.names)
		if len(results) == 0 {
			return fmt.Errorf("no matching agents found: %s", strings.Join(opts.names, ", "))
		}
	}

	var tracked []check.AgentCheckResult
	for _, r := range results {
		if r.Source != "" {
			tracked = append(tracked, r)
		}
	}

	if len(tracked) == 0 {
		ui.Info("No tracked project agents to update (all are local)")
		return nil
	}

	sp := ui.StartSpinner(fmt.Sprintf("Checking %d agent(s)...", len(tracked)))
	check.EnrichAgentResultsWithRemote(tracked, func() { sp.Success("Check complete") })

	var updatable []check.AgentCheckResult
	for _, r := range tracked {
		if r.Status == "update_available" {
			updatable = append(updatable, r)
		}
	}

	if len(updatable) == 0 {
		ui.Success("All project agents are up to date")
		return nil
	}

	ui.Header("Updating project agents")
	if opts.dryRun {
		ui.Warning("Dry run mode")
		for _, r := range updatable {
			ui.Info("  %s: update available from %s", r.Name, r.Source)
		}
		return nil
	}

	var updated, failed int
	for _, r := range updatable {
		if err := reinstallAgent(agentsDir, r); err != nil {
			ui.Error("  %s: %v", r.Name, err)
			failed++
		} else {
			ui.Success("  %s: updated", r.Name)
			updated++
		}
	}

	logUpdateAgentOp(config.ProjectConfigPath(projectRoot), len(updatable), updated, failed, opts.dryRun, start)

	if failed > 0 {
		return fmt.Errorf("%d agent(s) failed to update", failed)
	}
	return nil
}
