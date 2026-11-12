package pack

import (
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
)

// writeSkill refuses a file name with a backslash: on Windows a\..\x would land outside the skill folder.
func TestWriteSkillRefusesBackslashPaths(t *testing.T) {
	skills := t.TempDir()
	src := fstest.MapFS{"SKILL.md": {Data: []byte("# s\n")}, `a\..\..\x`: {Data: []byte("x")}}
	if err := writeSkill(src, skills, "s", marker{Skill: "s"}); err == nil {
		t.Fatal("a backslash path was written")
	}
	if _, err := os.Stat(filepath.Join(skills, "s")); !os.IsNotExist(err) {
		t.Error("nothing may be installed when a file is refused")
	}
}
