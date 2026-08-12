package gate

import "regexp"

// redactPatterns catch common shapes credentials take in a shell command or prompt: flag/env-var
// assignments whose name signals a secret, connection-string passwords, and known provider token
// prefixes. Heuristic, not a guarantee — a secret in an unrecognized shape still gets logged raw.
var redactPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(--?(?:password|passwd|pwd|token|secret|api[-_]?key|access[-_]?key)[= ])\S+`),
	regexp.MustCompile(`(?i)\b([A-Z_]*(?:PASSWORD|SECRET|TOKEN|API_KEY|ACCESS_KEY)[A-Z_]*=)\S+`),
	regexp.MustCompile(`(://[^:/@\s]+:)[^@\s]+(@)`),
	regexp.MustCompile(`\b(?:sk_live_|sk_test_|ghp_|gho_|ghs_|xox[baprs]-|AKIA)[A-Za-z0-9_\-]{8,}`),
}

// redact masks values matched by redactPatterns, keeping the flag/key name visible so the log
// stays readable — only the value is replaced.
func redact(s string) string {
	s = redactPatterns[0].ReplaceAllString(s, "${1}***")
	s = redactPatterns[1].ReplaceAllString(s, "${1}***")
	s = redactPatterns[2].ReplaceAllString(s, "${1}***${2}")
	s = redactPatterns[3].ReplaceAllString(s, "***")
	return s
}
