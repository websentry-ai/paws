# paws

**P**re-tool **A**gent **W**orkflow **S**ystem — Husky for AI coding agents.
Drop a `.paws/` folder of markdown rules in your repo; paws registers itself
as a Claude Code hook and gates every tool call against them. No daemon, no
compile step, no network. Fails open — any internal error allows the call.

Apache-2.0 · v0.1 targets Claude Code.

## Install

```
go build -o paws ./cmd/paws
./paws init                        # registers paws in ./.claude/settings.json
```

`--global` registers in `~/.claude/settings.json` instead (every repo on the
machine); `--settings <path>` targets an explicit file. `paws uninstall`
takes the same flags and removes the entry.

## Write a rule

`.paws/pre-tool/aws-tagging.md`:

```markdown
---
name: aws-tagging
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|])\s*(aws (ec2 run-instances|s3 mb)|terraform apply)
action:
  type: require-skill
  skill: aws-tagging-standards
  deny_message: Invoke the aws-tagging-standards skill, then retry this exact command.
---
Ensures every AWS resource carries Owner/Environment/ManagedBy tags.
```

Rules take effect immediately — no restart, no re-init. paws reads
`.paws/**/*.md` from cwd up to the repo root (or `$HOME`), plus `~/.paws`,
fresh on every tool call. `$PAWS_HOME`, when set to an absolute path, replaces
`~/.paws` (user rules and the audit log). Claude Code passes a settings
file's `env` to hooks, so a repo whose `.claude/settings.json` sets
`PAWS_HOME` can move your user rules and audit log for sessions in that repo;
review `.claude/settings.json` like you review `.paws/`.

## Events

| `event:` | fires on |
|---|---|
| `pre-tool` | any tool call |
| `pre-push` | `pre-tool`, plus the Bash command must match `git push` |
| `pre-commit` | `pre-tool`, plus the Bash command must match `git commit` |
| `user-prompt` | the submitted prompt text, not a tool call |

## Matcher

`tool:` — a tool name (`Bash`, `Read`, `Write`, `Edit`, `Grep`, `Glob`,
`WebFetch`, `WebSearch`, `mcp__<server>__<tool>`), an `A|B` list, or
omitted/`"*"` (quoted) for any.

`command_regex:` — matched against the Bash command, file path, URL, or
prompt text, depending on the event. It's a regex over literal text, not a
shell parser — anchor with `(^|[;&|])\s*` if the match must hold whether the
command is chained after `cd x &&`, `;`, or `|`.

## Actions

| action | effect |
|---|---|
| `block` | hard deny, no retry path |
| `require-skill` | deny; clears once the agent invokes `skill:` (checked against its session transcript, last 20 turns) and retries the exact same command |
| `warn` | Claude Code shows an interactive approval prompt; proceeds only if the human says yes. Not supported on `user-prompt` |
| `audit` | allowed silently; recorded to `~/.paws/audit.log`. Agent sees nothing — pure observation |
| `prompt-inject` | `user-prompt` rules only — appends `message:` to the prompt. Not recorded to `~/.paws/audit.log` |
| `exec-script` | delegates to `script:` (relative path inside the rule's own directory); exit 2 denies, stderr is the reason, anything else allows |

When multiple rules match, the most restrictive wins: `block`/`require-skill`
(unresolved) > `warn` > `audit`/`prompt-inject`/allow.

## Commands

```
paws explain 'terraform apply'             # dry-run: matching rules + decision, no session
paws explain --tool Edit config/prod.env   # dry-run a non-Bash tool call
paws explain --event user-prompt 'ship it' # dry-run a prompt
echo '<event json>' | paws explain -       # dry-run a full event JSON
paws gate                                  # the hook entrypoint (reads event JSON on stdin)
go test ./...
python3 .claude/skills/verify-paws/scripts/drive.py run --all   # live headless Claude Code e2e
```

## Layout

```
cmd/paws      CLI entrypoint (dispatch)
internal/hook Claude Code adapter: I/O contract + translation to/from the gate engine
internal/rule rule model: parse + match (pure, agent-agnostic)
internal/gate decision engine: load, resolve, exec-script, audit, transcript scan (agent-agnostic)
internal/cli  the gate / init / uninstall / explain commands
examples      sample .paws rules
```
