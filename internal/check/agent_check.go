package check

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"skillshare/internal/install"
	"skillshare/internal/utils"
)

// AgentCheckResult holds the check result for a single agent.
type AgentCheckResult struct {
	Name    string `json:"name"`
	Source  string `json:"source,omitempty"`
	Version string `json:"version,omitempty"`
	Status  string `json:"status"` // "up_to_date", "drifted", "local", "error"
	Message string `json:"message,omitempty"`
}

// CheckAgents scans the agents source directory for installed agents and
// compares their file hashes against metadata to detect drift.
func CheckAgents(agentsDir string) []AgentCheckResult {
	entries, err := os.ReadDir(agentsDir)
	if err != nil {
		return nil
	}

	var results []AgentCheckResult

	for _, entry := range entries {
		name := entry.Name()

		// Agent .md files
		if !entry.IsDir() && strings.HasSuffix(strings.ToLower(name), ".md") {
			agentName := strings.TrimSuffix(name, ".md")
			result := checkOneAgent(agentsDir, agentName, name)
			results = append(results, result)
		}
	}

	return results
}

func checkOneAgent(agentsDir, agentName, fileName string) AgentCheckResult {
	result := AgentCheckResult{Name: agentName}

	// Look for metadata file: <name>.skillshare-meta.json
	metaPath := filepath.Join(agentsDir, agentName+".skillshare-meta.json")
	metaData, err := os.ReadFile(metaPath)
	if err != nil {
		result.Status = "local"
		return result
	}

	var meta install.SkillMeta
	if err := json.Unmarshal(metaData, &meta); err != nil {
		result.Status = "error"
		result.Message = "invalid metadata"
		return result
	}

	result.Source = meta.Source
	result.Version = meta.Version

	// Compare file hash
	agentPath := filepath.Join(agentsDir, fileName)
	if meta.FileHashes == nil || meta.FileHashes[fileName] == "" {
		result.Status = "local"
		return result
	}

	currentHash, err := utils.FileHashFormatted(agentPath)
	if err != nil {
		result.Status = "error"
		result.Message = "cannot hash file"
		return result
	}

	if currentHash == meta.FileHashes[fileName] {
		result.Status = "up_to_date"
	} else {
		result.Status = "drifted"
		result.Message = "file content changed since install"
	}

	return result
}
