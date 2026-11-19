package history

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

type fakeSource struct {
	calls    []Call
	sessions int
}

func (fakeSource) Name() string { return "fake" }

func (s fakeSource) Calls(since, until time.Time, fn func(Call)) (int, error) {
	for _, c := range s.calls {
		fn(c)
	}
	return s.sessions, nil
}

func at(day int) time.Time { return time.Date(2026, 11, day, 12, 0, 0, 0, time.UTC) }

func TestSummarizeCountsEachCallOnce(t *testing.T) {
	src := fakeSource{sessions: 3, calls: []Call{
		{ID: "a", Tool: "Bash", Target: "rm -rf build", At: at(2)},
		{ID: "a", Tool: "Bash", Target: "rm -rf build", At: at(2)}, // the same call in a forked session's log
		{ID: "b", Tool: "Bash", Target: "rm -rf build", At: at(5)},
		{ID: "c", Tool: "Bash", Target: "git commit -m x", At: at(3)},
		{ID: "d", Tool: "Bash", Target: "ls", At: at(4)},
		{ID: "e", Tool: "Skill", Target: "coreutils", At: at(4)},
		{ID: "f", Tool: "Skill", Target: "my-private-skill", At: at(4)},
	}}
	triggers := []Trigger{
		{Skill: "coreutils", Action: "rm", Matches: func(tool, target string) bool { return strings.HasPrefix(target, "rm ") }},
		{Skill: "coreutils", Action: "recursive", Matches: func(tool, target string) bool { return strings.Contains(target, "-rf") }},
		{Skill: "git", Action: "git", Matches: func(tool, target string) bool { return strings.HasPrefix(target, "git ") }},
	}
	sum, err := Summarize(src, at(1), at(19), 30, triggers, map[string]bool{"coreutils": true, "git": true})
	if err != nil {
		t.Fatal(err)
	}
	if sum.Source != "fake" || sum.WindowDays != 30 || sum.Since != "2026-11-01" || sum.Until != "2026-11-19" || sum.Sessions != 3 || sum.ToolCalls != 4 {
		t.Errorf("header %+v", sum)
	}
	got, _ := json.Marshal(sum.Skills)
	want := `[{"skill":"coreutils","calls":2,"last_seen":"2026-11-05","actions":[{"id":"recursive","calls":2},{"id":"rm","calls":2}]},` +
		`{"skill":"git","calls":1,"last_seen":"2026-11-03","actions":[{"id":"git","calls":1}]}]`
	if string(got) != want {
		t.Errorf("skills\n got %s\nwant %s", got, want)
	}
	if len(sum.Loaded) != 1 || sum.Loaded[0] != (Count{"coreutils", 1}) {
		t.Errorf("loaded %v: only catalog skill names are sent", sum.Loaded)
	}
	if len(sum.Unmatched) != 1 || sum.Unmatched[0] != (Count{"bash ls", 1}) {
		t.Errorf("unmatched %v", sum.Unmatched)
	}
}

func TestSummarizeEmpty(t *testing.T) {
	sum, err := Summarize(fakeSource{}, at(1), at(2), 30, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(sum)
	if !strings.Contains(string(b), `"skills":[]`) || !strings.Contains(string(b), `"loaded":[]`) || !strings.Contains(string(b), `"unmatched":[]`) {
		t.Errorf("empty lists must marshal as [], got %s", b)
	}
}

func TestSignatureSendsNoArgumentsOrPaths(t *testing.T) {
	cases := map[[2]string]string{
		{"Bash", "rm -rf /Users/alice/secret-project/build"}:            "bash rm",
		{"Bash", "cd /Users/alice/acme && git push origin main"}:        "bash git push",
		{"Bash", "AWS_SECRET_ACCESS_KEY=abc aws s3api create-bucket"}:   "bash aws s3api create-bucket",
		{"Bash", "sudo kubectl delete pod web-1"}:                       "bash kubectl delete",
		{"Bash", "/Users/alice/bin/deploy-acme.sh --prod"}:              "bash other",
		{"Bash", "./scripts/release-customer-x"}:                        "bash other",
		{"Bash", "acme-internal-cli rotate"}:                            "bash other",
		{"Bash", "make deploy-acme-prod"}:                               "bash make",
		{"Bash", "npm run build:customer-x"}:                            "bash npm",
		{"Bash", "npx @acme/private-tool"}:                              "bash npx",
		{"Bash", "/usr/bin/rm -f x"}:                                    "bash rm",
		{"Bash", `psql -c "DROP TABLE customers"`}:                      "bash DROP",
		{"Bash", "git -C /Users/alice/acme commit -m 'acme'"}:           "bash git",
		{"Bash", "gh pr create --title 'Fix acme login'"}:               "bash gh pr create",
		{"Edit", "/Users/alice/acme/main.tf"}:                           "edit .tf",
		{"Write", "/Users/alice/acme/Dockerfile"}:                       "edit",
		{"Read", "/Users/alice/acme/.env.acme-prod"}:                    "read",
		{"mcp__claude_ai_Notion__notion-create-pages", `{"title":"x"}`}: "mcp claude_ai_notion notion-create-pages",
		{"mcp__bad server__x", "{}"}:                                    "mcp other",
		{"WebFetch", "https://internal.acme.com/x"}:                     "webfetch",
		{"Agent", "summarize acme's Q3 numbers"}:                        "agent",
	}
	for in, want := range cases {
		got := Signature(in[0], in[1])
		if got != want {
			t.Errorf("Signature(%q, %q) = %q, want %q", in[0], in[1], got, want)
		}
		for _, leak := range []string{"alice", "acme", "customer", "secret", "abc", "/"} {
			if strings.Contains(got, leak) {
				t.Errorf("Signature(%q, %q) = %q leaks %q", in[0], in[1], got, leak)
			}
		}
	}
}
