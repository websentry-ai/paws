# Local telemetry

paws keeps one JSONL audit log per user: `~/.paws/audit.log` (or
`$PAWS_HOME/audit.log`). It never leaves the machine; `paws sync` uploads
a few fields of it only after `paws login` (below). Files are 0600 in a
0700 directory. Past 50 MB the log moves to `audit.log.1`, replacing the
previous one.

## Records (schema v2)

Every record has `v` (2), `ts`, `agent`, `event` and `session`, plus
`agent_id` when a subagent made the call.

**Rule records** (a rule matched): `rule`, `action`, `skill` (the skill
the rule requires), `verdict` (`allow` or `deny`), `tool`, `target` (the
matched command or path, with secrets redacted), `source` (the rule file),
and when relevant:

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

## Where to see it

`paws list` shows how often each installed binding fired. The Activity tab of
`paws ui` shows the rest. Per rule: `fires` (all records), `denied`, `already-loaded` (passed via
skill-load in a session where the rule had not denied first: the agent had
the skill on its own), `loaded-after-deny`, and `delivered` (digest or
session context). It also lists rules in scope that never fired, skill
invocations, and session, turn and successful tool-call totals.

The PostToolUse and Stop hooks are registered `async`: Claude Code does not
wait for them. Matched targets are capped at 2,000 bytes per record.

## Portal upload (`paws sync`)

Opt-in. Nothing is sent until you run `paws login` and then `paws sync`;
the hook never calls the network. `paws login` stores a token in
`~/.paws/credentials.json` (0600). `paws sync` reads `audit.log.1` then
`audit.log` and uploads each rule record with verdict `allow` or `deny`
once, as:

| Field | Value |
|---|---|
| `ts` | the record's timestamp |
| `event` | the rule's event (`pre-tool`, `pre-commit`, `pre-push`, `session-start`) |
| `binding` | the rule's `name` |
| `pattern` | catalog bindings only (rule file under `~/.paws/packs` or `.paws/packs`): the matcher, e.g. `tool=Bash command_regex=...`. Omitted for your own rules |
| `skill` | the skill the rule requires |
| `verdict` | `allow` or `deny` |
| `already_loaded` | `true` when allowed via `skill-load` in a session where this rule had not denied first |
| `satisfied_by` | `skill-load` or `digest-delivered`, when set |

Nothing else is sent: no `target` (command or path text), tool name or
input, session or agent ids, or file paths. Telemetry records
(`skill_invoked`, `tool_result`, `turn_end`) are not uploaded.
`paws sync --dry-run` prints the request body (first 20 records) and sends
nothing. `~/.paws/sync-state.json` remembers the last record sent, so a
record is uploaded once; requests carry at most 1,000 records and 1 MB.
`paws logout` deletes the token.
