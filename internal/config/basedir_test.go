package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestBaseDir_DefaultFallback(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")
	home, _ := os.UserHomeDir()

	got := BaseDir()

	if runtime.GOOS == "windows" {
		winDir, _ := os.UserConfigDir()
		want := filepath.Join(winDir, "skillshare")
		if got != want {
			t.Errorf("BaseDir() = %q, want %q", got, want)
		}
	} else {
		want := filepath.Join(home, ".config", "skillshare")
		if got != want {
			t.Errorf("BaseDir() = %q, want %q", got, want)
		}
	}
}

func TestBaseDir_RespectsXDGConfigHome(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/custom/config")

	got := BaseDir()
	want := filepath.Join("/custom/config", "skillshare")
	if got != want {
		t.Errorf("BaseDir() = %q, want %q", got, want)
	}
}

func TestConfigPath_RespectsXDGConfigHome(t *testing.T) {
	t.Setenv("SKILLSHARE_CONFIG", "")
	t.Setenv("XDG_CONFIG_HOME", "/custom/config")

	got := ConfigPath()
	want := filepath.Join("/custom/config", "skillshare", "config.yaml")
	if got != want {
		t.Errorf("ConfigPath() = %q, want %q", got, want)
	}
}

func TestEffectiveAgentsSource_Default(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")
	cfg := &Config{}

	got := cfg.EffectiveAgentsSource()
	want := filepath.Join(BaseDir(), "agents")
	if got != want {
		t.Errorf("EffectiveAgentsSource() = %q, want %q", got, want)
	}
}

func TestEffectiveAgentsSource_Explicit(t *testing.T) {
	cfg := &Config{AgentsSource: "/custom/agents"}

	got := cfg.EffectiveAgentsSource()
	if got != "/custom/agents" {
		t.Errorf("EffectiveAgentsSource() = %q, want %q", got, "/custom/agents")
	}
}

func TestConfigPath_SKILLSHARECONFIGTakesPriority(t *testing.T) {
	t.Setenv("SKILLSHARE_CONFIG", "/override/config.yaml")
	t.Setenv("XDG_CONFIG_HOME", "/custom/config")

	got := ConfigPath()
	want := "/override/config.yaml"
	if got != want {
		t.Errorf("ConfigPath() = %q, want %q", got, want)
	}
}
