package cleanup

import "fmt"

// New validates rules and returns a cleaner that applies a defensive copy of them.
func New(rules []Rule) (*Cleaner, error) {
	seen := make(map[string]struct{}, len(rules))
	copied := make([]Rule, len(rules))
	for i, rule := range rules {
		if rule.ID == "" {
			return nil, fmt.Errorf("cleanup rule %d: id is required", i)
		}
		if _, ok := seen[rule.ID]; ok {
			return nil, fmt.Errorf("cleanup rule %q: duplicate id", rule.ID)
		}
		seen[rule.ID] = struct{}{}

		switch rule.Kind {
		case KindTrimSpace, KindCollapseSpace:
			if rule.Replacement != "" {
				return nil, fmt.Errorf("cleanup rule %q: replacement is only allowed for %s", rule.ID, KindReplacePhrase)
			}
		case KindRemovePrefixPhrase, KindRemoveSuffixPhrase:
			if rule.Pattern == "" {
				return nil, fmt.Errorf("cleanup rule %q: pattern is required for %s", rule.ID, rule.Kind)
			}
			if rule.Replacement != "" {
				return nil, fmt.Errorf("cleanup rule %q: replacement is only allowed for %s", rule.ID, KindReplacePhrase)
			}
		case KindReplacePhrase:
			if rule.Pattern == "" {
				return nil, fmt.Errorf("cleanup rule %q: pattern is required for %s", rule.ID, rule.Kind)
			}
		default:
			return nil, fmt.Errorf("cleanup rule %q: unknown kind %q", rule.ID, rule.Kind)
		}
		copied[i] = rule
	}
	return &Cleaner{rules: copied}, nil
}
