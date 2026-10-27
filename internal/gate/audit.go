package gate

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/websentry-ai/paws/internal/rule"
)

// DryRun suppresses side effects (audit writes, exec-script) for `paws explain`.
var DryRun bool

func auditPath() string {
	dir := UserDir()
	if dir == "" {
		return ""
	}
	return filepath.Join(dir, "audit.log")
}

// logAudit appends a JSONL record of a fired rule. Best-effort, never blocks.
func logAudit(ev Event, r *rule.Rule, oc outcome) {
	if DryRun {
		return
	}
	path := auditPath()
	if path == "" {
		return
	}
	rec := map[string]any{
		"ts":      time.Now().UTC().Format(time.RFC3339),
		"agent":   ev.Agent,
		"event":   r.Event,
		"tool":    ev.Tool,
		"target":  redact(ev.Target),
		"rule":    r.Name,
		"action":  r.Action.Type,
		"verdict": verdictName(oc.verdict),
		"source":  r.SourcePath,
		"session": ev.Session,
	}
	if oc.satisfiedBy != "" {
		rec["satisfied_by"] = oc.satisfiedBy
	}
	if oc.digestErr != "" {
		rec["digest_error"] = oc.digestErr
	}
	if oc.context != "" {
		rec["context_added"] = true // session-start require-skill and advise: allowed, but the agent got text
	}
	if ev.AgentID != "" {
		rec["agent_id"] = ev.AgentID
	}
	if ev.AgentType != "" {
		rec["agent_type"] = ev.AgentType
	}
	appendAudit(path, rec)
}

// AuditSchema versions the record shape: v2 adds turn_end, tool_result and skill_invoked.
const AuditSchema = 2

// maxAuditBytes rotates the log: past it, audit.log becomes audit.log.1 (replacing the old one).
var maxAuditBytes int64 = 50 << 20

// LogTelemetry appends a count-only record (turn_end, tool_result, skill_invoked). It never
// carries command, prompt or file text. Best-effort, never blocks.
func LogTelemetry(event string, ev Event, fields map[string]any) {
	if DryRun {
		return
	}
	path := auditPath()
	if path == "" {
		return
	}
	rec := map[string]any{"ts": time.Now().UTC().Format(time.RFC3339), "agent": ev.Agent, "event": event, "session": ev.Session}
	if ev.AgentID != "" {
		rec["agent_id"] = ev.AgentID
	}
	for k, v := range fields {
		rec[k] = v
	}
	appendAudit(path, rec)
}

func appendAudit(path string, rec map[string]any) {
	rec["v"] = AuditSchema
	line, err := json.Marshal(rec)
	// 0600/0700: the log holds raw matched command/prompt text, which can carry secrets.
	if err != nil || os.MkdirAll(filepath.Dir(path), 0o700) != nil {
		return
	}
	if st, err := os.Stat(path); err == nil && st.Size() > maxAuditBytes {
		os.Rename(path, path+".1")
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	defer f.Close()
	f.Chmod(0o600) // O_CREATE only sets mode on first creation, not on an existing looser file.
	f.Write(append(line, '\n'))
}

// AuditPath is where the audit log lives ("" if no user dir resolves).
func AuditPath() string { return auditPath() }

func verdictName(v Verdict) string {
	switch v {
	case Deny, DenyRetry:
		return "deny"
	case Ask:
		return "ask"
	default:
		return "allow"
	}
}
