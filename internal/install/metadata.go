package install

import (
	"sort"
	"time"
)

// MetadataFileName is the centralized metadata file stored in each directory.
const MetadataFileName = ".metadata.json"

// MetadataStore holds all entries for a single directory (skills/ or agents/).
type MetadataStore struct {
	Version int                       `json:"version"`
	Entries map[string]*MetadataEntry `json:"entries"`
}

// MetadataEntry merges the old SkillMeta + RegistryEntry fields.
type MetadataEntry struct {
	// Registry fields
	Source  string `json:"source"`
	Kind    string `json:"kind,omitempty"`
	Type    string `json:"type,omitempty"`
	Tracked bool   `json:"tracked,omitempty"`
	Group   string `json:"group,omitempty"`
	Branch  string `json:"branch,omitempty"`
	Into    string `json:"into,omitempty"`
	Name    string `json:"-"` // runtime only, not persisted (map key is the name)

	// Meta fields
	InstalledAt time.Time         `json:"installed_at,omitzero"`
	RepoURL     string            `json:"repo_url,omitempty"`
	Subdir      string            `json:"subdir,omitempty"`
	Version     string            `json:"version,omitempty"`
	TreeHash    string            `json:"tree_hash,omitempty"`
	FileHashes  map[string]string `json:"file_hashes,omitempty"`
}

// NewMetadataStore returns an empty store with version 1.
func NewMetadataStore() *MetadataStore {
	return &MetadataStore{
		Version: 1,
		Entries: make(map[string]*MetadataEntry),
	}
}

// Get returns the entry for the given name, or nil if not found.
func (s *MetadataStore) Get(name string) *MetadataEntry {
	return s.Entries[name]
}

// Set adds or replaces an entry.
func (s *MetadataStore) Set(name string, entry *MetadataEntry) {
	s.Entries[name] = entry
}

// Remove deletes an entry by name.
func (s *MetadataStore) Remove(name string) {
	delete(s.Entries, name)
}

// Has returns true if an entry exists for the given name.
func (s *MetadataStore) Has(name string) bool {
	_, ok := s.Entries[name]
	return ok
}

// List returns sorted entry names.
func (s *MetadataStore) List() []string {
	names := make([]string, 0, len(s.Entries))
	for name := range s.Entries {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// EffectiveKind returns "skill" if Kind is empty.
func (e *MetadataEntry) EffectiveKind() string {
	if e.Kind == "" {
		return "skill"
	}
	return e.Kind
}

// FullName returns "group/name" if Group is set, otherwise Name.
func (e *MetadataEntry) FullName() string {
	if e.Group != "" {
		return e.Group + "/" + e.Name
	}
	return e.Name
}
