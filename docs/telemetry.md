# Local telemetry

paws keeps one JSONL audit log per user: `~/.paws/audit.log` (or
`$PAWS_HOME/audit.log`). It never leaves the machine. Files are 0600 in a
0700 directory. Past 50 MB the log moves to `audit.log.1`, replacing the
previous one.

## Records (schema v2)

Every record has `v` (2), `ts`, `agent`, `event` and `session`, plus
`agent_id` when a subagent made the call.

**Rule records** (a rule matched): `rule`, `action`, `verdict`
(`allow` or `deny`), `tool`, `target` (the matched command or path, with
secrets redacted), `source` (the rule file), and when relevant:

| Field | Meaning |
|---|---|
| `satisfied_by: skill-load` | a require-skill gate passed: the skill was invoked in the window |
| `satisfied_by: digest-delivered` | passed: its digest already rode in a deny this window |
| `context_added: true` | allowed, and the agent received text (session-start) |
| `digest_error` | an `on_miss: digest` file was unusable; the deny fell back to load-required |

**Telemetry records** carry no command or file text:

| `event` | Written on | Extra fields |
|---|---|---|
| `skill_invoked` | PreToolUse on the Skill tool | `skill` |
| `tool_result` | PostToolUse: a tool call that ran and succeeded (denied and failed calls write none) | `tool` |
| `turn_end` | Stop (the agent finished its turn) | — |

## paws stats

```
paws stats          # table
paws stats --json   # for the local UI and scripts
```

Per rule: `fires` (all records), `denied`, `already-loaded` (passed via
skill-load in a session where the rule had not denied first: the agent had
the skill on its own), `loaded-after-deny`, and `delivered` (digest or
session context). It also lists rules in scope that never fired, skill
invocations, and session, turn and successful tool-call totals.

The PostToolUse and Stop hooks are registered `async`: Claude Code does not
wait for them. Matched targets are capped at 2,000 bytes per record.
