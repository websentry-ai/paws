package cli

import (
	"strings"
	"testing"
)

func TestSetContentHashTouchesOnlyUpstreamBlock(t *testing.T) {
	in := "---\nname: x\nupstream:\n  repo: github.com/a/b\n  path: SKILL.md\n  ref: 0123\nprovenance:\n  ref: not-this-one\ntags: [a]\n---\nBody mentions ref: and content_hash: too.\n"
	out, ok := setContentHash(in, "sha256:abc")
	if !ok || strings.Count(out, "content_hash: sha256:abc") != 1 || !strings.Contains(out, "  ref: 0123\n  content_hash: sha256:abc\nprovenance:") {
		t.Fatalf("insert went wrong:\n%s", out)
	}
	again, ok := setContentHash(out, "sha256:def")
	if !ok || strings.Contains(again, "sha256:abc") || strings.Count(again, "content_hash: sha256:def") != 1 || !strings.Contains(again, "Body mentions ref: and content_hash: too.") {
		t.Fatalf("replace went wrong:\n%s", again)
	}
	if _, ok := setContentHash("---\nname: x\n---\n", "h"); ok {
		t.Error("no upstream block: nothing to pin")
	}
}

func TestParsePackFlagsEqualsForm(t *testing.T) {
	f, err := parsePackFlags([]string{"--pick=prose=plain", "--catalog=/tmp/c", "writing"}, "--pick", "--catalog")
	if err != nil || f.picks["prose"] != "plain" || f.catalog != "/tmp/c" || len(f.args) != 1 {
		t.Errorf("got %+v, %v", f, err)
	}
}
