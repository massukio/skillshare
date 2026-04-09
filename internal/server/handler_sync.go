package server

import (
	"encoding/json"
	"maps"
	"net/http"
	"time"

	"skillshare/internal/config"
	"skillshare/internal/resource"
	"skillshare/internal/skillignore"
	ssync "skillshare/internal/sync"
)

// ignorePayload builds the common ignored-skills fields for JSON responses.
func ignorePayload(stats *skillignore.IgnoreStats) map[string]any {
	skills := []string{}
	rootFile := ""
	repoFiles := []string{}
	if stats != nil {
		if len(stats.IgnoredSkills) > 0 {
			skills = stats.IgnoredSkills
		}
		rootFile = stats.RootFile
		if stats.RepoFiles != nil {
			repoFiles = stats.RepoFiles
		}
	}
	return map[string]any{
		"ignored_count":  len(skills),
		"ignored_skills": skills,
		"ignore_root":    rootFile,
		"ignore_repos":   repoFiles,
	}
}

type syncTargetResult struct {
	Target     string   `json:"target"`
	Linked     []string `json:"linked"`
	Updated    []string `json:"updated"`
	Skipped    []string `json:"skipped"`
	Pruned     []string `json:"pruned"`
	DirCreated string   `json:"dir_created,omitempty"`
}

func (s *Server) handleSync(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()

	var body struct {
		DryRun bool   `json:"dryRun"`
		Force  bool   `json:"force"`
		Kind   string `json:"kind"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		// Default to non-dry-run, non-force, empty kind (both)
	}

	if body.Kind != "" && body.Kind != "skill" && body.Kind != "agent" {
		writeError(w, http.StatusBadRequest, "invalid kind: must be 'skill', 'agent', or empty")
		return
	}

	globalMode := s.cfg.Mode
	if globalMode == "" {
		globalMode = "merge"
	}

	// Pre-check warnings via shared config validation
	warnings, validErr := config.ValidateConfig(s.cfg)
	if validErr != nil {
		writeError(w, http.StatusBadRequest, validErr.Error())
		return
	}

	results := make([]syncTargetResult, 0)

	var ignoreStats *skillignore.IgnoreStats

	// Skill sync (skip when kind == "agent")
	if body.Kind != "agent" {
		var allSkills []ssync.DiscoveredSkill
		var err error
		allSkills, ignoreStats, err = ssync.DiscoverSourceSkillsWithStats(s.cfg.Source)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to discover skills: "+err.Error())
			return
		}

		if len(allSkills) == 0 {
			warnings = append(warnings, "source directory is empty (0 skills)")
		}

		// Registry entries are managed by install/uninstall, not sync.
		// Sync only manages symlinks — it must not prune registry entries
		// for installed skills whose files may be missing from disk.

		for name, target := range s.cfg.Targets {
			sc := target.SkillsConfig()
			mode := sc.Mode
			if mode == "" {
				mode = globalMode
			}

			res := syncTargetResult{
				Target:  name,
				Linked:  make([]string, 0),
				Updated: make([]string, 0),
				Skipped: make([]string, 0),
				Pruned:  make([]string, 0),
			}

			syncErrArgs := map[string]any{
				"targets_total":  len(s.cfg.Targets),
				"targets_failed": 1,
				"target":         name,
				"dry_run":        body.DryRun,
				"force":          body.Force,
				"scope":          "ui",
			}

			switch mode {
			case "merge":
				mergeResult, err := ssync.SyncTargetMergeWithSkills(name, target, allSkills, s.cfg.Source, body.DryRun, body.Force, s.projectRoot)
				if err != nil {
					s.writeOpsLog("sync", "error", start, syncErrArgs, err.Error())
					writeError(w, http.StatusInternalServerError, "sync failed for "+name+": "+err.Error())
					return
				}
				res.Linked = mergeResult.Linked
				res.Updated = mergeResult.Updated
				res.Skipped = mergeResult.Skipped
				res.DirCreated = mergeResult.DirCreated

				pruneResult, err := ssync.PruneOrphanLinksWithSkills(ssync.PruneOptions{
					TargetPath: sc.Path, SourcePath: s.cfg.Source, Skills: allSkills,
					Include: sc.Include, Exclude: sc.Exclude, TargetNaming: sc.TargetNaming, TargetName: name,
					DryRun: body.DryRun, Force: body.Force,
				})
				if err == nil {
					res.Pruned = pruneResult.Removed
				}

			case "copy":
				copyResult, err := ssync.SyncTargetCopyWithSkills(name, target, allSkills, s.cfg.Source, body.DryRun, body.Force, nil)
				if err != nil {
					s.writeOpsLog("sync", "error", start, syncErrArgs, err.Error())
					writeError(w, http.StatusInternalServerError, "sync failed for "+name+": "+err.Error())
					return
				}
				res.Linked = copyResult.Copied
				res.Updated = copyResult.Updated
				res.Skipped = copyResult.Skipped
				res.DirCreated = copyResult.DirCreated

				pruneResult, err := ssync.PruneOrphanCopiesWithSkills(sc.Path, allSkills, sc.Include, sc.Exclude, name, sc.TargetNaming, body.DryRun)
				if err == nil {
					res.Pruned = pruneResult.Removed
				}

			default:
				err := ssync.SyncTarget(name, target, s.cfg.Source, body.DryRun, s.projectRoot)
				if err != nil {
					s.writeOpsLog("sync", "error", start, syncErrArgs, err.Error())
					writeError(w, http.StatusInternalServerError, "sync failed for "+name+": "+err.Error())
					return
				}
				res.Linked = []string{"(symlink mode)"}
			}

			results = append(results, res)
		}
	}

	// Agent sync (skip when kind == "skill")
	if body.Kind != "skill" {
		agentsSource := s.agentsSource()
		if agentsSource != "" {
			agentDiscovered, _ := resource.AgentKind{}.Discover(agentsSource)
			agents := resource.ActiveAgents(agentDiscovered)

			if len(agents) > 0 {
				var builtinAgents map[string]config.TargetConfig
				if s.IsProjectMode() {
					builtinAgents = config.ProjectAgentTargets()
				} else {
					builtinAgents = config.DefaultAgentTargets()
				}

				for name, target := range s.cfg.Targets {
					ac := target.AgentsConfig()
					agentPath := ac.Path
					if agentPath == "" {
						if builtin, ok := builtinAgents[name]; ok {
							agentPath = builtin.Path
						}
					}
					if agentPath == "" {
						continue
					}
					agentPath = config.ExpandPath(agentPath)

					agentMode := ac.Mode
					if agentMode == "" {
						agentMode = "merge"
					}

					agentResult, err := ssync.SyncAgents(agents, agentsSource, agentPath, agentMode, body.DryRun, body.Force)
					if err != nil {
						warnings = append(warnings, "agent sync failed for "+name+": "+err.Error())
						continue
					}

					// Merge into existing result or create new
					merged := false
					for i := range results {
						if results[i].Target == name {
							results[i].Linked = append(results[i].Linked, agentResult.Linked...)
							results[i].Updated = append(results[i].Updated, agentResult.Updated...)
							results[i].Skipped = append(results[i].Skipped, agentResult.Skipped...)
							merged = true
							break
						}
					}
					if !merged && (len(agentResult.Linked) > 0 || len(agentResult.Updated) > 0 || len(agentResult.Skipped) > 0) {
						results = append(results, syncTargetResult{
							Target:  name,
							Linked:  agentResult.Linked,
							Updated: agentResult.Updated,
							Skipped: agentResult.Skipped,
							Pruned:  make([]string, 0),
						})
					}

					// Prune orphan agents
					if agentMode == "merge" {
						pruned, _ := ssync.PruneOrphanAgentLinks(agentPath, agents, body.DryRun)
						for i := range results {
							if results[i].Target == name {
								results[i].Pruned = append(results[i].Pruned, pruned...)
								break
							}
						}
					} else if agentMode == "copy" {
						pruned, _ := ssync.PruneOrphanAgentCopies(agentPath, agents, body.DryRun)
						for i := range results {
							if results[i].Target == name {
								results[i].Pruned = append(results[i].Pruned, pruned...)
								break
							}
						}
					}
				}
			}
		}
	}

	// Log the sync operation
	s.writeOpsLog("sync", "ok", start, map[string]any{
		"targets_total":  len(results),
		"targets_failed": 0,
		"dry_run":        body.DryRun,
		"force":          body.Force,
		"kind":           body.Kind,
		"scope":          "ui",
	}, "")

	resp := map[string]any{
		"results":  results,
		"warnings": warnings,
	}
	maps.Copy(resp, ignorePayload(ignoreStats))
	writeJSON(w, resp)
}

type diffItem struct {
	Skill  string `json:"skill"`
	Action string `json:"action"` // "link", "update", "skip", "prune", "local"
	Reason string `json:"reason"` // human-readable description
	Kind   string `json:"kind,omitempty"`
}

type diffTarget struct {
	Target         string     `json:"target"`
	Items          []diffItem `json:"items"`
	SkippedCount   int        `json:"skippedCount,omitempty"`
	CollisionCount int        `json:"collisionCount,omitempty"`
}

func (s *Server) handleDiff(w http.ResponseWriter, r *http.Request) {
	// Snapshot config under RLock, then release before slow I/O.
	s.mu.RLock()
	source := s.cfg.Source
	agentsSource := s.agentsSource()
	globalMode := s.cfg.Mode
	targets := s.cloneTargets()
	s.mu.RUnlock()

	if globalMode == "" {
		globalMode = "merge"
	}

	filterTarget := r.URL.Query().Get("target")

	discovered, ignoreStats, err := ssync.DiscoverSourceSkillsWithStats(source)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	diffs := make([]diffTarget, 0)
	for name, target := range targets {
		if filterTarget != "" && filterTarget != name {
			continue
		}
		diffs = append(diffs, s.computeTargetDiff(name, target, discovered, globalMode, source))
	}

	// Agent diffs — discover agents and compute per-target diffs
	var agents []resource.DiscoveredResource
	if agentsSource != "" {
		discovered, _ := resource.AgentKind{}.Discover(agentsSource)
		agents = resource.ActiveAgents(discovered)
	}

	if len(agents) > 0 {
		var builtinAgents map[string]config.TargetConfig
		if s.IsProjectMode() {
			builtinAgents = config.ProjectAgentTargets()
		} else {
			builtinAgents = config.DefaultAgentTargets()
		}

		for name, target := range targets {
			if filterTarget != "" && filterTarget != name {
				continue
			}

			ac := target.AgentsConfig()
			agentPath := ac.Path
			if agentPath == "" {
				if builtin, ok := builtinAgents[name]; ok {
					agentPath = builtin.Path
				}
			}
			if agentPath == "" {
				continue
			}
			agentPath = config.ExpandPath(agentPath)

			agentItems := computeAgentTargetDiff(agentPath, agents)
			if len(agentItems) == 0 {
				continue
			}

			// Merge into existing diff for this target
			merged := false
			for i := range diffs {
				if diffs[i].Target == name {
					diffs[i].Items = append(diffs[i].Items, agentItems...)
					merged = true
					break
				}
			}
			if !merged {
				diffs = append(diffs, diffTarget{
					Target: name,
					Items:  agentItems,
				})
			}
		}
	}

	resp := map[string]any{"diffs": diffs}
	maps.Copy(resp, ignorePayload(ignoreStats))
	writeJSON(w, resp)
}
