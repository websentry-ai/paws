# Writing paws rules

A rule is one markdown file under `.paws/` (repo) or `~/.paws/` (you). The
frontmatter is the rule; the body is its README. paws reads every rule fresh
on each tool call, so an edit takes effect on the next call.

```markdown
---
name: humanizer-pr                # unique, shows in deny messages and the audit log
event: pre-tool                   # when it can fire
matcher:                          # which calls (all fields must match)
  tool: Bash
  command_regex: (^|[;&|\n])\s*gh\s+pr\s+create\b
when: dependency:react            # optional repo / session conditions
action:                           # the skill to load first
  type: require-skill
  skill: humanizer
---
Loads humanizer before a PR body goes out.
```

## Events

| `event:` | Fires on | Target the regex sees |
|---|---|---|
| `pre-tool` | any tool call | see the table below |
| `pre-commit` | a Bash call that runs `git commit` or `git commit-tree` anywhere in it (flags and `-C dir` allowed; not `commit-graph`) | the command |
| `pre-push` | a Bash call that runs `git push` anywhere in it | the command |
| `session-start` | a session starting | `startup`, `resume`, `clear` or `compact` |

Per tool, `pre-tool` matches against:

| Tool | Target |
|---|---|
| Bash | `command` |
| Read, Write, Edit | `file_path` |
| NotebookEdit | `notebook_path` |
| Grep, Glob | `pattern` |
| WebFetch | `url` |
| WebSearch | `query` |
| Task / Agent | `prompt` |
| `mcp__*` | the whole tool input as JSON |

## Matcher

| Field | Matches |
|---|---|
| `tool` | a name, a pattern where `*` stands for any characters (e.g. `mcp__slack__*`, `mcp__*__get_issue`), an `A\|B` list, or omitted for any |
| `command_regex` | Go RE2 regex over the target; for Bash, over each command in it (see below) |
| `file_glob` | `**`, `*`, `?`, `{a,b}` over the absolute path; Read/Write/Edit/NotebookEdit only |
| `mcp_action` | `send`, `create`, `update`, `delete` (or `A\|B`): MCP tools whose name says so |

### Bash: one command at a time

A `command_regex` is matched against the whole Bash call, and also against
each command the shell will run, which paws finds with a shell parser. A rule
fires if either matches, so the parser only adds matches:

- `cd web && npm install x` holds the command `npm install x`.
- Commands run by `bash -c "..."` (any flags, as in `bash -euo pipefail -c`),
  `sh <<EOF`, `bash <<< '...'`, `cat <<EOF | bash`, `echo ... | sh`, `eval`,
  `( )`, `$( )`, `if` and loops count. So does a function's body where the
  line calls the function, but not where it only defines it.
- A command behind `sudo`, `env`, `xargs`, `time`, `nohup`, `command` or
  `exec` is tested with and without that prefix.
- A command's quotes and backslashes are removed, as the program sees them
  (`\git push` is `git push`). `$VAR`, `$(...)` and assignments
  (`HUSKY=0 git commit`, `export X=1`) stay as written.
- A call that isn't valid Bash is matched whole only.

`pre-commit` and `pre-push` find git the same way.

### RE2, not PCRE

`command_regex` is Go RE2: no lookahead `(?=`, no lookbehind `(?<=`, no
backreferences. Match the command at a command boundary, not anywhere:

```
(^|[;&|\n])\s*git\s+push\b
```

`(^|[;&|\n])\s*` holds whether the command is first, chained after `&&`, `;`,
`|`, or on a new line, and keeps `echo "git push"` from matching. Quote the
regex in YAML when it starts with a special character or contains `: `.

### mcp_action

paws classifies an MCP tool by the leftmost verb in its own name, whatever the
server prefix: `slack_send_message` is a send, `create_pull_request` a create,
`list_messages` a read (never matched). Creating a comment, draft, message or
reply counts as a send, and so does updating a draft.

## when

One condition, or a list where every condition must hold.

| Condition | Holds when |
|---|---|
| `dependency:PKG` | the repo root lists PKG as a dependency: package.json (4 dependency maps), go.mod direct requires (full path, or last element without `/vN`), Cargo.toml dependency tables (key or `package` rename), pyproject.toml dependency tables, requirements.txt |
| `repo-marker:PATH` | PATH exists under the repo root |
| `new-file` | the Read/Write/Edit target does not exist yet |
| `no-edit-matching:GLOB` | no successful Edit/Write on a matching path in the last 1 MB of the transcript (denied or failed edits don't count; very long sessions forget older edits) |
| `agent-count>N` | the agent spawned more than N subagents since the last prompt you typed |

The repo root is the nearest directory with `.git` above the agent's cwd.

## Action

`require-skill` is the one action. It denies the call until `skill` was
invoked in the last 20 turns, then allows it. A turn is one message the user
sends and everything the agent does to answer it, however many tool calls.
On `session-start`, which can't be denied, it adds context naming the skill
instead (`message` replaces that text).

A skill is loaded by the Skill tool, or by reading its `SKILL.md` in a
folder a harness loads skills from (`~/.claude/skills`, the project's
`.claude/skills`, a Claude Code plugin's `skills/`): the Read tool on the
file, or a Bash command
that only prints it (`cat`, `sed -n '1,240p'`, `head`, `tail`, `nl`, `less`,
`more`, `bat`, an `awk` print). A read chained with anything else
(`cat SKILL.md && git push`) is not a load. A load is never gated, so a rule
that matches every call can't block the call that clears it; it counts only
when it succeeded. A skill installed after the session started isn't in
Claude Code's skill list, so the default deny also names the skill's
`SKILL.md` when paws finds it in `~/.claude/skills` or the project's
`.claude/skills`.

`deny_message` replaces the default deny, which names the skill and tells
the agent to invoke it and retry the same command. A compaction resets the
20-turn window, so the next gated call denies once more.

A rule with any other action, or an event not listed above, is skipped with
a notice, and `unbound-paws explain` lists it as `! skipped`.

### on_miss: digest

For a skill the user may not have installed, ship a digest beside the rule:

```yaml
action:
  type: require-skill
  skill: aws-s3
  on_miss: digest               # default: load-required
  digest: digests/aws-s3.md     # relative to the rule file, at most 7,000 bytes
```

The deny then carries the digest. Once that deny is in the transcript window,
the retry passes without the skill; the audit log records
`satisfied_by: digest-delivered`. A loaded skill still passes first. A missing or oversize digest falls back to
the plain load-required deny, and the audit record carries `digest_error`.
Delivery is keyed by rule name: two rules with the same name share it.

When several rules match one call, one deny names every skill still missing,
so a single retry clears them all.

## expect

Optional checks of what should follow when the rule blocks a call. paws grades them at the end of the turn, on
the turn's own tool calls, and keeps only pass or fail (see [outcome records](telemetry.md#records-schema-v2)):

```yaml
expect:
  - id: listed-first
    label: Listed what it would delete first
    when: before
    if: call.tool == "Bash" && call.text matches "^(ls|du|stat)\\b"
  - id: not-home
    label: Never deleted the home directory
    when: never
    if: call.text matches "rm\\s+-\\S*r\\S*\\s+(~|\\$HOME)/?(\\s|$)"
```

`if` is an [expr](https://expr-lang.org) expression over two calls: `call`, the one being checked, and `blocked`,
the call the rule denied. Each has `tool` (`Bash`, `Write`, `Edit`, `mcp__notion__create_pages` …), `text` (a Bash
command, what a Write writes, an Edit's new text, an MCP call's text arguments), `input` (the raw tool input, so
`call.input.parent != nil` works for any field) and, for MCP calls, `mcp.server`, `mcp.tool` and `mcp.class`.

For a Bash call, `commands` is each command the line runs, with quotes removed and `sudo`/`env` dropped:
`cd web && sudo git push -f` holds `cd web` and `git push -f`. Text a command only reads isn't a command: a heredoc
into `cat` or `python`, a commit message, an `echo`'s argument. A script a shell runs is (`bash -c '...'`, a heredoc
into `sh`). To ask "did it run X", match `commands`, not `text`: `any(call.commands, # matches "^git push\\b.* -f")`
ignores a Python script that only prints `git push -f`. A command counts whether or not it ran: paws can't see which side of `&&` or `||`
ran, so `[ -f x ] && npm test` counts as running the tests. A line that doesn't parse (cut short inside a heredoc, say)
is split at `&&`, `||`, `;`, `|` and newlines instead.

A check can have `cases` instead of an `if`, one per way of doing the thing, for a rule that covers several (git and
the gh CLI, or a command and an MCP tool). The first case whose `match` holds on the retry, or on the blocked call
when there's no retry, gives the `if`. When no case matches, the check doesn't apply to that block: it's left out of
the grade instead of passing, so a commit-message check isn't counted on a `gh pr create`.

```yaml
  - id: pr-described
    label: The pull request has a description
    when: retry
    cases:
      - match: 'call.tool == "Bash" && any(call.commands, # matches "^gh pr (create|edit)\\b")'
        if: 'any(call.commands, # matches "^gh pr (create|edit)\\b.* (--body|-b|--body-file|-F) \\S")'
      - match: 'call.tool endsWith "create_pull_request"'
        if: 'len(trim(call.input.body ?? "")) >= 20'
```

Quote an expression that holds `#` or `: ` in single quotes, as above: unquoted, YAML reads ` #` as a comment.

`when` picks the calls the check looks at. All of them come from the same turn and the blocked agent; for a deny in
the main conversation, subagents it starts after the deny count too, and one already running doesn't (a subagent
can't start subagents of its own in Claude Code). A compaction summary or a task notification mid-turn doesn't end
the turn; only a prompt you type does:

| `when` | Passes when |
|---|---|
| `before` | a call earlier in the turn than the retry, the retry included, matches (before the deny counts) |
| `retry` | the retry (the next call the rule matches that goes through) matches |
| `after` | a call logged after the retry matches (not one in its own parallel batch) |
| `never` | no call after the deny matches |

`unbound-paws explain '<command>'` lists each check's result with that call as the retry (pass, FAIL, or doesn't
apply), so a check can be tried on calls that should pass and fail before it ships. The paws skill's `expect.md`
walks through writing checks.

A rule whose checks all pass (the ones that apply; at least one) is graded `fixed`. A check that doesn't compile is dropped with a warning (`unbound-paws
explain` shows it): the rule keeps enforcing, since checks only measure. A retried Write or Edit counts as the same
call only when both the path and the new text match. An `if` that fails to run on a call (a string compared as a
number, say) is no match, except for `never`, where it fails the check: it can't show the call didn't happen. Guard
fields with the tool first (`call.tool == "Bash" && ...`); `&&` stops before reading them.

## Test a rule

```
unbound-paws explain 'gh pr create --fill'              # Bash by default
unbound-paws explain --tool Edit src/app/page.tsx
unbound-paws explain --event session-start compact
echo '<hook event json>' | unbound-paws explain -
```

`explain` prints the rules in scope, marks the ones that apply (matcher and
`when:`), and shows the decision. It writes no audit record.

## Failure modes

paws fails open on its own errors: a rule that fails to parse is skipped
(Claude Code shows a one-line notice) and an internal error allows the call.
Missing session state is read as "nothing happened yet": with no transcript,
a skill counts as not loaded (require-skill denies), no edit is known
(no-edit-matching holds) and no subagents were spawned (agent-count is 0). `unbound-paws validate` checks catalog bindings
strictly, including unknown keys.
