//go:build online

package integration

import (
	"path/filepath"
	"strings"
	"testing"

	"skillshare/internal/install"
	"skillshare/internal/testutil"
)

// TestInstall_BatchAuditOutput_Antigravity validates that install --all produces
// rich audit output: blocked/failed section, severity breakdown, hints.
func TestInstall_BatchAuditOutput_Antigravity(t *testing.T) {
	sb := testutil.NewSandbox(t)
	defer sb.Cleanup()

	projectRoot := sb.SetupProjectDir("claude")
	result := sb.RunCLIInDir(projectRoot, "install", "sickn33/antigravity-awesome-skills/skills", "--all", "-p")

	// Batch install exits 0 when some skills succeed (blocked count is a warning)
	result.AssertSuccess(t)

	// Blocked section
	result.AssertAnyOutputContains(t, "Blocked / Failed")
	result.AssertAnyOutputContains(t, "blocked by security audit")
	result.AssertAnyOutputContains(t, "CRITICAL")

	// Severity breakdown
	result.AssertAnyOutputContains(t, "finding(s) across")
	result.AssertAnyOutputContains(t, "HIGH")

	// Hint for more details
	result.AssertAnyOutputContains(t, "--audit-verbose")

	// Install count
	result.AssertAnyOutputContains(t, "Installed")

	// Next steps
	result.AssertAnyOutputContains(t, "Next Steps")
}

// TestUpdateAll_AuditOutputParity_Antigravity verifies that update --all produces
// audit output with similar richness to install --all.
func TestUpdateAll_AuditOutputParity_Antigravity(t *testing.T) {
	sb := testutil.NewSandbox(t)
	defer sb.Cleanup()

	projectRoot := sb.SetupProjectDir("claude")

	// Step 1: install (use --force to bypass blocked skills so we have something to update)
	installResult := sb.RunCLIInDir(projectRoot, "install", "sickn33/antigravity-awesome-skills/skills", "--all", "--force", "-p")
	installResult.AssertSuccess(t)

	// Step 2: invalidate one skill's metadata version so update treats it as
	// needing re-install.  Without this, the skip-unchanged optimisation
	// (ec81fc1) causes every skill to be skipped and no audit output is produced.
	skillsDir := filepath.Join(projectRoot, ".skillshare", "skills")
	invalidateOneSkillMeta(t, skillsDir)

	// Step 3: update --all — the invalidated skill gets re-installed, producing audit output
	updateResult := sb.RunCLIInDir(projectRoot, "update", "--all", "-p")
	updateResult.AssertSuccess(t)

	// Audit section present
	updateResult.AssertAnyOutputContains(t, "Audit")

	// Has audit results (CLEAN or findings)
	combined := updateResult.Stdout + updateResult.Stderr
	if !(strings.Contains(combined, "CLEAN") || strings.Contains(combined, "finding(s)")) {
		t.Errorf("expected audit results (CLEAN or findings), got:\nstdout: %s\nstderr: %s",
			updateResult.Stdout, updateResult.Stderr)
	}

	// Batch summary line (most skills are still skipped)
	updateResult.AssertAnyOutputContains(t, "skipped")

	// No blocked skills on re-install (--force was used initially)
	updateResult.AssertOutputNotContains(t, "Blocked / Failed")
	updateResult.AssertOutputNotContains(t, "Blocked / Rolled Back")
}

// invalidateOneSkillMeta finds the first skill with metadata in the centralized
// store and sets its "version" to a stale value, forcing the next update to re-install it.
func invalidateOneSkillMeta(t *testing.T, skillsDir string) {
	t.Helper()

	store := install.LoadMetadataOrNew(skillsDir)
	for _, name := range store.List() {
		entry := store.Get(name)
		if entry == nil || entry.Source == "" {
			continue
		}
		entry.Version = "stale"
		entry.TreeHash = ""
		if err := store.Save(skillsDir); err != nil {
			t.Fatalf("save store: %v", err)
		}
		t.Logf("invalidated metadata for skill %q to force re-install", name)
		return
	}

	t.Fatal("no skill with metadata found to invalidate")
}
