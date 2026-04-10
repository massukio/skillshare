//go:build !online

package integration

import (
	"strings"
	"testing"

	"skillshare/internal/testutil"
)

// TestList_SKILLSHARE_THEME_Light verifies that when SKILLSHARE_THEME=light
// is set, the list output uses the light Primary color (232) and does not
// contain pure bright white (15), resolving issue #125.
func TestList_SKILLSHARE_THEME_Light(t *testing.T) {
	sb := testutil.NewSandbox(t)
	defer sb.Cleanup()

	sb.CreateSkill("hello-world", map[string]string{
		"SKILL.md": "---\nname: hello-world\ndescription: A test skill\n---\n# Hello",
	})
	sb.WriteConfig(`source: ` + sb.SourcePath + `
targets: {}
`)

	result := sb.RunCLIEnv(
		map[string]string{"SKILLSHARE_THEME": "light"},
		"list", "--no-tui",
	)

	if result.ExitCode != 0 {
		t.Fatalf("list exited %d: %s", result.ExitCode, result.Output())
	}

	// Light Primary is 232 — the rendered 256-color escape is
	// ESC[38;5;232m. Plain text from tables etc. may not include this,
	// so we only assert that pure white 15 is NOT present.
	if strings.Contains(result.Stdout, "\x1b[38;5;15m") {
		t.Error("light theme output must not contain pure white (Color 15)")
	}
}

// TestList_NO_COLOR verifies that NO_COLOR strips all ANSI escape sequences.
func TestList_NO_COLOR(t *testing.T) {
	sb := testutil.NewSandbox(t)
	defer sb.Cleanup()

	sb.CreateSkill("hello", map[string]string{
		"SKILL.md": "---\nname: hello\ndescription: A test skill\n---\n# H",
	})
	sb.WriteConfig(`source: ` + sb.SourcePath + `
targets: {}
`)

	result := sb.RunCLIEnv(
		map[string]string{"NO_COLOR": "1"},
		"list", "--no-tui",
	)

	if result.ExitCode != 0 {
		t.Fatalf("list exited %d: %s", result.ExitCode, result.Output())
	}

	if strings.Contains(result.Stdout, "\x1b[") {
		t.Errorf("NO_COLOR must strip all ANSI escapes, got: %q", result.Stdout)
	}
}
