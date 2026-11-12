package cli

import (
	"testing"
	"testing/fstest"

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

// A file name the backend sends must stay inside the skill folder on every OS: on Windows \ is a separator and C: a
// drive, so those are refused too.
func TestAddFilesRefusesPathsThatLeaveTheSkill(t *testing.T) {
	for _, p := range []string{`../x`, `/etc/x`, `a/../../x`, `a\..\..\x`, `C:x`, "a\x00b", ``} {
		if err := addFiles(fstest.MapFS{}, "s", []remoteFile{{Path: p}}); err == nil {
			t.Errorf("%q was accepted", p)
		}
	}
	if err := addFiles(fstest.MapFS{}, "s", []remoteFile{{Path: "scripts/run.sh"}}); err != nil {
		t.Errorf("a plain path was refused: %v", err)
	}
}
