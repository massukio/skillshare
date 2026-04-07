package install

import (
	"testing"
	"time"
)

func TestMetadataStore_SetAndGet(t *testing.T) {
	s := NewMetadataStore()
	now := time.Now()
	entry := &MetadataEntry{
		Source:      "org/repo",
		Kind:        "skill",
		Type:        "github",
		Tracked:     true,
		Group:       "mygroup",
		Branch:      "main",
		Into:        "frontend",
		InstalledAt: now,
		RepoURL:     "https://github.com/org/repo.git",
		Subdir:      "skills/foo",
		Version:     "abc123",
		TreeHash:    "deadbeef",
		FileHashes:  map[string]string{"SKILL.md": "sha256:aabbcc"},
	}

	s.Set("foo", entry)
	got := s.Get("foo")

	if got == nil {
		t.Fatal("Get returned nil after Set")
	}
	if got.Source != entry.Source {
		t.Errorf("Source = %q, want %q", got.Source, entry.Source)
	}
	if got.Kind != entry.Kind {
		t.Errorf("Kind = %q, want %q", got.Kind, entry.Kind)
	}
	if got.Type != entry.Type {
		t.Errorf("Type = %q, want %q", got.Type, entry.Type)
	}
	if got.Tracked != entry.Tracked {
		t.Errorf("Tracked = %v, want %v", got.Tracked, entry.Tracked)
	}
	if got.Group != entry.Group {
		t.Errorf("Group = %q, want %q", got.Group, entry.Group)
	}
	if got.Branch != entry.Branch {
		t.Errorf("Branch = %q, want %q", got.Branch, entry.Branch)
	}
	if got.Into != entry.Into {
		t.Errorf("Into = %q, want %q", got.Into, entry.Into)
	}
	if !got.InstalledAt.Equal(entry.InstalledAt) {
		t.Errorf("InstalledAt = %v, want %v", got.InstalledAt, entry.InstalledAt)
	}
	if got.RepoURL != entry.RepoURL {
		t.Errorf("RepoURL = %q, want %q", got.RepoURL, entry.RepoURL)
	}
	if got.Subdir != entry.Subdir {
		t.Errorf("Subdir = %q, want %q", got.Subdir, entry.Subdir)
	}
	if got.Version != entry.Version {
		t.Errorf("Version = %q, want %q", got.Version, entry.Version)
	}
	if got.TreeHash != entry.TreeHash {
		t.Errorf("TreeHash = %q, want %q", got.TreeHash, entry.TreeHash)
	}
	if len(got.FileHashes) != 1 || got.FileHashes["SKILL.md"] != "sha256:aabbcc" {
		t.Errorf("FileHashes = %v, want map with one entry", got.FileHashes)
	}
}

func TestMetadataStore_GetMissing(t *testing.T) {
	s := NewMetadataStore()
	got := s.Get("nonexistent")
	if got != nil {
		t.Errorf("Get nonexistent = %v, want nil", got)
	}
}

func TestMetadataStore_Has(t *testing.T) {
	s := NewMetadataStore()
	s.Set("present", &MetadataEntry{Source: "org/repo"})

	if !s.Has("present") {
		t.Error("Has(present) = false, want true")
	}
	if s.Has("absent") {
		t.Error("Has(absent) = true, want false")
	}
}

func TestMetadataStore_Remove(t *testing.T) {
	s := NewMetadataStore()
	s.Set("to-remove", &MetadataEntry{Source: "org/repo"})

	if !s.Has("to-remove") {
		t.Fatal("entry should exist before Remove")
	}

	s.Remove("to-remove")

	if s.Has("to-remove") {
		t.Error("entry still present after Remove")
	}
	if s.Get("to-remove") != nil {
		t.Error("Get after Remove should return nil")
	}
}

func TestMetadataStore_Remove_Nonexistent(t *testing.T) {
	s := NewMetadataStore()
	// Should not panic
	s.Remove("nonexistent")
}

func TestMetadataStore_List(t *testing.T) {
	s := NewMetadataStore()
	s.Set("zebra", &MetadataEntry{})
	s.Set("alpha", &MetadataEntry{})
	s.Set("mango", &MetadataEntry{})

	names := s.List()

	if len(names) != 3 {
		t.Fatalf("List() = %v, want 3 entries", names)
	}
	want := []string{"alpha", "mango", "zebra"}
	for i, w := range want {
		if names[i] != w {
			t.Errorf("List()[%d] = %q, want %q", i, names[i], w)
		}
	}
}

func TestMetadataStore_List_Empty(t *testing.T) {
	s := NewMetadataStore()
	names := s.List()
	if len(names) != 0 {
		t.Errorf("List() on empty store = %v, want []", names)
	}
}

func TestMetadataEntry_EffectiveKind(t *testing.T) {
	tests := []struct {
		name string
		kind string
		want string
	}{
		{"empty kind defaults to skill", "", "skill"},
		{"explicit skill", "skill", "skill"},
		{"agent", "agent", "agent"},
		{"custom kind preserved", "custom", "custom"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &MetadataEntry{Kind: tt.kind}
			got := e.EffectiveKind()
			if got != tt.want {
				t.Errorf("EffectiveKind() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestMetadataEntry_FullName(t *testing.T) {
	tests := []struct {
		name  string
		group string
		entry string
		want  string
	}{
		{"no group", "", "my-skill", "my-skill"},
		{"with group", "frontend", "my-skill", "frontend/my-skill"},
		{"nested group", "team/frontend", "my-skill", "team/frontend/my-skill"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &MetadataEntry{
				Name:  tt.entry,
				Group: tt.group,
			}
			got := e.FullName()
			if got != tt.want {
				t.Errorf("FullName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNewMetadataStore_InitialState(t *testing.T) {
	s := NewMetadataStore()
	if s == nil {
		t.Fatal("NewMetadataStore returned nil")
	}
	if s.Version != 1 {
		t.Errorf("Version = %d, want 1", s.Version)
	}
	if s.Entries == nil {
		t.Error("Entries map is nil")
	}
	if len(s.Entries) != 0 {
		t.Errorf("Entries not empty on new store: %v", s.Entries)
	}
}
