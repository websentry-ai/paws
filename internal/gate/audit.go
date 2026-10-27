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
	if ev.AgentID != "" {
		rec["agent_id"] = ev.AgentID
	}
	if ev.AgentType != "" {
		rec["agent_type"] = ev.AgentType
	}
	line, err := json.Marshal(rec)
	// 0600/0700: the log holds raw matched command/prompt text, which can carry secrets.
	if err != nil || os.MkdirAll(filepath.Dir(path), 0o700) != nil {
		return
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	defer f.Close()
	f.Chmod(0o600) // O_CREATE only sets mode on first creation, not on an existing looser file.
	f.Write(append(line, '\n'))
}

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
