package server

import (
	"maps"
	"net/http"
	"os"
	"path/filepath"

	"skillshare/internal/config"
	"skillshare/internal/resource"
	ssync "skillshare/internal/sync"
	"skillshare/internal/utils"
)

// handleDiffStream serves an SSE endpoint that streams diff computation progress.
// Events:
//   - "discovering" → {"phase":"..."}                immediately on connect
//   - "start"       → {"total": N}                   after discovery (N = target count)
//   - "result"      → diffTarget                     per-target diff result
//   - "done"        → {"diffs":[...]}                final payload (same shape as GET /api/diff)
func (s *Server) handleDiffStream(w http.ResponseWriter, r *http.Request) {
	safeSend, ok := initSSE(w)
	if !ok {
		return
	}

	ctx := r.Context()

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

	safeSend("discovering", map[string]string{"phase": "scanning source directory"})

	discovered, ignoreStats, err := ssync.DiscoverSourceSkillsWithStats(source)
	if err != nil {
		safeSend("error", map[string]string{"error": err.Error()})
		return
	}

	safeSend("start", map[string]int{"total": len(targets)})

	var diffs []diffTarget
	checked := 0

	for name, target := range targets {
		select {
		case <-ctx.Done():
			return
		default:
		}

		dt := s.computeTargetDiff(name, target, discovered, globalMode, source)
		diffs = append(diffs, dt)
		checked++

		safeSend("result", map[string]any{
			"diff":    dt,
			"checked": checked,
		})
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
			select {
			case <-ctx.Done():
				return
			default:
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

	donePayload := map[string]any{"diffs": diffs}
	maps.Copy(donePayload, ignorePayload(ignoreStats))
	safeSend("done", donePayload)
}

// computeTargetDiff computes the diff for a single target.
// Extracted from handleDiff to share logic with the stream handler.
func (s *Server) computeTargetDiff(name string, target config.TargetConfig, discovered []ssync.DiscoveredSkill, globalMode, source string) diffTarget {
	sc := target.SkillsConfig()
	mode := sc.Mode
	if mode == "" {
		mode = globalMode
	}

	dt := diffTarget{Target: name, Items: make([]diffItem, 0)}

	if mode == "symlink" {
		status := ssync.CheckStatus(sc.Path, source)
		if status != ssync.StatusLinked {
			dt.Items = append(dt.Items, diffItem{Skill: "(entire directory)", Action: "link", Reason: "source only", Kind: "skill"})
		}
		return dt
	}

	filtered, err := ssync.FilterSkills(discovered, sc.Include, sc.Exclude)
	if err != nil {
		return dt
	}
	filtered = ssync.FilterSkillsByTarget(filtered, name)
	resolution, err := ssync.ResolveTargetSkillsForTarget(name, config.ResourceTargetConfig{
		Path:         sc.Path,
		TargetNaming: sc.TargetNaming,
	}, filtered)
	if err != nil {
		dt.Items = append(dt.Items, diffItem{Skill: "(target naming)", Action: "skip", Reason: err.Error(), Kind: "skill"})
		return dt
	}
	// Surface collision/validation stats so the UI can show why skills were skipped
	dt.CollisionCount = len(resolution.Collisions)
	dt.SkippedCount = len(filtered) - len(resolution.Skills)
	validNames := resolution.ValidTargetNames()
	legacyNames := resolution.LegacyFlatNames()

	if mode == "copy" {
		manifest, _ := ssync.ReadManifest(sc.Path)
		for _, resolved := range resolution.Skills {
			skill := resolved.Skill
			oldChecksum, isManaged := manifest.Managed[resolved.TargetName]
			targetSkillPath := filepath.Join(sc.Path, resolved.TargetName)
			if !isManaged {
				if info, statErr := os.Stat(targetSkillPath); statErr == nil {
					if info.IsDir() {
						dt.Items = append(dt.Items, diffItem{Skill: resolved.TargetName, Action: "skip", Reason: "local copy (sync --force to replace)", Kind: "skill"})
					} else {
						dt.Items = append(dt.Items, diffItem{Skill: resolved.TargetName, Action: "update", Reason: "target entry is not a directory", Kind: "skill"})
					}
				} else if os.IsNotExist(statErr) {
					dt.Items = append(dt.Items, diffItem{Skill: resolved.TargetName, Action: "link", Reason: "source only", Kind: "skill"})
				} else {
					dt.Items = append(dt.Items, diffItem{Skill: resolved.TargetName, Action: "update", Reason: "cannot access target entry", Kind: "skill"})
				}
			} else {
				targetInfo, statErr := os.Stat(targetSkillPath)
				if os.IsNotExist(statErr) {
					dt.Items = append(dt.Items, diffItem{Skill: resolved.TargetName, Action: "link", Reason: "missing (deleted from target)", Kind: "skill"})
				} else if statErr != nil {
					dt.Items = append(dt.Items, diffItem{Skill: resolved.TargetName, Action: "update", Reason: "cannot access target entry", Kind: "skill"})
				} else if !targetInfo.IsDir() {
					dt.Items = append(dt.Items, diffItem{Skill: resolved.TargetName, Action: "update", Reason: "target entry is not a directory", Kind: "skill"})
				} else {
					oldMtime := manifest.Mtimes[resolved.TargetName]
					currentMtime, mtimeErr := ssync.DirMaxMtime(skill.SourcePath)
					if mtimeErr == nil && oldMtime > 0 && currentMtime == oldMtime {
						continue
					}
					srcChecksum, checksumErr := ssync.DirChecksum(skill.SourcePath)
					if checksumErr != nil {
						dt.Items = append(dt.Items, diffItem{Skill: resolved.TargetName, Action: "update", Reason: "cannot compute checksum", Kind: "skill"})
					} else if srcChecksum != oldChecksum {
						dt.Items = append(dt.Items, diffItem{Skill: resolved.TargetName, Action: "update", Reason: "content changed", Kind: "skill"})
					}
				}
			}
		}
		for managedName := range manifest.Managed {
			if _, keepLegacy := legacyNames[managedName]; keepLegacy {
				continue
			}
			if !validNames[managedName] {
				dt.Items = append(dt.Items, diffItem{Skill: managedName, Action: "prune", Reason: "orphan copy", Kind: "skill"})
			}
		}
		return dt
	}

	// Merge mode
	for _, resolved := range resolution.Skills {
		skill := resolved.Skill
		targetSkillPath := filepath.Join(sc.Path, resolved.TargetName)
		_, err := os.Lstat(targetSkillPath)
		if err != nil {
			if os.IsNotExist(err) {
				dt.Items = append(dt.Items, diffItem{Skill: resolved.TargetName, Action: "link", Reason: "source only", Kind: "skill"})
			}
			continue
		}

		if utils.IsSymlinkOrJunction(targetSkillPath) {
			absLink, linkErr := utils.ResolveLinkTarget(targetSkillPath)
			if linkErr != nil {
				dt.Items = append(dt.Items, diffItem{Skill: resolved.TargetName, Action: "update", Reason: "link target unreadable", Kind: "skill"})
				continue
			}
			absSource, _ := filepath.Abs(skill.SourcePath)
			if !utils.PathsEqual(absLink, absSource) {
				dt.Items = append(dt.Items, diffItem{Skill: resolved.TargetName, Action: "update", Reason: "symlink points elsewhere", Kind: "skill"})
			}
		} else {
			dt.Items = append(dt.Items, diffItem{Skill: resolved.TargetName, Action: "skip", Reason: "local copy (sync --force to replace)", Kind: "skill"})
		}
	}

	// Orphan check
	entries, _ := os.ReadDir(sc.Path)
	manifest, _ := ssync.ReadManifest(sc.Path)
	for _, entry := range entries {
		eName := entry.Name()
		if utils.IsHidden(eName) {
			continue
		}
		if _, keepLegacy := legacyNames[eName]; keepLegacy {
			continue
		}
		entryPath := filepath.Join(sc.Path, eName)
		if !validNames[eName] {
			info, statErr := os.Lstat(entryPath)
			if statErr != nil {
				continue
			}
			if utils.IsSymlinkOrJunction(entryPath) {
				absLink, linkErr := utils.ResolveLinkTarget(entryPath)
				if linkErr != nil {
					continue
				}
				absSource, _ := filepath.Abs(source)
				if utils.PathHasPrefix(absLink, absSource+string(filepath.Separator)) {
					dt.Items = append(dt.Items, diffItem{Skill: eName, Action: "prune", Reason: "orphan symlink", Kind: "skill"})
				}
			} else if info.IsDir() {
				if _, inManifest := manifest.Managed[eName]; inManifest {
					dt.Items = append(dt.Items, diffItem{Skill: eName, Action: "prune", Reason: "orphan managed directory (manifest)", Kind: "skill"})
				} else {
					if resolution.Naming == "flat" && (utils.HasNestedSeparator(eName) || utils.IsTrackedRepoDir(eName)) {
						dt.Items = append(dt.Items, diffItem{Skill: eName, Action: "prune", Reason: "orphan managed directory", Kind: "skill"})
					} else {
						dt.Items = append(dt.Items, diffItem{Skill: eName, Action: "local", Reason: "local only", Kind: "skill"})
					}
				}
			}
		}
	}

	return dt
}
