# paws

**P**re-tool **A**gent **W**orkflow **S**ystem — Husky for AI coding agents.
Drop a `.paws/` folder of markdown rules in your repo; paws registers itself
as a Claude Code hook and gates every tool call against them. No daemon, no
compile step, no network. Fails open — any internal error allows the call.

Apache-2.0 · targets Claude Code.

## Install

```
npx unbound-paws init              # registers paws in ./.claude/settings.json
```

The npm package downloads the release binary for your platform (checked
against the release's `checksums.txt`). `paws init` copies the running binary
to `~/.paws/bin/paws` and registers that path, so clearing the npm cache or
deleting a `go build` output never breaks the hook. Re-run `init` after an
upgrade to replace the copy. Every repo and the global hook share that one
binary, so `init` from any version (including an older one, or a dev build)
swaps it for all of them.

From source: `go build -o paws ./cmd/paws && ./paws init`.

Upgrading from 0.1: re-run `paws init` once. 0.2 registers a SessionStart
hook as well, and `session-start` rules never fire without it.

`--global` registers in `~/.claude/settings.json` instead (every repo on the
machine); `--settings <path>` targets an explicit file. `paws uninstall`
takes the same flags and removes the entry; it leaves `~/.paws/bin/paws` for
other repos that still use it.

## Release

Push a `v*` tag. GoReleaser builds darwin/linux/windows for amd64/arm64 and
attaches archives plus `checksums.txt` to the GitHub release. The `npm` job
then publishes `unbound-paws` at the tag's version (prereleases to the `next`
dist-tag) once an `NPM_TOKEN` secret exists. The npm installer downloads from
the release, so the repo must be public for `npx` installs to work.

Test the npm path locally without a release: `goreleaser release --snapshot
--clean`, serve `dist/` over HTTP, and run the packed `npm/` tarball with
`PAWS_DOWNLOAD_BASE=http://127.0.0.1:<port>`.

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

The full authoring reference, with RE2 tips and `when:` conditions, is
[docs/rules.md](docs/rules.md).

## Events

| `event:` | fires on |
|---|---|
| `pre-tool` | any tool call |
| `pre-push` | `pre-tool`, plus the Bash command must match `git push` |
| `pre-commit` | `pre-tool`, plus the Bash command must match `git commit` |
| `user-prompt` | the submitted prompt text, not a tool call |
| `session-start` | a session starting; `command_regex` matches the source (`startup`, `resume`, `clear`, `compact`) |

## Matcher

`tool:` — a tool name (`Bash`, `Read`, `Write`, `Edit`, `Grep`, `Glob`,
`WebFetch`, `WebSearch`, `mcp__<server>__<tool>`), a `prefix*` pattern
(`mcp__slack__*`), an `A|B` list of either, or omitted/`"*"` (quoted) for any.

`mcp_action:` — `send`, `create`, `update` or `delete` (or an `A|B` list):
any MCP tool whose own name says it does that, on any server. Reads never
match; drafts and comments count as `send`.

`file_glob:` — `**`, `*`, `?`, `{a,b}` over the absolute `file_path` of
Read/Write/Edit. A relative glob (`src/**/*.ts`) matches at any directory.

`command_regex:` — matched against the Bash command, file path, URL, or
prompt text, depending on the event. It's a regex over literal text, not a
shell parser — anchor with `(^|[;&|])\s*` if the match must hold whether the
command is chained after `cd x &&`, `;`, or `|`.

## Actions

| action | effect |
|---|---|
| `block` | hard deny, no retry path |
| `require-skill` | deny; clears once the agent invokes `skill:` (checked against its session transcript, last 20 turns) and retries the exact same command. On `session-start` it adds context naming the skill instead |
| `warn` | Claude Code shows an interactive approval prompt; proceeds only if the human says yes. Not supported on `user-prompt` |
| `audit` | allowed silently; recorded to `~/.paws/audit.log`. Agent sees nothing — pure observation |
| `prompt-inject` | `user-prompt` rules only — appends `message:` to the prompt. Not recorded to `~/.paws/audit.log` |
| `advise` | allows the call and attaches `message:` as context, once per 20-turn window (again after a compaction) |
| `exec-script` | delegates to `script:` (relative path inside the rule's own directory); exit 2 denies, stderr is the reason, anything else allows |

When multiple rules match, the most restrictive wins: `block`/`require-skill`
(unresolved) > `warn` > `audit`/`prompt-inject`/allow.

## Commands

```
paws explain 'terraform apply'             # dry-run: matching rules + decision, no session
paws explain --tool Edit config/prod.env   # dry-run a non-Bash tool call
paws explain --event user-prompt 'ship it' # dry-run a prompt
echo '<event json>' | paws explain -       # dry-run a full event JSON
paws validate .                            # check catalog/ bindings and packs/ manifests
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
internal/catalog catalog bindings + pack manifests: load and validate
internal/cli  the gate / init / uninstall / explain / validate commands
catalog       shipped bindings (see docs/catalog.md)
packs         pack manifests: the unit `paws add` installs
docs          format references
npm           the unbound-paws npm wrapper (downloads the release binary)
examples      sample .paws rules
```
