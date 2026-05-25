package cleanup

import "testing"

func TestNewValidatesRules(t *testing.T) {
	tests := []struct {
		name    string
		rules   []Rule
		wantErr bool
	}{
		{
			name: "rejects empty id",
			rules: []Rule{{
				Kind: KindTrimSpace,
			}},
			wantErr: true,
		},
		{
			name: "rejects duplicate id",
			rules: []Rule{
				{ID: "trim", Kind: KindTrimSpace},
				{ID: "trim", Kind: KindCollapseSpace},
			},
			wantErr: true,
		},
		{
			name: "rejects unknown kind",
			rules: []Rule{{
				ID:   "mystery",
				Kind: RuleKind("mystery"),
			}},
			wantErr: true,
		},
		{
			name: "rejects missing prefix pattern",
			rules: []Rule{{
				ID:   "prefix",
				Kind: KindRemovePrefixPhrase,
			}},
			wantErr: true,
		},
		{
			name: "rejects missing suffix pattern",
			rules: []Rule{{
				ID:   "suffix",
				Kind: KindRemoveSuffixPhrase,
			}},
			wantErr: true,
		},
		{
			name: "rejects missing replace pattern",
			rules: []Rule{{
				ID:          "replace",
				Kind:        KindReplacePhrase,
				Replacement: "after",
			}},
			wantErr: true,
		},
		{
			name: "accepts valid rules",
			rules: []Rule{
				{ID: "trim", Kind: KindTrimSpace},
				{ID: "collapse", Kind: KindCollapseSpace},
				{ID: "prefix", Kind: KindRemovePrefixPhrase, Pattern: "гермес"},
				{ID: "suffix", Kind: KindRemoveSuffixPhrase, Pattern: "пожалуйста"},
				{ID: "replace", Kind: KindReplacePhrase, Pattern: "ё", Replacement: "е"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(tt.rules)
			if tt.wantErr && err == nil {
				t.Fatal("expected error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
