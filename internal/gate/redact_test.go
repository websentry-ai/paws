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
		{"authorization header", `curl -H "Authorization: Basic dXNlcjpwYXNz" https://x`, "dXNlcjpwYXNz", "Authorization: Basic "},
		{"authorization header, no scheme", `curl -H "Authorization: tok_abc123" -X POST https://x`, "tok_abc123", "-X POST"},
		{"bearer token", `curl -H "Authorization: Bearer abc.def-123" https://x`, "abc.def-123", "Bearer"},
		{"bearer outside a header", `http GET api.example.com bearer zzTopSecret99`, "zzTopSecret99", "bearer"},
		{"x-api-key header", `curl -H 'x-api-key: k-12345' https://x`, "k-12345", "x-api-key: "},
		{"anthropic key", `export KEY=x; echo sk-ant-api03-abcdefghijklmnop`, "sk-ant-api03-abcdefghijklmnop", "echo"},
		{"openai project key", `echo sk-proj-ABCDEFGHIJKLMNOPQR`, "sk-proj-ABCDEFGHIJKLMNOPQR", "echo"},
		{"github fine-grained pat", `gh auth login --with-token github_pat_11ABCDEF_xyz`, "github_pat_11ABCDEF_xyz", "gh auth login"},
		{"gitlab pat", `glab auth login -t glpat-abcdefghij0123456789`, "glpat-abcdefghij0123456789", "glab auth login"},
		{"bare jwt", `echo eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxIn0.c2lnbmF0dXJl | jq`, "eyJzdWIiOiIxIn0", "| jq"},
		{"mysql glued -p", `mysql -u root -phunter2 app`, "hunter2", "-u root -p"},
		{"mariadb-dump glued -p", `mariadb-dump -phunter2 app`, "hunter2", "app"},
		{"mysql bare -p prompts, unchanged", `mysql -u root -p app`, "", "-p app"},
		{"glued -p outside mysql is not covered", `rsync -pv src dst`, "", "-pv"},
		{"generic key=value", `./deploy.sh db_password=hunter2 api-key=abc123`, "hunter2", "db_password="},
		{"generic api_key", `./run api_key=abc123`, "abc123", "api_key="},
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
