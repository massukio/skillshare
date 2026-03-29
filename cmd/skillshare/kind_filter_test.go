package main

import "testing"

func TestParseKindArg(t *testing.T) {
	tests := []struct {
		args     []string
		wantKind resourceKindFilter
		wantRest []string
	}{
		{nil, kindAll, nil},
		{[]string{}, kindAll, []string{}},
		{[]string{"skills"}, kindSkills, []string{}},
		{[]string{"skill"}, kindSkills, []string{}},
		{[]string{"agents"}, kindAgents, []string{}},
		{[]string{"agent"}, kindAgents, []string{}},
		{[]string{"agents", "tutor"}, kindAgents, []string{"tutor"}},
		{[]string{"--json"}, kindAll, []string{"--json"}},
		{[]string{"my-skill"}, kindAll, []string{"my-skill"}},
	}

	for _, tt := range tests {
		kind, rest := parseKindArg(tt.args)
		if kind != tt.wantKind {
			t.Errorf("parseKindArg(%v) kind = %v, want %v", tt.args, kind, tt.wantKind)
		}
		if len(rest) != len(tt.wantRest) {
			t.Errorf("parseKindArg(%v) rest = %v, want %v", tt.args, rest, tt.wantRest)
		}
	}
}

func TestParseKindFlag(t *testing.T) {
	tests := []struct {
		args     []string
		wantKind resourceKindFilter
		wantRest []string
		wantErr  bool
	}{
		{[]string{}, kindAll, []string{}, false},
		{[]string{"--kind", "agent"}, kindAgents, []string{}, false},
		{[]string{"--kind", "skill"}, kindSkills, []string{}, false},
		{[]string{"--json", "--kind", "agent", "foo"}, kindAgents, []string{"--json", "foo"}, false},
		{[]string{"--kind"}, kindAll, nil, true},
		{[]string{"--kind", "invalid"}, kindAll, nil, true},
	}

	for _, tt := range tests {
		kind, rest, err := parseKindFlag(tt.args)
		if (err != nil) != tt.wantErr {
			t.Errorf("parseKindFlag(%v) err = %v, wantErr %v", tt.args, err, tt.wantErr)
			continue
		}
		if err != nil {
			continue
		}
		if kind != tt.wantKind {
			t.Errorf("parseKindFlag(%v) kind = %v, want %v", tt.args, kind, tt.wantKind)
		}
		if len(rest) != len(tt.wantRest) {
			t.Errorf("parseKindFlag(%v) rest = %v, want %v", tt.args, rest, tt.wantRest)
		}
	}
}

func TestResourceKindFilter_Includes(t *testing.T) {
	if !kindAll.IncludesSkills() || !kindAll.IncludesAgents() {
		t.Error("kindAll should include both")
	}
	if !kindSkills.IncludesSkills() || kindSkills.IncludesAgents() {
		t.Error("kindSkills should include only skills")
	}
	if kindAgents.IncludesSkills() || !kindAgents.IncludesAgents() {
		t.Error("kindAgents should include only agents")
	}
}
