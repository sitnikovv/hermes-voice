package cleanup

// DefaultRules returns conservative deterministic speech cleanup rules.
func DefaultRules() []Rule {
	return []Rule{
		{ID: "trim_space_initial", Description: "trim leading and trailing whitespace", Kind: KindTrimSpace},
		{ID: "collapse_space_initial", Description: "collapse repeated whitespace", Kind: KindCollapseSpace},

		{ID: "remove_prefix_okey_hermes", Description: "remove leading wake phrase", Kind: KindRemovePrefixPhrase, Pattern: "окей гермес"},
		{ID: "trim_space_after_okey_hermes", Kind: KindTrimSpace},
		{ID: "collapse_space_after_okey_hermes", Kind: KindCollapseSpace},

		{ID: "remove_prefix_hey_hermes", Description: "remove leading wake phrase", Kind: KindRemovePrefixPhrase, Pattern: "эй гермес"},
		{ID: "trim_space_after_hey_hermes", Kind: KindTrimSpace},
		{ID: "collapse_space_after_hey_hermes", Kind: KindCollapseSpace},

		{ID: "remove_prefix_listen_hermes", Description: "remove leading wake phrase", Kind: KindRemovePrefixPhrase, Pattern: "слушай гермес"},
		{ID: "trim_space_after_listen_hermes", Kind: KindTrimSpace},
		{ID: "collapse_space_after_listen_hermes", Kind: KindCollapseSpace},

		{ID: "remove_prefix_hermes", Description: "remove leading wake phrase", Kind: KindRemovePrefixPhrase, Pattern: "гермес"},
		{ID: "trim_space_after_hermes", Kind: KindTrimSpace},
		{ID: "collapse_space_after_hermes", Kind: KindCollapseSpace},

		{ID: "remove_prefix_filler_nu", Description: "remove leading filler", Kind: KindRemovePrefixPhrase, Pattern: "ну"},
		{ID: "trim_space_after_filler_nu", Kind: KindTrimSpace},
		{ID: "collapse_space_after_filler_nu", Kind: KindCollapseSpace},

		{ID: "remove_prefix_filler_ee", Description: "remove leading filler", Kind: KindRemovePrefixPhrase, Pattern: "ээ"},
		{ID: "trim_space_after_filler_ee", Kind: KindTrimSpace},
		{ID: "collapse_space_after_filler_ee", Kind: KindCollapseSpace},

		{ID: "remove_prefix_filler_em", Description: "remove leading filler", Kind: KindRemovePrefixPhrase, Pattern: "эм"},
		{ID: "trim_space_after_filler_em", Kind: KindTrimSpace},
		{ID: "collapse_space_after_filler_em", Kind: KindCollapseSpace},

		{ID: "remove_suffix_please", Description: "remove trailing polite phrase", Kind: KindRemoveSuffixPhrase, Pattern: "пожалуйста"},
		{ID: "trim_space_final", Description: "trim after cleanup", Kind: KindTrimSpace},
		{ID: "collapse_space_final", Description: "collapse after cleanup", Kind: KindCollapseSpace},
	}
}
