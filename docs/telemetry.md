# Local telemetry

paws keeps one JSONL audit log per user: `~/.paws/audit.log` (or
`$PAWS_HOME/audit.log`). Nothing in it leaves the machine until you run
`unbound-paws login`; after that, part of it is uploaded ([below](#portal-upload)).
Files are 0600 in a 0700 directory. Past 50 MB the log moves to
`audit.log.1`, replacing the previous one.

## Records (schema v2)

Every record has `v` (2), `ts`, `agent`, `event` and `session`, plus
`agent_id` when a subagent made the call, `harness` (the harness that ran the hook: `claude-code` or `codex`; `agent`
holds the same value, for older readers) and `model` when known (on Claude Code, the model of the session's last
assistant message, so one response behind right after `/model`; on Codex, the model the hook names).

On Codex, paws also keeps a ledger per session thread, because Codex's own session files are not a stable format:
`$PAWS_HOME/sessions/codex/<session>.jsonl` (`<session>.<agent>.jsonl` for a subagent), 0600. It holds each tool
call with paws's verdict and its input (secrets masked as in the audit log), whether it succeeded, a marker per
turn (never the prompt) and per compaction. The gate reads it for skill loads, delivered digests, `when:` conditions
and outcome grading. It never leaves the machine, and a ledger nobody wrote for 7 days is deleted. Like Claude Code's
transcript, it is a file the agent's own user can write: paws guides an agent that follows its rules, it does not
stop code running as that user from forging a load.

**Rule records** (a rule matched): `rule`, `action`, `skill` (the skill
the rule requires), `verdict` (`allow` or `deny`), `tool`, `target` (the
matched command or path, with secrets [redacted](#redaction)), `source` (the rule file),
`at` (`ts` to the nanosecond, so grading can tell a deny at a turn's start from one at the end of the turn before;
sync never sends it), and when relevant:

| Field | Meaning |
|---|---|
| `satisfied_by: skill-load` | a require-skill gate passed: the skill was invoked in the window |
| `satisfied_by: digest-delivered` | passed: its digest already rode in a deny this window |
| `context_added: true` | allowed, and the agent received text (session-start) |
| `digest_error` | an `on_miss: digest` file was unusable; the deny fell back to load-required |

**Action records** (`event: action`, one per PostToolUse, beside its
`tool_result`): `tool`, `action` (a normalized key, below), and when they
apply `command` (Bash: the command with secrets redacted, cut to 4,096 bytes),
`file` (Read, Write, Edit, NotebookEdit: the path relative to the repo root,
or to the working directory outside a repo; left out when the file is outside
that root, so it is never absolute) and `mcp` (`server` and `tool` as Claude
Code names them, `class`: `send`, `create`, `update` or `delete`, left out for
reads), plus `target_hash` (the first 8 bytes of the SHA-256 of the call's full input, in hex; `paws scan` uses it
to match transcript calls to these records, and sync never sends it). Never file contents, MCP input, URLs, search
patterns or tool output.

The `action` key:

| Tool | Key | Examples |
|---|---|---|
| Bash, SQL | `DELETE`, `DROP`, `TRUNCATE` or `UPDATE` when the command holds `delete from`, `drop table/database/schema/index/view`, `truncate table` (or upper-case `TRUNCATE`), `update <t> set` | `psql -c "DELETE FROM t"` is `DELETE` |
| Bash | the first command of a `&&`, `\|\|`, `;`, `\|` chain that isn't `cd`, without env assignments and `sudo`: its program name, plus the subcommand for common CLIs (git, npm, go, docker, kubectl, terraform, aws, gcloud, make and others; `-C dir` is skipped) and two words for `gh` and `glab` | `git commit`, `gh pr create`, `aws s3`, `npm test`, `ls` |
| Read | `read <top-level dir>/**`, `read <file>` at the repo root, `read` outside it | `read src/**`, `read README.md` |
| Write, Edit, NotebookEdit | the same with `edit` | `edit migrations/**` |
| `mcp__<server>__<tool>` | `<server> <tool>` in lower case, `claude_ai_` and `plugin_<name>_` dropped from the server, the server's name dropped from the tool, `_` as `-` | `gmail create-draft`, `notion create-pages`, `slack send-message` |
| anything else | the tool name in lower case | `grep`, `webfetch` |

**Outcome records** (`event: outcome`, written by the Stop hook, at the end of the turn): one per call a
require-skill rule denied in that turn, saying what Claude did next. paws reads the turn's calls from Claude Code's
own log (the main conversation, and subagents started after the deny) and keeps the result and the one call that answered the block: `rule`, `skill`,
`source`, `tool`, `blocked_at` (the deny's `ts`), `blocked_exact` (the deny's `at`, local only), `outcome`, `checks` (each [`expect`](rules.md#expect) check's
id and whether it passed) and `retry_command` (the retry, or the first call on another route: a Bash command redacted
and cut to 4 KB, else the tool with its repo-relative file or MCP tool; empty when Claude stopped). Nothing else of the
turn's calls.

| `outcome` | What Claude did after the deny, in the same turn |
|---|---|
| `fixed` | retried (the next call the rule matched went through), and every `expect` check passed |
| `same_retry` | retried the same call (a `cd dir &&` in front and spacing aside; an env assignment such as `AWS_PROFILE=prod` counts), and a check failed or there are none |
| `changed_failed` | retried a different call, and a check failed |
| `changed_unchecked` | retried a different call; the rule has no checks |
| `other_route` | never retried; other calls ran |
| `stopped` | never retried, and nothing else ran |

A deny with no outcome record (you interrupted, or the turn never ended) counts as unknown.

**Telemetry records** carry no command or file text:

| `event` | Written on | Extra fields |
|---|---|---|
| `skill_invoked` | PreToolUse on a skill load: the Skill tool, or a plain read that prints a skill's whole `SKILL.md` (`Read`, or a lone `cat`/`sed -n`/`head`/... of the file, or of several SKILL.md files at once: one record per skill; a part of a file is not an invocation) | `skill` |
| `tool_result` | PostToolUse: a tool call that ran and succeeded (denied and failed calls write none) | `tool` |
| `turn_end` | Stop (the agent finished its turn) | — |

## Where to see it

`unbound-paws list` shows how often each installed binding fired. The Activity tab of
`unbound-paws ui` shows the rest. Per rule: `fires` (all records), `denied`, `already-loaded` (passed via
skill-load in a session where the rule had not denied first: the agent had
the skill on its own), `loaded-after-deny`, and `delivered` (digest or
session context). It also lists rules in scope that never fired, skill
invocations, and session, turn and successful tool-call totals.

The UserPromptSubmit, PostToolUse and Stop hooks are registered `async`:
Claude Code does not wait for them. UserPromptSubmit never reads or records
the prompt; it only runs the portal round below. Matched targets are capped at 2,000 bytes per record.

## Redaction

Commands and targets are redacted before they are written to the audit log,
and again before upload. The value is replaced by `***`; the name in front of
it stays, so the log still reads:

- secret-named flags and assignments: `--password=`, `--token`, `-api-key`,
  `--access-key`, `*_SECRET=`, `*_TOKEN=`, `*PASSWORD*=`, and any
  `password=`, `passwd=`, `secret=`, `token=`, `api_key=` / `api-key=`;
- the password in a connection string (`postgres://user:***@host`);
- `Authorization:` header values (the scheme, such as `Bearer`, stays),
  `Bearer <token>` anywhere, and `x-api-key:` headers;
- provider tokens: `sk-...` (OpenAI, Anthropic `sk-ant-`, `sk-proj-`),
  `sk_live_`/`sk_test_` (Stripe), `ghp_`/`gho_`/`ghs_`/`github_pat_`
  (GitHub), `glpat-` (GitLab), `xox?-` (Slack), `AKIA...` (AWS key ids);
- bare JWTs (`eyJ...`.`...`.`...`);
- `-p<password>` when the command is `mysql` or `mariadb` (and their
  `mysqldump`/`mariadb-dump` tools). Elsewhere `-p` means too many things.

It is a heuristic: a secret in a shape not listed here is written as it is.

## Portal upload

Opt-in. Nothing is sent until you run `unbound-paws login`, which stores a token, and
the account's email when the portal sends it, in `~/.paws/credentials.json`
(0600). The portal address (`PAWS_PORTAL_API`)
must be https; plain http works only for `localhost`, `127.0.0.1` or `::1`,
and credentials saved for any other http address are refused. From then on:

- the async **UserPromptSubmit** and **Stop hooks** (when you send Claude a
  message, and after each Claude turn), when the last upload was more than
  `PAWS_SYNC_INTERVAL` ago (default `5m`), send one upload: the new records,
  or none, with the inventory. Its reply carries the portal's config hash, and
  the config is fetched only when that hash changed. Between uploads, when the
  last check was more than `PAWS_CHECK_INTERVAL` ago (default `30s`), they ask
  `GET /config/hash` alone, upload nothing, and fetch the config only when the
  hash changed. A config the prompt hook pulls applies from the next tool call
  in that turn. No paws process runs between hooks: with no prompts and no
  turns, nothing is sent. Each request times out after 5 seconds
  and the whole step, skill downloads included, after 12; any failure is
  dropped silently and retried later;
- the **SessionStart hook** starts the same round in a separate background
  process (`unbound-paws sync --background`) when a session starts, resumes or
  is cleared, at most once a minute, before the first prompt. The
  hook itself never waits on it, so a change to a tool rule applies from the
  next tool call after the round lands, and a new session-start rule from the
  next session;
- **`unbound-paws sync`** does both at once and prints what it did;
- **`unbound-paws login`** syncs once, so the portal sees the machine right away.

`unbound-paws logout` deletes the token and removes every rule and digest the portal
config wrote, and `~/.paws/portal/state.json`. A 401 from the portal (the
login was revoked) does the same, from `unbound-paws sync` or a hook round; `paws
sync` then says the login was revoked.

The PreToolUse and SessionStart hooks, which decide tool calls, never touch
the network.

`unbound-paws sync` reads `audit.log.1` then `audit.log` and uploads each rule
record (verdict `allow` or `deny`) and action record once.
`~/.paws/sync-state.json` remembers the last record sent; requests carry at
most 1,000 records and 1 MB. `unbound-paws sync --dry-run` prints the request body
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
| `command` | Bash only: the matched command, secrets redacted, cut to 4,096 bytes. A path or MCP input is never sent here |
| `tool` | the tool name |
| `verdict` | `allow` or `deny` |
| `already_loaded` | `true` when allowed via `skill-load` in a session where this rule had not denied first |
| `satisfied_by` | `skill-load` or `digest-delivered`, when set |
| `harness`, `model` | as in the audit log, when the record has them; a record without `harness` is from Claude Code |

**Action record:** `kind: action`, `ts`, `tool`, `action`, and `command`,
`file`, `mcp`, `harness`, `model` as in the audit log (above).

**Outcome record:** `kind: outcome`, `ts`, `blocked_at`, `skill`, `action_id` (catalog and portal rules: the
trigger's id), `tool`, `outcome`, `checks` and `retry_command` (the call Claude ran after the block, whether it retried or went
another route; a Bash command, redacted and cut to 4 KB, else the tool with its repo-relative file or MCP tool; empty
when Claude stopped), `harness` and `model`, as in the audit log (above).

**Inventory:** the directory names in every harness's skill folders, whether or not its hook is installed:
`~/.claude/skills` and the project's `.claude/skills` (Claude Code), `~/.agents/skills`, the project's
`.agents/skills` and `$CODEX_HOME/skills` (Codex), and every paws rule in scope with its skill and a display
form of what it runs before (a portal rule's label; otherwise its matcher,
shortened, e.g. `gh pr create`, `edit docs/**`, `each new session`).

Nothing else is sent: no prompts, replies, file contents, tool output,
absolute paths, rule file paths, session or agent ids. Telemetry records
(`skill_invoked`, `tool_result`, `turn_end`) are not uploaded.

## History scan

Opt-in, and only when you ask: `unbound-paws scan`, or `unbound-paws login --scan`, which scans once the login
works. It reads the tool calls in your past Claude Code sessions (`~/.claude/projects/**/*.jsonl`, subagent logs
included, `$CLAUDE_CONFIG_DIR` when set), matches them against the catalog's triggers on this machine with the
gate's own matcher, and sends `POST /history` one summary of counts, so the portal can rank skills before
the hooks have synced anything.

- `--days 30` (the default) or `--days 90`. Claude Code deletes sessions older than 30 days unless its
  `cleanupPeriodDays` setting was raised, so 90 finds more only then.
- Once logged in, every call the hook logged (`~/.paws/audit.log`, from before the login too) is left out: sync uploads those. Each call is matched to its own record by session, tool, time and command or file, so calls the hook never logged still count. Only records this login's syncs uploaded count: `login` notes where sync's cursor stood the first time each account logged in on this machine (`sync_from` in `credentials.json`, kept per account in `~/.paws/sync-from.json`, which logout leaves), so after switching accounts the calls an earlier account received still count for the new one, and logging in again with the same account doesn't count its synced calls twice. A call from the last 12 hours waits for a later scan until its result has been in the transcript for 2 minutes: it may still be running, or its async PostToolUse hook may still be writing the record. Once the audit log has rotated, tool calls older than its oldest kept record are left out, since their records, and what they uploaded, may be gone; skill loads still count.
- Only the `tool_use` blocks of assistant records are decoded. Prompts, replies, thinking and tool results are
  never read. A call copied into a resumed or forked session's log counts once, by its id.
- `unbound-paws scan --dry-run` prints the body and sends nothing; an upload sends those same bytes. It works
  without a login.
- Each scan replaces this laptop's last one on the portal, so running it again never adds to the counts.

The body:

```
{ "source": "claude-code", "window_days": 30, "since": "2026-10-20", "until": "2026-11-19",
  "sessions": 412, "tool_calls": 18234,
  "skills": [{"skill": "coreutils", "calls": 19, "last_seen": "2026-11-17",
              "actions": [{"id": "coreutils", "calls": 19}]}],
  "loaded": [{"name": "humanizer", "count": 12}],
  "unmatched": [{"name": "bash kubectl delete", "count": 4}, {"name": "edit .tf", "count": 9}],
  "catalog_version": "..." }
```

| Field | Value |
|---|---|
| `skills` | catalog skills whose triggers matched: calls (each call once, however many of the skill's actions matched it), the last day one ran, and calls per action id |
| `loaded` | catalog skills Claude loaded without being asked. Other skill names are never sent |
| `unmatched` | calls no trigger matched, at most 100, by signature. Every word comes from a list paws ships or from the catalog, never from you: a known program plus the CLI's own subcommands (`kubectl delete`, `rm`; for aws, gcloud and az, an operation only when it is made of known cloud verbs and nouns, as in `aws s3api create-bucket`), a SQL verb, `read`/`edit` plus a known file extension, `mcp <server> <tool>` when a catalog trigger names that server and tool exactly, `mcp <server> <class>` for another tool on such a server, `mcp <word> <class>` for a server a wildcard trigger covers (the trigger's own word: `*azure*` sends `azure`, never your server's name), or a Claude Code tool's name. Anything else is `other`: an unknown program is `bash other`, a git or kubectl alias or plugin stops at `git` or `kubectl`, an MCP server you set up yourself is `mcp other <create\|update\|delete\|send\|read>`, and an unknown extension or tool is left out |
| `sessions`, `tool_calls` | how many sessions had a call in the window, and how many calls (Skill loads not included) |

Never sent: commands, arguments, paths, hostnames, MCP inputs, prompts, replies, tool output, session or project
names.

The counts answer the catalog's questions, so a scan trusts the catalog the way installing its skills already
does: a catalog of thousands of narrow triggers could learn about past commands from which ones matched. A scan
refuses a catalog of more than 500 skills, 1,000 triggers or 20 triggers per skill (the published catalog has 169,
217 and 3). The scan date and window are kept in `~/.paws/portal/scan.json`.

## Portal settings

Settings come down; the gate stays offline. When the config hash changes
(from `GET /config/hash`, or from an upload's response), paws fetches the
config and applies it:

- one rule file per skill and action under `~/.paws/portal/`
  (`<skill>--<action id>.md`; an id that had to be shortened or cleaned up
  gets a short hash of the original), a plain `require-skill` rule that paws
  loads like any other. Files the config no longer has, including turned-off
  skills, are removed. Every write is atomic;
- an action that would gate the Skill tool (no tool and no matcher, or tool
  `Skill`) is skipped and reported: it would deny the call that loads the
  skill. The gate never gates a skill load in any case: the Skill tool, or a
  plain read of the skill's `SKILL.md`;
- a `load: rule` skill becomes `on_miss: digest` with its rule text as the
  digest (`~/.paws/portal/digests/<skill>.md`); no SKILL.md is installed.
  Rule text over 7,000 bytes is skipped;
- a catalog skill installs into `~/.claude/skills/<name>`: from upstream at
  its pinned commit, or from the catalog in the paws binary, and only when its
  content hash matches the config. An upstream skill must match a catalog pin
  in this paws binary exactly (a `github.com` repo, path, commit and content
  hash), so the portal can only choose among pins paws already ships; anything
  else is skipped as "not in this unbound-paws version's catalog". A mismatch is
  skipped and reported. A skill directory paws didn't install, or one you
  edited, is left as it is, as with `unbound-paws add`. Turning a skill off removes
  its rules, not its skill directory;
- a skill's rules are written only when its `SKILL.md` is in
  `~/.claude/skills/<name>` or the project's `.claude/skills/<name>` (or it
  is `load: rule`); otherwise it is skipped as "skill not installed". A custom
  skill is yours to install: paws never downloads one;
- the applied hash, the applied skills and the last upload and check times
  are kept in `~/.paws/portal/state.json`. A skill that failed to download
  leaves the hash unrecorded, so the next check tries again.
