package cli

import (
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/websentry-ai/paws/internal/pack"
)

// The skills the test catalog pins: humanizer at its repo's root, tdd in a subfolder of its repo.
var (
	humanizerFiles = fstest.MapFS{"SKILL.md": {Data: []byte("---\nname: humanizer\ndescription: Removes AI tells.\n---\n"), Mode: 0o644}}
	tddFiles       = fstest.MapFS{"SKILL.md": {Data: []byte("---\nname: tdd\ndescription: Test first.\n---\n"), Mode: 0o644}}
)

func hashOf(t *testing.T, fsys fstest.MapFS) string {
	t.Helper()
	h, err := pack.HashSkill(fsys)
	if err != nil {
		t.Fatal(err)
	}
	return h
}

func writeFile(t *testing.T, path, data string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
}

// writeTestCatalog lays out a small catalog the way `paws catalog pull` writes one: two upstream skills, one skill
// Unbound wrote with a digest, a pack, and the trigger cases for every rule file.
func writeTestCatalog(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	const ref = "0123456789abcdef0123456789abcdef01234567"
	upstream := func(repo, path, hash string) string {
		return "upstream:\n  repo: " + repo + "\n  path: " + path + "\n  ref: " + ref + "\n  content_hash: " + hash +
			"\nprovenance:\n  stars: 900\n  license: MIT\n"
	}
	hum := upstream("github.com/blader/humanizer", "SKILL.md", hashOf(t, humanizerFiles))
	writeFile(t, filepath.Join(dir, "catalog/writing/humanizer-pr.md"), "---\nname: humanizer-pr\nevent: pre-tool\nmatcher:\n  tool: Bash\n"+
		"  command_regex: (^|[;&|\\n])\\s*gh\\s+pr\\s+create\\b\naction:\n  type: require-skill\n  skill: humanizer\n"+hum+
		"tags: [writing]\n---\nLoads humanizer before a PR title or body goes out.\n")
	writeFile(t, filepath.Join(dir, "catalog/writing/humanizer-commit.md"), "---\nname: humanizer-commit\nevent: pre-commit\nmatcher:\n  tool: Bash\n"+
		"action:\n  type: require-skill\n  skill: humanizer\n"+hum+"summary: Rewrites text that sounds machine-written.\n"+
		"tags: [writing]\n---\nLoads humanizer before a commit message is written.\n")
	writeFile(t, filepath.Join(dir, "catalog/testing/tdd.md"), "---\nname: tdd\nevent: pre-tool\nmatcher:\n  tool: Write\n"+
		"  file_glob: '**/*.test.ts'\naction:\n  type: require-skill\n  skill: tdd\n"+
		upstream("github.com/mattpocock/skills", "skills/engineering/tdd/SKILL.md", hashOf(t, tddFiles))+
		"summary: Red, green, refactor.\ntags: [testing]\n---\nLoads tdd before a test file is written.\n")
	writeFile(t, filepath.Join(dir, "catalog/source-control/git.md"), "---\nname: git\nevent: pre-tool\nmatcher:\n  tool: Bash\n"+
		"  command_regex: (^|[;&|\\n])\\s*git\\s+push\\b\naction:\n  type: require-skill\n  skill: git\n  on_miss: digest\n"+
		"  digest: digests/git.md\nsummary: Real output before a push.\ntags: [org, git]\n---\nLoads git before a push.\n")
	writeFile(t, filepath.Join(dir, "catalog/source-control/digests/git.md"), "Show real command output before you push.\n")
	writeFile(t, filepath.Join(dir, "catalog/source-control/skills/git/SKILL.md"), "---\nname: git\ndescription: Git habits.\n---\nShow output.\n")
	writeFile(t, filepath.Join(dir, "packs/writing.yaml"), "name: writing\ndescription: Prose that reads like a person wrote it.\n"+
		"bindings: [writing/humanizer-pr.md, writing/humanizer-commit.md]\n")
	writeFile(t, filepath.Join(dir, "trigger_cases.json"), `{
  "humanizer-pr": {"fires": [["Bash", "gh pr create --fill"]], "ignores": [["Bash", "gh pr view"]]},
  "humanizer-commit": {"fires": [["Bash", "git commit -m x"]], "ignores": [["Bash", "echo \"git commit\""]]},
  "tdd": {"fires": [["Write", "src/a.test.ts"]], "ignores": [["Write", "src/a.ts"]]},
  "git": {"fires": [["Bash", "git push origin main"]], "ignores": [["Bash", "git status"]]}
}
`)
	return dir
}
