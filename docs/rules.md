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

paws splits a Bash call into the commands the shell will run and tests
`command_regex` against each one. `cd web && npm install x` is two commands,
`npm install x` one of them. Commands inside `bash -c "..."`, `sh <<EOF`,
`cat <<EOF | bash`, `eval`, `( )`, `$( )`, `if`, loops and pipes count too, and
so does a function's body where the line calls it (not where it only defines it). So does a command
behind `sudo`, `env`, `xargs`, `time`, `nohup`, `command` or `exec`, which is
tested both with and without that prefix. Quotes are removed, as the program
sees them; `$VAR` and `$(...)` stay as written. Variable assignments
(`HUSKY=0 git commit`, `export X=1`) stay visible to the regex. A `;`, `&` or `|` inside an argument becomes a space, so
`echo "a; npm install x"` holds no `npm install` command. A call that isn't
valid Bash is matched whole. `pre-commit` and `pre-push` find git the same way.

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
invoked in the last 20 agent turns, then allows it. On `session-start`,
which can't be denied, it adds context naming the skill instead (`message`
replaces that text).

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
