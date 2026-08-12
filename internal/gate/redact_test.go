package gate

import (
	"strings"
	"testing"
)

func TestRedact(t *testing.T) {
	cases := []struct {
		name     string
		in       string
		wantGone string // substring that must not survive
		wantKept string // substring that must survive (readability)
	}{
		{"password flag equals", `psql prod -c "select 1" --password=hunter2`, "hunter2", "--password="},
		{"password flag space", `mysql -u root --password hunter2`, "hunter2", "--password"},
		// bare "-p" is too ambiguous across tools (port, parallelism, print, ...) to redact
		// on sight — flagging it would bury real signal in false-positive noise elsewhere.
		{"bare -p is intentionally not covered", `curl -p s3cr3t https://api.example.com`, "", "s3cr3t"},
		{"env var assignment", `AWS_SECRET_ACCESS_KEY=abc123XYZ aws s3 ls`, "abc123XYZ", "AWS_SECRET_ACCESS_KEY="},
		{"connection string", `psql "postgres://prod-user:hunter2@prod-db.internal:5432/app"`, "hunter2", "prod-user"},
		{"stripe live key", `curl -H "Authorization: Bearer sk_live_51H8x9aBcDeFgHiJ"`, "sk_live_51H8x9aBcDeFgHiJ", "Bearer"},
		{"github token", `git remote set-url origin https://ghp_abcdef0123456789@github.com/org/repo.git`, "ghp_abcdef0123456789", "github.com"},
		{"aws access key id", `echo AKIAABCDEFGHIJKLMNOP`, "AKIAABCDEFGHIJKLMNOP", ""},
		{"no secret, unchanged", `git push origin main`, "", "git push origin main"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := redact(c.in)
			if c.wantGone != "" && strings.Contains(got, c.wantGone) {
				t.Errorf("redact(%q) = %q, secret %q still present", c.in, got, c.wantGone)
			}
			if c.wantKept != "" && !strings.Contains(got, c.wantKept) {
				t.Errorf("redact(%q) = %q, expected %q to survive for readability", c.in, got, c.wantKept)
			}
		})
	}
}
