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
action:                           # what happens
  type: require-skill
  skill: humanizer
---
Loads humanizer before a PR body goes out.
```

## Events

| `event:` | Fires on | Target the regex sees |
|---|---|---|
| `pre-tool` | any tool call | see the table below |
| `pre-commit` | a Bash call that runs `git commit` (flags and `-C dir` allowed) | the command |
| `pre-push` | a Bash call that runs `git push` | the command |
| `user-prompt` | the prompt the user submits | the prompt text |
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
| `tool` | a name, `prefix*` (e.g. `mcp__slack__*`), an `A\|B` list, or omitted for any |
| `command_regex` | Go RE2 regex over the target |
| `file_glob` | `**`, `*`, `?`, `{a,b}` over the absolute path; Read/Write/Edit/NotebookEdit only |
| `mcp_action` | `send`, `create`, `update`, `delete` (or `A\|B`): MCP tools whose name says so |

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
| `dependency:PKG` | the repo root's package.json, go.mod, Cargo.toml, pyproject.toml or requirements.txt lists PKG |
| `repo-marker:PATH` | PATH exists under the repo root |
| `new-file` | the Read/Write/Edit target does not exist yet |
| `no-edit-matching:GLOB` | the agent has not edited or written a matching path in this session |
| `agent-count>N` | the agent spawned more than N subagents since your last prompt |

The repo root is the nearest directory with `.git` above the agent's cwd.

## Actions

| `action.type` | Effect | Events |
|---|---|---|
| `require-skill` | deny until `skill` was invoked in the last 20 agent turns, then allow; on `session-start`, add context naming the skill | all |
| `advise` | allow, and attach `message` as context once per 20-turn window | tool events, `session-start` |
| `block` | hard deny with `deny_message` | tool events, `user-prompt` |
| `warn` | ask the human to approve | tool events |
| `audit` | allow silently; log it | all |
| `prompt-inject` | add `message` to the prompt | `user-prompt` |
| `exec-script` | run `script` (relative, inside the rule's dir) with the event JSON on stdin; exit 2 denies with stderr as the reason | tool events, `user-prompt` |

`require-skill` takes an optional `deny_message`; the default names the
skill and tells the agent to invoke it and retry the same command. A
compaction resets the 20-turn window, so the next gated call denies once more.

When several rules match, the most restrictive wins: `block` > unresolved
`require-skill` / `exec-script` deny > `warn` > allow. Context from every
matching rule is joined.

## Test a rule

```
paws explain 'gh pr create --fill'              # Bash by default
paws explain --tool Edit src/app/page.tsx
paws explain --event user-prompt 'deploy it'
paws explain --event session-start compact
echo '<hook event json>' | paws explain -
```

`explain` prints the rules in scope, marks the ones that apply (matcher and
`when:`), and shows the decision. It writes no audit record and runs no
script.

## Failure modes

paws fails open: a rule that fails to parse is skipped (Claude Code shows a
one-line notice), a missing transcript counts as "skill not loaded", and an
internal error allows the call. `paws validate` checks catalog bindings
strictly, including unknown keys.
