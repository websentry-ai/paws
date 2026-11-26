package hook

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/websentry-ai/paws/internal/outcome"
	"github.com/websentry-ai/paws/internal/rule"
)

// A read of a skill's SKILL.md answers a deny the way the Skill tool does: it is neither the retry of a catch-all
// Bash rule nor another route.
func TestGradingTreatsASKILLmdReadAsTheAnswer(t *testing.T) {
	md := filepath.Join(t.TempDir(), "skills", "git", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(md), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(md, []byte("# git\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	r, err := rule.Parse([]byte("---\nname: any-bash\nevent: pre-tool\nmatcher:\n  tool: Bash\naction:\n  type: require-skill\n  skill: git\n---\n"), "r.md")
	if err != nil {
		t.Fatal(err)
	}
	at := func(i int, c outcome.Call) outcome.Call {
		c.At = time.Unix(int64(i), 0)
		c.ID = string(rune('a' + i))
		return c
	}
	push := CallOf("Bash", map[string]any{"command": "git push"})
	blocked := push
	blocked.Denied = true
	read := CallOf("Bash", map[string]any{"command": "sed -n '1,240p' " + md})
	if read.SkillLoad != "git" {
		t.Fatalf("a sed -n read of git's SKILL.md should load git, got %q", read.SkillLoad)
	}
	res := outcome.Grade(r, []outcome.Call{at(0, blocked), at(1, read), at(2, push)}, 0)
	if res.Outcome != outcome.SameRetry || res.Retry == nil || res.Retry.Target != "git push" {
		t.Errorf("read then retry: want same_retry on git push, got %s %+v", res.Outcome, res.Retry)
	}
	if res := outcome.Grade(r, []outcome.Call{at(0, blocked), at(1, read)}, 0); res.Outcome != outcome.Stopped {
		t.Errorf("read then nothing: want stopped, got %s", res.Outcome)
	}
}
