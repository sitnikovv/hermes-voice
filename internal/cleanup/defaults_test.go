package cleanup

import "testing"

func TestDefaultRulesCleanConservativeSpeechNoise(t *testing.T) {
	cleaner := mustCleaner(t, DefaultRules())

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "leading hermes", input: "гермес включи свет", want: "включи свет"},
		{name: "leading okey hermes", input: "окей гермес включи свет", want: "включи свет"},
		{name: "leading hey hermes", input: "эй гермес включи свет", want: "включи свет"},
		{name: "leading listen hermes", input: "слушай гермес включи свет", want: "включи свет"},
		{name: "leading nu filler", input: "ну включи свет", want: "включи свет"},
		{name: "leading ee filler", input: "ээ включи свет", want: "включи свет"},
		{name: "leading em filler", input: "эм включи свет", want: "включи свет"},
		{name: "trailing polite phrase", input: "включи свет пожалуйста", want: "включи свет"},
		{name: "combined defaults", input: "  окей гермес   ну   включи свет   пожалуйста  ", want: "включи свет"},
		{name: "middle words preserved", input: "включи гермес и пожалуйста свет", want: "включи гермес и пожалуйста свет"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := cleaner.Clean(tt.input); got != tt.want {
				t.Fatalf("Clean(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestDefaultRulesAreIdempotent(t *testing.T) {
	cleaner := mustCleaner(t, DefaultRules())
	inputs := []string{
		"  окей гермес   ну   включи свет   пожалуйста  ",
		"включи гермес и пожалуйста свет",
		"нет стоп отмена",
	}

	for _, input := range inputs {
		first := cleaner.Clean(input)
		second := cleaner.Clean(first)
		if first != second {
			t.Fatalf("default cleanup not idempotent for %q: first %q, second %q", input, first, second)
		}
	}
}

func TestDefaultRulesFallbackWhenNonWhitespaceWouldBecomeEmpty(t *testing.T) {
	cleaner := mustCleaner(t, DefaultRules())

	if got := cleaner.Clean("  гермес  "); got != "гермес" {
		t.Fatalf("Clean() = %q, want safe original", got)
	}
	if got := cleaner.Clean("  \t  "); got != "" {
		t.Fatalf("whitespace-only Clean() = %q, want empty", got)
	}
}

func TestDefaultRulesPreserveIntentAndNames(t *testing.T) {
	cleaner := mustCleaner(t, DefaultRules())

	for _, input := range []string{
		"не включай свет",
		"нет",
		"стоп",
		"отмена",
		"позови гермес",
		"передай Алисе пожалуйста привет завтра",
		"для Саши включи музыку",
		"нук включи свет",
		"герместон включи свет",
		"включи перепожалуйста",
	} {
		t.Run(input, func(t *testing.T) {
			if got := cleaner.Clean(input); got != input {
				t.Fatalf("Clean(%q) = %q, want unchanged", input, got)
			}
		})
	}
}
