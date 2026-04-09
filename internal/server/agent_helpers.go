package server

import (
	"skillshare/internal/config"
	"skillshare/internal/resource"
)

// Kind constants for diff/sync operations.
const (
	kindSkill = "skill"
	kindAgent = "agent"
)

// discoverActiveAgents discovers agents from the given source directory,
// returning only non-disabled agents. Returns nil if source is empty.
func discoverActiveAgents(agentsSource string) []resource.DiscoveredResource {
	if agentsSource == "" {
		return nil
	}
	discovered, _ := resource.AgentKind{}.Discover(agentsSource)
	return resource.ActiveAgents(discovered)
}

// resolveAgentPath returns the expanded agent target path for a target,
// checking user config first, then builtin defaults. Returns "" if no path.
func resolveAgentPath(target config.TargetConfig, builtinAgents map[string]config.TargetConfig, name string) string {
	if ac := target.AgentsConfig(); ac.Path != "" {
		return config.ExpandPath(ac.Path)
	}
	if builtin, ok := builtinAgents[name]; ok {
		return config.ExpandPath(builtin.Path)
	}
	return ""
}

// builtinAgentTargets returns the builtin agent target map for the server's mode.
func (s *Server) builtinAgentTargets() map[string]config.TargetConfig {
	if s.IsProjectMode() {
		return config.ProjectAgentTargets()
	}
	return config.DefaultAgentTargets()
}

// mergeAgentDiffItems appends agent diff items into the existing diffs slice,
// merging with an existing target entry or creating a new one.
func mergeAgentDiffItems(diffs []diffTarget, name string, items []diffItem) []diffTarget {
	for i := range diffs {
		if diffs[i].Target == name {
			diffs[i].Items = append(diffs[i].Items, items...)
			return diffs
		}
	}
	return append(diffs, diffTarget{
		Target: name,
		Items:  items,
	})
}
