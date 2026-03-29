package main

import "fmt"

// resourceKindFilter represents the kind filtering for CLI commands.
type resourceKindFilter int

const (
	kindAll    resourceKindFilter = iota // no filter — all kinds
	kindSkills                           // skills only
	kindAgents                           // agents only
)

// parseKindArg extracts a kind filter from the first positional argument.
// Returns the filter and remaining args.
// Recognized values: "skills", "skill", "agents", "agent".
// If the first arg is not a kind keyword, returns kindAll with args unchanged.
func parseKindArg(args []string) (resourceKindFilter, []string) {
	if len(args) == 0 {
		return kindAll, args
	}

	switch args[0] {
	case "skills", "skill":
		return kindSkills, args[1:]
	case "agents", "agent":
		return kindAgents, args[1:]
	default:
		return kindAll, args
	}
}

// parseKindFlag extracts --kind flag from args.
// Returns the filter and remaining args with --kind removed.
func parseKindFlag(args []string) (resourceKindFilter, []string, error) {
	kind := kindAll
	rest := make([]string, 0, len(args))

	for i := 0; i < len(args); i++ {
		if args[i] == "--kind" {
			if i+1 >= len(args) {
				return kindAll, nil, fmt.Errorf("--kind requires a value (skill or agent)")
			}
			i++
			switch args[i] {
			case "skill", "skills":
				kind = kindSkills
			case "agent", "agents":
				kind = kindAgents
			default:
				return kindAll, nil, fmt.Errorf("--kind must be 'skill' or 'agent', got %q", args[i])
			}
		} else {
			rest = append(rest, args[i])
		}
	}

	return kind, rest, nil
}

func (k resourceKindFilter) String() string {
	switch k {
	case kindSkills:
		return "skills"
	case kindAgents:
		return "agents"
	default:
		return "all"
	}
}

func (k resourceKindFilter) IncludesSkills() bool {
	return k == kindAll || k == kindSkills
}

func (k resourceKindFilter) IncludesAgents() bool {
	return k == kindAll || k == kindAgents
}
