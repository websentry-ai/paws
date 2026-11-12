package cli

import (
	"testing"

	"github.com/websentry-ai/paws/internal/rule"
)

// A catalog trigger starts with rule.CommandStart; it reads as its command, like a plain anchored regex.
func TestRegexDisplayReadsTheCatalogAnchor(t *testing.T) {
	for re, want := range map[string]string{
		rule.CommandStart + `gh\s+pr\s+(create|edit)\b`: "gh pr (create|edit)",
		`(^|[;&|\n])\s*gh\s+pr\s+(create|edit)\b`:       "gh pr (create|edit)",
	} {
		if got := regexDisplay(re); got != want {
			t.Errorf("regexDisplay(%q) = %q, want %q", re, got, want)
		}
	}
}
