package gate

import "regexp"

// redactPatterns catch common shapes credentials take in a shell command or prompt: flag/env-var
// assignments whose name signals a secret, connection-string passwords, auth headers, JWTs and
// known provider token prefixes. Heuristic, not a guarantee — a secret in an unrecognized shape
// still gets logged raw. Where a pattern has a group, the name before the value stays visible.
var redactPatterns = []struct {
	re   *regexp.Regexp
	repl string
}{
	{regexp.MustCompile(`(?i)(--?(?:password|passwd|pwd|token|secret|api[-_]?key|access[-_]?key)[= ])\S+`), "${1}***"},
	{regexp.MustCompile(`(?i)\b([A-Z_]*(?:PASSWORD|SECRET|TOKEN|API_KEY|ACCESS_KEY)[A-Z_]*=)\S+`), "${1}***"},
	{regexp.MustCompile(`(?i)((?:password|passwd|secret|token|api[_-]?key)=)\S+`), "${1}***"},
	{regexp.MustCompile(`(://[^:/@\s]+:)[^@\s]+(@)`), "${1}***${2}"},
	// "Authorization: Bearer x" keeps the scheme; "Authorization: x" (no scheme) loses x.
	{regexp.MustCompile(`(?i)(authorization:\s*(?:\w+\s+)?)[^\s"']+`), "${1}***"},
	{regexp.MustCompile(`(?i)\b(bearer\s+)[A-Za-z0-9._~+/=-]+`), "${1}***"},
	{regexp.MustCompile(`(?i)(x-api-key:\s*)[^\s"']+`), "${1}***"},
	{regexp.MustCompile(`\b(?:sk_live_|sk_test_|ghp_|gho_|ghs_|xox[baprs]-|AKIA)[A-Za-z0-9_\-]{8,}`), "***"},
	{regexp.MustCompile(`\bsk-[A-Za-z0-9_-]{16,}`), "***"}, // OpenAI, Anthropic (sk-ant-), sk-proj-
	{regexp.MustCompile(`\bgithub_pat_\w+`), "***"},
	{regexp.MustCompile(`\bglpat-[\w-]{16,}`), "***"},
	{regexp.MustCompile(`\beyJ[\w-]+\.[\w-]+\.[\w-]+`), "***"}, // a bare JWT
}

// mysql and mariadb take the password glued to -p (-phunter2). Elsewhere -p means a port,
// parallelism or print, so it is only redacted when the command is one of these clients.
var (
	mysqlCommand  = regexp.MustCompile(`^\s*(?:mysql|mariadb)`)
	mysqlPassword = regexp.MustCompile(`(\s-p)\S+`)
)

// redact masks values matched by redactPatterns, keeping the flag/key name visible so the log
// stays readable — only the value is replaced.
func redact(s string) string {
	for _, p := range redactPatterns {
		s = p.re.ReplaceAllString(s, p.repl)
	}
	if mysqlCommand.MatchString(s) {
		s = mysqlPassword.ReplaceAllString(s, "${1}***")
	}
	return s
}
