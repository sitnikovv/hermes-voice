package cleanup

import "strings"

// Clean returns the final safe cleaned utterance text.
func (c *Cleaner) Clean(input string) string {
	return c.CleanWithTrace(input).Cleaned
}

// CleanWithTrace applies rules in declaration order and records rules that change text.
func (c *Cleaner) CleanWithTrace(input string) Result {
	current := input
	result := Result{Original: input}

	for _, rule := range c.rules {
		before := current
		current = applyRule(rule, current)
		if current != before {
			result.Applied = append(result.Applied, AppliedRule{
				ID:     rule.ID,
				Before: before,
				After:  current,
			})
		}
	}

	if current == "" && strings.TrimSpace(input) != "" {
		current = collapseWhitespace(strings.TrimSpace(input))
	}
	result.Cleaned = current
	return result
}

func applyRule(rule Rule, input string) string {
	switch rule.Kind {
	case KindTrimSpace:
		return strings.TrimSpace(input)
	case KindCollapseSpace:
		return collapseWhitespace(input)
	case KindRemovePrefixPhrase:
		return strings.TrimPrefix(input, rule.Pattern)
	case KindRemoveSuffixPhrase:
		return strings.TrimSuffix(input, rule.Pattern)
	case KindReplacePhrase:
		return strings.ReplaceAll(input, rule.Pattern, rule.Replacement)
	default:
		return input
	}
}

func collapseWhitespace(input string) string {
	return strings.Join(strings.Fields(input), " ")
}
