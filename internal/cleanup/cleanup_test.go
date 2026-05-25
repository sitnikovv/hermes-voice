package cleanup

import (
	"reflect"
	"testing"
)

func TestCleanWithTraceAppliesRulesInOrder(t *testing.T) {
	cleaner := mustCleaner(t, []Rule{
		{ID: "trim", Kind: KindTrimSpace},
		{ID: "collapse", Kind: KindCollapseSpace},
		{ID: "prefix", Kind: KindRemovePrefixPhrase, Pattern: "гермес"},
		{ID: "suffix", Kind: KindRemoveSuffixPhrase, Pattern: "пожалуйста"},
		{ID: "replace", Kind: KindReplacePhrase, Pattern: "включи", Replacement: "запусти"},
		{ID: "trim-final", Kind: KindTrimSpace},
		{ID: "collapse-final", Kind: KindCollapseSpace},
	})

	got := cleaner.CleanWithTrace("  гермес   включи свет   пожалуйста  ")
	want := Result{
		Original: "  гермес   включи свет   пожалуйста  ",
		Cleaned:  "запусти свет",
		Applied: []AppliedRule{
			{ID: "trim", Before: "  гермес   включи свет   пожалуйста  ", After: "гермес   включи свет   пожалуйста"},
			{ID: "collapse", Before: "гермес   включи свет   пожалуйста", After: "гермес включи свет пожалуйста"},
			{ID: "prefix", Before: "гермес включи свет пожалуйста", After: " включи свет пожалуйста"},
			{ID: "suffix", Before: " включи свет пожалуйста", After: " включи свет "},
			{ID: "replace", Before: " включи свет ", After: " запусти свет "},
			{ID: "trim-final", Before: " запусти свет ", After: "запусти свет"},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("CleanWithTrace() = %#v, want %#v", got, want)
	}
}

func TestPrefixAndSuffixOnlyMatchBoundaries(t *testing.T) {
	cleaner := mustCleaner(t, []Rule{
		{ID: "prefix", Kind: KindRemovePrefixPhrase, Pattern: "гермес"},
		{ID: "suffix", Kind: KindRemoveSuffixPhrase, Pattern: "пожалуйста"},
		{ID: "trim", Kind: KindTrimSpace},
	})

	if got := cleaner.Clean("вызови гермес пожалуйста завтра"); got != "вызови гермес пожалуйста завтра" {
		t.Fatalf("middle phrases changed: %q", got)
	}
}

func TestCleanReturnsTraceCleaned(t *testing.T) {
	cleaner := mustCleaner(t, []Rule{{ID: "trim", Kind: KindTrimSpace}})
	input := "  текст  "
	if got, want := cleaner.Clean(input), cleaner.CleanWithTrace(input).Cleaned; got != want {
		t.Fatalf("Clean() = %q, want trace cleaned %q", got, want)
	}
}

func TestCleanWithTraceFallbackRecordsAppliedRule(t *testing.T) {
	cleaner := mustCleaner(t, []Rule{
		{ID: "prefix", Kind: KindRemovePrefixPhrase, Pattern: "гермес"},
		{ID: "trim", Kind: KindTrimSpace},
	})

	got := cleaner.CleanWithTrace("гермес")
	if got.Cleaned != "гермес" {
		t.Fatalf("Cleaned = %q, want fallback original", got.Cleaned)
	}
	if len(got.Applied) == 0 || got.Applied[len(got.Applied)-1].ID != "fallback_original" {
		t.Fatalf("Applied = %#v, want final fallback_original rule", got.Applied)
	}
	last := got.Applied[len(got.Applied)-1]
	if last.Before != "" || last.After != "гермес" {
		t.Fatalf("fallback trace = %#v, want before empty after original", last)
	}
}

func mustCleaner(t *testing.T, rules []Rule) *Cleaner {
	t.Helper()
	cleaner, err := New(rules)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	return cleaner
}
