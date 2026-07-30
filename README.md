# paws

**P**re-tool **A**gent **W**orkflow **S**ystem — Husky for AI coding agents.

Declare workflow gates on agent behavior as checked-in markdown files. Drop a
`.paws/` folder in your repo; paws registers itself once as a Claude Code hook
and gates each tool call against your rules.

Apache-2.0 · v0.1 targets Claude Code.

## How it works

`paws` is a single static binary registered as a Claude Code hook. On every tool
call Claude Code runs `paws gate` and pipes it the event JSON. paws walks from
the current directory up to the git repo root (or `$HOME` if there is none),
reads every `.paws/**/*.md` rule in scope (plus `~/.paws`), matches the pending
call, and returns Claude Code's native allow/deny decision. No daemon, no
compile step, no network.

It **fails open** — any error returns "allow", so it can never wedge your agent.
It also never touches other hooks: Claude Code runs every registered hook and
takes the most restrictive result, so paws runs happily alongside them.

## A rule

One markdown file per gate. Frontmatter drives execution; the body is for humans.

```markdown
---
name: aws-tagging
event: pre-tool
matcher:
  tool: Bash
  command_regex: ^(aws (ec2 run-instances|s3 mb)|terraform apply)
action:
  type: require-skill
  skill: aws-tagging-standards
  deny_message: Load aws-tagging-standards, read its SKILL.md, then retry.
---
# AWS Tagging Hook
Ensures every AWS resource carries Owner/Environment/ManagedBy tags.
```

**Events:** `pre-tool` (any tool call), `pre-commit` / `pre-push` (git sugar —
Bash-only, matches `git commit` / `git push`), `user-prompt` (prompt submit).

**Matcher:** `tool` is a tool name (`Bash`, `Read`, `Write`, `Edit`, `Grep`,
`Glob`, `WebFetch`, `WebSearch`, `mcp__<server>__<tool>`, ...), a `A|B` pipe
list, or omitted/`"*"` (must be quoted in YAML) for any tool. `command_regex`
matches against the Bash command, file path, URL, or prompt text, depending
on the event.

**Actions:**

| action             | effect                                                 |
|--------------------|--------------------------------------------------------|
| `block`            | hard deny                                              |
| `require-skill`    | deny with a "load this skill, then retry" message      |
| `require-approval` | ask the user to approve (Claude Code `ask`)            |
| `warn`             | allow, but surface an advisory to the agent            |
| `audit`            | allow silently, record to `~/.paws/audit.log`          |
| `prompt-inject`    | append text to the prompt (`user-prompt` rules only)   |
| `exec-script`      | run a script; it denies via exit code 2 (escape hatch) |

When several rules match, the most restrictive wins: deny > ask > allow.
`exec-script` scripts get paws-shaped JSON on stdin and run under a 5s timeout.

## Usage

```
paws init                                 # register paws in ./.claude/settings.json
paws init --global                        # register in ~/.claude/settings.json instead
paws init --settings <path>                # register in an explicit settings.json
paws uninstall [--global | --settings ..]  # remove paws from a settings.json
paws explain 'terraform apply'             # dry-run: show matches + decision, no session
paws explain --tool Edit config/prod.env   # dry-run a non-Bash tool call
echo '<event json>' | paws explain -       # dry-run a full event JSON
paws gate                                  # the hook entrypoint (reads event JSON on stdin)
```

`paws init` writes the settings.json at your repo root by default (walking up
from cwd to find it); `--global` targets `~/.claude` instead. Re-running `init`
after moving the binary repoints the existing entry rather than duplicating it.

## Build

```
go build -o paws ./cmd/paws
go test ./...
```

## Layout

```
cmd/paws      CLI entrypoint (dispatch)
internal/hook Claude Code adapter: I/O contract + translation to/from the gate engine
internal/rule rule model: parse + match (pure, agent-agnostic)
internal/gate decision engine: load, resolve, exec-script, audit (agent-agnostic)
internal/cli  the gate / init / uninstall / explain commands
examples      sample .paws rules
```

Dependency flow is one-way: `hook` (the adapter) depends on `gate`, which
depends on `rule`. Neither `gate` nor `rule` knows Claude Code exists — a
future adapter for another agent is a sibling package next to `hook`.
