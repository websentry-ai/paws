# Local telemetry

paws keeps one JSONL audit log per user: `~/.paws/audit.log` (or
`$PAWS_HOME/audit.log`). Nothing in it leaves the machine until you run
`paws login`; after that, part of it is uploaded ([below](#portal-upload)).
Files are 0600 in a 0700 directory. Past 50 MB the log moves to
`audit.log.1`, replacing the previous one.

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

**Action records** (`event: action`, one per PostToolUse, beside its
`tool_result`): `tool`, `action` (a normalized key, below), and when they
apply `command` (Bash: the command with secrets redacted, cut to 500 bytes),
`file` (Read, Write, Edit, NotebookEdit: the path relative to the repo root,
or to the working directory outside a repo; left out when the file is outside
that root, so it is never absolute) and `mcp` (`server` and `tool` as Claude
Code names them, `class`: `send`, `create`, `update` or `delete`, left out for
reads). Never file contents, MCP input, URLs, search patterns or tool output.

The `action` key:

| Tool | Key | Examples |
|---|---|---|
| Bash, SQL | `DELETE`, `DROP`, `TRUNCATE` or `UPDATE` when the command holds `delete from`, `drop table/database/schema/index/view`, `truncate table` (or upper-case `TRUNCATE`), `update <t> set` | `psql -c "DELETE FROM t"` is `DELETE` |
| Bash | the first command of a `&&`, `\|\|`, `;`, `\|` chain that isn't `cd`, without env assignments and `sudo`: its program name, plus the subcommand for common CLIs (git, npm, go, docker, kubectl, terraform, aws, gcloud, make and others; `-C dir` is skipped) and two words for `gh` and `glab` | `git commit`, `gh pr create`, `aws s3`, `npm test`, `ls` |
| Read | `read <top-level dir>/**`, `read <file>` at the repo root, `read` outside it | `read src/**`, `read README.md` |
| Write, Edit, NotebookEdit | the same with `edit` | `edit migrations/**` |
| `mcp__<server>__<tool>` | `<server> <tool>` in lower case, `claude_ai_` and `plugin_<name>_` dropped from the server, the server's name dropped from the tool, `_` as `-` | `gmail create-draft`, `notion create-pages`, `slack send-message` |
| anything else | the tool name in lower case | `grep`, `webfetch` |

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

## Portal upload

Opt-in. Nothing is sent until you run `paws login`, which stores a token in
`~/.paws/credentials.json` (0600; `paws logout` deletes it). From then on:

- the async **Stop hook** (after each Claude turn) uploads new records when
  there are some and the last upload was more than `PAWS_SYNC_INTERVAL` ago
  (default `10m`), and checks the portal's config hash when the last check
  was more than `PAWS_CONFIG_INTERVAL` ago (default `15m`). Each request times
  out after 5 seconds; any failure is dropped silently and retried later;
- **`paws sync`** does both at once and prints what it did;
- **`paws login`** syncs once, so the portal sees the machine right away.

The PreToolUse and SessionStart hooks, which decide tool calls, never touch
the network.

`paws sync` reads `audit.log.1` then `audit.log` and uploads each rule
record (verdict `allow` or `deny`) and action record once.
`~/.paws/sync-state.json` remembers the last record sent; requests carry at
most 1,000 records and 1 MB. `paws sync --dry-run` prints the request body
(first 20 records) and sends nothing. The body:

```
{ "records": [ <rule record> | <action record> ],
  "applied_config_hash": "<hash of the portal config last applied, or null>",
  "inventory": { "skills": ["humanizer", ...],
                 "rules": [{"name": ..., "skill": ..., "actions": ["gh pr create"]}] } }
```

**Rule record:**

| Field | Value |
|---|---|
| `kind` | `rule` |
| `ts` | the record's timestamp |
| `event` | the rule's event (`pre-tool`, `pre-commit`, `pre-push`, `session-start`) |
| `skill` | the skill the rule requires |
| `action_label` | rules from the portal only: the action's label |
| `matcher` | catalog rules (under `~/.paws/packs` or `.paws/packs`) and portal rules: the matcher, e.g. `tool=Bash command_regex=...`. Left out for your own rules |
| `command` | Bash only: the matched command, secrets redacted, cut to 500 bytes. A path or MCP input is never sent here |
| `tool` | the tool name |
| `verdict` | `allow` or `deny` |
| `already_loaded` | `true` when allowed via `skill-load` in a session where this rule had not denied first |
| `satisfied_by` | `skill-load` or `digest-delivered`, when set |

**Action record:** `kind: action`, `ts`, `tool`, `action`, and `command`,
`file`, `mcp` as in the audit log (above).

**Inventory:** the directory names in `~/.claude/skills` and the project's
`.claude/skills`, and every paws rule in scope with its skill and a display
form of what it runs before (a portal rule's label; otherwise its matcher,
shortened, e.g. `gh pr create`, `edit docs/**`, `each new session`).

Nothing else is sent: no prompts, replies, file contents, tool output,
absolute paths, rule file paths, session or agent ids. Telemetry records
(`skill_invoked`, `tool_result`, `turn_end`) are not uploaded.

## Portal settings

Settings come down; the gate stays offline. When the config hash changes
(from `GET /config/hash`, or from an upload's response), paws fetches the
config and applies it:

- one rule file per skill and action under `~/.paws/portal/`
  (`<skill>--<action id>.md`), a plain `require-skill` rule that paws loads
  like any other. Files the config no longer has, including turned-off
  skills, are removed. Every write is atomic;
- a `load: rule` skill becomes `on_miss: digest` with its rule text as the
  digest (`~/.paws/portal/digests/<skill>.md`); no SKILL.md is installed;
- a catalog skill installs into `~/.claude/skills/<name>`: from upstream at
  its pinned commit, or from the catalog in the paws binary, and only when its
  content hash matches the config. A mismatch is skipped and reported. A skill
  directory paws didn't install, or one you edited, is left as it is, as with
  `paws add`. Turning a skill off removes its rules, not its skill directory;
- the applied hash, the applied skills and the last upload and check times
  are kept in `~/.paws/portal/state.json`. A skill that failed to download
  leaves the hash unrecorded, so the next check tries again.
