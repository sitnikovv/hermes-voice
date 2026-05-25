package cleanup

// RuleKind identifies a deterministic cleanup operation.
type RuleKind string

const (
	KindTrimSpace          RuleKind = "trim_space"
	KindCollapseSpace      RuleKind = "collapse_space"
	KindRemovePrefixPhrase RuleKind = "remove_prefix_phrase"
	KindRemoveSuffixPhrase RuleKind = "remove_suffix_phrase"
	KindReplacePhrase      RuleKind = "replace_phrase"
)

// Rule describes one ordered speech cleanup operation.
type Rule struct {
	ID          string
	Description string
	Kind        RuleKind
	Pattern     string
	Replacement string
}

// AppliedRule records a rule that changed text.
type AppliedRule struct {
	ID     string
	Before string
	After  string
}

// Result contains the original utterance, final safe cleanup output, and trace.
type Result struct {
	Original string
	Cleaned  string
	Applied  []AppliedRule
}

// Cleaner applies rules in declaration order.
type Cleaner struct {
	rules []Rule
}
