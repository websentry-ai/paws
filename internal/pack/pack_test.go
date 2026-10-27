package pack

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/websentry-ai/paws/internal/catalog"
)

func write(t *testing.T, path, data string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
}

// upstreamRepo is a fake upstream checkout served by the test Fetch.
func upstreamRepo(t *testing.T) string {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "SKILL.md"), "---\nname: humanizer\n---\nWrite like a person.\n")
	write(t, filepath.Join(dir, "references", "patterns.md"), "patterns\n")
	write(t, filepath.Join(dir, ".github", "ci.yml"), "ignored\n")
	write(t, filepath.Join(dir, "scripts", "run.sh"), "#!/bin/sh\n")
	os.Chmod(filepath.Join(dir, "scripts", "run.sh"), 0o755)
	return dir
}

func copyTree(src, dest string) error {
	return filepath.WalkDir(src, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, p)
		out := filepath.Join(dest, rel)
		if d.IsDir() {
			return os.MkdirAll(out, 0o755)
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		st, _ := d.Info()
		return os.WriteFile(out, data, st.Mode().Perm())
	})
}

func fakeFetch(t *testing.T, repos map[string]string) {
	t.Helper()
	orig := Fetch
	Fetch = func(repo, ref, dest string) error {
		src, ok := repos[repo]
		if !ok {
			return os.ErrNotExist
		}
		return copyTree(src, dest)
	}
	t.Cleanup(func() { Fetch = orig })
}

// testCatalog builds a catalog with an upstream binding (pinned to hash), an org binding with a
// local skill, placeholders and a digest, an exec-script binding, and a pack with a pick_one group.
func testCatalog(t *testing.T, hash string) *catalog.Catalog {
	root := t.TempDir()
	up := "upstream:\n  repo: github.com/example/humanizer\n  path: SKILL.md\n  ref: 0123456789abcdef0123456789abcdef01234567\n  content_hash: " + hash + "\n"
	write(t, filepath.Join(root, "catalog/writing/humanizer-pr.md"), "---\nname: humanizer-pr\nevent: pre-tool\nmatcher:\n  tool: Bash\n  command_regex: gh pr create\naction:\n  type: require-skill\n  skill: humanizer\n"+up+"tags: [writing]\n---\nHumanize PRs.\n")
	write(t, filepath.Join(root, "catalog/writing/plain-pr.md"), "---\nname: plain-pr\nevent: pre-tool\nmatcher:\n  tool: Bash\n  command_regex: gh pr create\naction:\n  type: audit\ntags: [writing]\n---\n")
	write(t, filepath.Join(root, "catalog/aws/s3.md"), "---\nname: aws-s3\nevent: pre-tool\nmatcher:\n  tool: Bash\n  command_regex: aws s3 mb\naction:\n  type: require-skill\n  skill: aws-s3\n  on_miss: digest\n  digest: digests/aws-s3.md\ntags: [org, aws]\n---\n")
	write(t, filepath.Join(root, "catalog/aws/digests/aws-s3.md"), "Block public access.\n")
	write(t, filepath.Join(root, "catalog/aws/skills/aws-s3/SKILL.md"), "---\nname: aws-s3\n---\nPrefix buckets with {{BUCKET_PREFIX}}.\n")
	write(t, filepath.Join(root, "catalog/aws/skills/aws-s3/config.yaml"), "placeholders:\n  BUCKET_PREFIX: acme-\n")
	write(t, filepath.Join(root, "catalog/shell/rm.md"), "---\nname: rm-guard\nevent: pre-tool\nmatcher:\n  tool: Bash\naction:\n  type: exec-script\n  script: rm.sh\ntags: [safety]\n---\n")
	write(t, filepath.Join(root, "catalog/shell/rm.sh"), "#!/bin/sh\nexit 0\n")
	write(t, filepath.Join(root, "packs/writing.yaml"), "name: writing\ndescription: d\npick_one:\n  - name: prose\n    default: humanizer\n    options:\n      humanizer: [writing/humanizer-pr.md]\n      plain: [writing/plain-pr.md]\n")
	write(t, filepath.Join(root, "packs/aws.yaml"), "name: aws\ndescription: d\nbindings: [aws/s3.md, shell/rm.md]\n")
	c, probs := catalog.Validate(root)
	if len(probs) > 0 {
		t.Fatalf("test catalog invalid: %v", probs)
	}
	return c
}

func goodHash(t *testing.T) (string, string) {
	repo := upstreamRepo(t)
	h, err := HashSkill(os.DirFS(repo))
	if err != nil {
		t.Fatal(err)
	}
	return repo, h
}

func add(t *testing.T, c *catalog.Catalog, name string, picks map[string]string, tg Target) *Result {
	t.Helper()
	pk, bindings, chosen, err := Resolve(c, name, picks, nil)
	if err != nil {
		t.Fatal(err)
	}
	res, err := Add(c, pk, bindings, chosen, tg)
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func TestAddInstallsBindingsAndUpstreamSkill(t *testing.T) {
	repo, hash := goodHash(t)
	fakeFetch(t, map[string]string{"github.com/example/humanizer": repo})
	c := testCatalog(t, hash)
	tg := ProjectTarget(t.TempDir())

	res := add(t, c, "writing", nil, tg)
	if len(res.Installed) != 1 || res.Installed[0] != "humanizer" {
		t.Fatalf("humanizer should be installed, got %+v", res)
	}
	if _, err := os.Stat(filepath.Join(tg.PacksDir, "writing", "writing", "humanizer-pr.md")); err != nil {
		t.Errorf("binding not installed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(tg.PacksDir, "writing", "writing", "plain-pr.md")); err == nil {
		t.Error("the non-default pick_one option must not be installed")
	}
	skill := filepath.Join(tg.SkillsDir, "humanizer")
	for _, f := range []string{"SKILL.md", "references/patterns.md", markerFile} {
		if _, err := os.Stat(filepath.Join(skill, f)); err != nil {
			t.Errorf("skill file %s missing: %v", f, err)
		}
	}
	if _, err := os.Stat(filepath.Join(skill, ".github")); err == nil {
		t.Error("dot directories from upstream must not be copied")
	}
	st, _ := LoadState(tg.StateFile)
	if st.Packs["writing"] == nil || st.Packs["writing"].Picks["prose"] != "humanizer" {
		t.Errorf("state should record the pack and its pick, got %+v", st.Packs["writing"])
	}
}

func TestAddRefusesHashMismatch(t *testing.T) {
	repo, _ := goodHash(t)
	fakeFetch(t, map[string]string{"github.com/example/humanizer": repo})
	c := testCatalog(t, "sha256:"+strings.Repeat("0", 64))
	tg := ProjectTarget(t.TempDir())
	pk, bindings, picks, _ := Resolve(c, "writing", nil, nil)
	_, err := Add(c, pk, bindings, picks, tg)
	if err == nil || !strings.Contains(err.Error(), "content hash mismatch") {
		t.Fatalf("want a hash mismatch refusal, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(tg.SkillsDir, "humanizer")); err == nil {
		t.Error("nothing may be installed on a mismatch")
	}
	if _, err := os.Stat(filepath.Join(tg.PacksDir, "writing")); err == nil {
		t.Error("bindings must not go live without their skill")
	}
}

func TestAddKeepsUsersOwnSkill(t *testing.T) {
	fakeFetch(t, map[string]string{}) // any fetch would fail: the user's copy must short-circuit it
	_, hash := goodHash(t)
	c := testCatalog(t, hash)
	tg := ProjectTarget(t.TempDir())
	write(t, filepath.Join(tg.SkillsDir, "humanizer", "SKILL.md"), "my own humanizer\n")

	res := add(t, c, "writing", nil, tg)
	if len(res.Kept) != 1 || len(res.Installed) != 0 {
		t.Fatalf("the user's skill should be kept, got %+v", res)
	}
	if data, _ := os.ReadFile(filepath.Join(tg.SkillsDir, "humanizer", "SKILL.md")); string(data) != "my own humanizer\n" {
		t.Error("the user's skill was modified")
	}
	if _, err := Remove("writing", tg); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(tg.SkillsDir, "humanizer", "SKILL.md")); err != nil {
		t.Error("remove must never delete a skill paws did not install")
	}
}

func TestAddOrgSkillFillsPlaceholdersAndCopiesDigestAndScript(t *testing.T) {
	_, hash := goodHash(t)
	c := testCatalog(t, hash)
	tg := ProjectTarget(t.TempDir())
	add(t, c, "aws", nil, tg)

	data, err := os.ReadFile(filepath.Join(tg.SkillsDir, "aws-s3", "SKILL.md"))
	if err != nil || !strings.Contains(string(data), "Prefix buckets with acme-.") {
		t.Errorf("placeholders not filled: %q %v", data, err)
	}
	if _, err := os.Stat(filepath.Join(tg.PacksDir, "aws", "aws", "digests", "aws-s3.md")); err != nil {
		t.Errorf("digest not copied beside its binding: %v", err)
	}
	st, err := os.Stat(filepath.Join(tg.PacksDir, "aws", "shell", "rm.sh"))
	if err != nil || st.Mode().Perm()&0o100 == 0 {
		t.Errorf("exec-script must be installed executable: %v %v", st, err)
	}
}

func TestPickAndSingleBinding(t *testing.T) {
	_, hash := goodHash(t)
	c := testCatalog(t, hash)
	if _, b, _, err := Resolve(c, "writing", map[string]string{"prose": "plain"}, nil); err != nil || len(b) != 1 || b[0] != "writing/plain-pr.md" {
		t.Errorf("--pick prose=plain should select plain-pr, got %v %v", b, err)
	}
	if _, _, _, err := Resolve(c, "writing", map[string]string{"prose": "nope"}, nil); err == nil {
		t.Error("an unknown option must fail")
	}
	name, b, _, err := Resolve(c, "aws/s3", nil, nil)
	if err != nil || name != "aws/s3" || len(b) != 1 {
		t.Errorf("a single binding should resolve, got %q %v %v", name, b, err)
	}
	if _, _, _, err := Resolve(c, "nothing", nil, nil); err == nil {
		t.Error("an unknown name must fail")
	}
}

func TestRemoveKeepsSkillsOtherPacksNeed(t *testing.T) {
	repo, hash := goodHash(t)
	fakeFetch(t, map[string]string{"github.com/example/humanizer": repo})
	c := testCatalog(t, hash)
	tg := ProjectTarget(t.TempDir())
	add(t, c, "writing", nil, tg)
	add(t, c, "writing/humanizer-pr", nil, tg) // a second install that also needs humanizer

	if removed, err := Remove("writing", tg); err != nil || len(removed) != 0 {
		t.Fatalf("humanizer is still needed, removed %v (%v)", removed, err)
	}
	if removed, err := Remove("writing/humanizer-pr", tg); err != nil || len(removed) != 1 {
		t.Fatalf("the last pack needing humanizer should remove it, got %v (%v)", removed, err)
	}
	if _, err := os.Stat(filepath.Join(tg.SkillsDir, "humanizer")); err == nil {
		t.Error("humanizer should be gone")
	}
	if _, err := Remove("writing", tg); err == nil {
		t.Error("removing a pack that is not installed must fail")
	}
}

func TestReAddReplacesPack(t *testing.T) {
	repo, hash := goodHash(t)
	fakeFetch(t, map[string]string{"github.com/example/humanizer": repo})
	c := testCatalog(t, hash)
	tg := ProjectTarget(t.TempDir())
	add(t, c, "writing", nil, tg)
	add(t, c, "writing", map[string]string{"prose": "plain"}, tg)
	if _, err := os.Stat(filepath.Join(tg.PacksDir, "writing", "writing", "humanizer-pr.md")); err == nil {
		t.Error("re-adding with another pick should drop the old option's binding")
	}
	if _, err := os.Stat(filepath.Join(tg.PacksDir, "writing", "writing", "plain-pr.md")); err != nil {
		t.Errorf("new pick missing: %v", err)
	}
}

func TestPackAndSameNamedCategoryDontCollide(t *testing.T) {
	repo, hash := goodHash(t)
	fakeFetch(t, map[string]string{"github.com/example/humanizer": repo})
	c := testCatalog(t, hash)
	tg := ProjectTarget(t.TempDir())
	add(t, c, "writing/humanizer-pr", nil, tg)
	add(t, c, "writing", nil, tg)
	if _, err := os.Stat(filepath.Join(tg.PacksDir, "_bindings", "writing", "humanizer-pr", "writing", "humanizer-pr.md")); err != nil {
		t.Errorf("the single binding must survive adding the pack named like its category: %v", err)
	}
	if _, err := Remove("writing", tg); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(tg.PacksDir, "_bindings", "writing", "humanizer-pr")); err != nil {
		t.Errorf("removing the pack must not remove the single binding: %v", err)
	}
}

func TestEditedSkillIsKept(t *testing.T) {
	repo, hash := goodHash(t)
	fakeFetch(t, map[string]string{"github.com/example/humanizer": repo})
	c := testCatalog(t, hash)
	tg := ProjectTarget(t.TempDir())
	add(t, c, "writing", nil, tg)
	skill := filepath.Join(tg.SkillsDir, "humanizer", "SKILL.md")
	if err := os.WriteFile(skill, []byte("my tweaks\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if res := add(t, c, "writing", nil, tg); len(res.Kept) != 1 {
		t.Errorf("an edited paws skill should be kept on re-add, got %+v", res)
	}
	if _, err := Remove("writing", tg); err != nil {
		t.Fatal(err)
	}
	if data, _ := os.ReadFile(skill); string(data) != "my tweaks\n" {
		t.Error("remove deleted a skill the user edited")
	}
}

func TestRePickRemovesOrphanedSkill(t *testing.T) {
	repo, hash := goodHash(t)
	fakeFetch(t, map[string]string{"github.com/example/humanizer": repo})
	c := testCatalog(t, hash)
	tg := ProjectTarget(t.TempDir())
	add(t, c, "writing", nil, tg)
	add(t, c, "writing", map[string]string{"prose": "plain"}, tg)
	if _, err := os.Stat(filepath.Join(tg.SkillsDir, "humanizer")); err == nil {
		t.Error("switching the pick away from humanizer should remove the skill nothing needs")
	}
}

func TestUnknownPickGroupFails(t *testing.T) {
	_, hash := goodHash(t)
	c := testCatalog(t, hash)
	if _, _, _, err := Resolve(c, "writing", map[string]string{"proze": "plain"}, nil); err == nil {
		t.Error("a misspelled pick group must fail")
	}
}

func TestUpstreamExecBitKept(t *testing.T) {
	repo, hash := goodHash(t)
	fakeFetch(t, map[string]string{"github.com/example/humanizer": repo})
	c := testCatalog(t, hash)
	tg := ProjectTarget(t.TempDir())
	add(t, c, "writing", nil, tg)
	st, err := os.Stat(filepath.Join(tg.SkillsDir, "humanizer", "scripts", "run.sh"))
	if err != nil || st.Mode().Perm()&0o100 == 0 {
		t.Errorf("upstream script should stay executable: %v %v", st, err)
	}
}

func TestFetchRefusesUnsafeInputs(t *testing.T) {
	sha := "0123456789abcdef0123456789abcdef01234567"
	for _, c := range [][2]string{
		{"github.com/a/b", "--upload-pack=touch /tmp/x"},
		{"github.com/a/b", "main"},
		{"/tmp/evil", sha},
		{"file:///tmp/evil", sha},
		{"ext::sh -c touch% /tmp/x", sha},
	} {
		if err := Fetch(c[0], c[1], t.TempDir()); err == nil || !strings.Contains(err.Error(), "refusing to fetch") {
			t.Errorf("Fetch(%q, %q) should refuse, got %v", c[0], c[1], err)
		}
	}
}
