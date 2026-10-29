# paws

**Skills that actually fire.**

You install a skill (humanizer, a TDD skill, your team's AWS rules) and your
coding agent decides on every turn whether it's relevant enough to load.
Often it decides no: the PR body goes out with the AI tells, the tests get
skipped, the bucket gets created without tags. Installed is not applied.

paws is a small local hook for Claude Code that fixes that one thing. When a
bound action is about to run (a commit, a PR, a Slack send, an S3 bucket),
paws checks the session transcript for the skill. If it isn't loaded, the call
is denied once with a message naming the skill; the agent loads it and
retries, and the retry goes through. Husky, for agents.

paws guarantees the skill was loaded; the skill governs the behaviour. It
never rewrites tool input, reads tool output, or calls the network.

The same hook runs plain guardrails too: block a command, ask you first, log
it, or hand it to a script. Any Bash command, file path or MCP call can be
matched; see [Guardrails](#guardrails).

Apache-2.0 · Claude Code · no account.

## Ten minutes

```bash
npx unbound-paws init      # registers the hook in this repo's .claude/settings.json
paws add writing           # humanizer before PR bodies, commits and MCP sends
```

(If `paws` isn't on your PATH yet, it's at `~/.paws/bin/paws`.)

Now ask Claude Code for a commit or a PR. You'll see:

```
agent runs `git commit ...`  → paws denies once:
  This action requires the "humanizer" skill. Invoke it with the Skill tool,
  then retry this exact command.
agent invokes humanizer, retries → the commit goes through, humanized
```

Dry-run any action without a session:

```bash
paws explain 'gh pr create --fill'
```

## Packs

```bash
paws search aws            # browse the catalog built into the binary
paws add writing           # install for you (~/.paws, ~/.claude/skills)
paws add --project writing # install into this repo, to commit for the team
paws list                  # what's installed, how often each binding fired
paws stats                 # what fired, and how often the skill was already loaded
paws ui                    # all of that as a local page
paws remove writing
```

`paws add` installs the bindings and the skills they require. Third-party
skills are fetched from upstream at a pinned commit and refused if their
content changed; skills you already have, or have edited, are left alone.
See [docs/catalog.md](docs/catalog.md).

## Your own rules

A rule is a markdown file in `.paws/` (repo) or `~/.paws/` (you). It takes
effect on the next tool call.

```markdown
---
name: humanizer-pr
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n])\s*gh\s+pr\s+(create|edit)\b
action:
  type: require-skill
  skill: humanizer
---
PR titles and bodies read like a person wrote them.
```

Or ask Claude Code: "always load humanizer before I open a PR". `paws init`
installs a `paws` skill that knows the format and checks its work with
`paws explain`.

| | |
|---|---|
| events | `pre-tool`, `pre-commit`, `pre-push`, `user-prompt`, `session-start` |
| matcher | `tool` (names, `mcp__server__*`), `command_regex`, `file_glob`, `mcp_action` (`send`, `create`, `update`, `delete`) |
| when | `dependency:PKG`, `repo-marker:PATH`, `new-file`, `no-edit-matching:GLOB`, `agent-count>N` |
| actions | `require-skill`, `advise`, `block`, `warn`, `audit`, `prompt-inject`, `exec-script` |

The full reference: [docs/rules.md](docs/rules.md).

## Guardrails

Rules don't need a skill. Swap the action and the same file is a guardrail
on commands, files or MCP calls:

| Action | What the agent sees | Example |
|---|---|---|
| `block` | a hard deny naming the rule, no retry | reading `.env` or `secrets/` |
| `warn` | Claude Code asks you to approve the call | `psql` against prod |
| `audit` | nothing; the call is logged to `~/.paws/audit.log` | every `npm publish` |
| `exec-script` | your script decides; exit 2 denies | `rm -rf` checked by a shell script |
| `advise` | the call runs, with a note added to context | style guide on the first `.tsx` edit |

```markdown
---
name: prod-db-approval
event: pre-tool
matcher:
  tool: Bash
  command_regex: (?s)psql.*(prod|production)
action:
  type: warn
  deny_message: Touching the prod DB needs approval.
---
```

MCP calls match by server and by what the call does, so one rule covers
every Slack tool that sends:

```yaml
matcher:
  tool: mcp__slack__*
  mcp_action: send      # send, create, update or delete
```

When several rules match one call, the strictest wins: `block`, then a
missing skill, then `warn`, then `audit`. More in
[examples/.paws](examples/.paws).

## What paws does and doesn't do

- **Local only.** The hook reads rule files and the session transcript and
  writes `~/.paws/audit.log`. Nothing leaves the machine
  ([docs/telemetry.md](docs/telemetry.md)).
- **Fails open.** A rule that doesn't parse is skipped with a one-line
  notice; an internal error allows the call. A hook that wedges your terminal
  gets uninstalled within the hour.
- **Per context window.** A skill counts as loaded when it was invoked in the
  last 20 agent turns. After a compaction its text is gone, so the next gated
  call denies once more.
- **Not a sandbox.** Regex rules match command text, not shell semantics.
  `block` stops an agent following instructions, not an adversary.

## Install details

```bash
npx unbound-paws init            # this repo
npx unbound-paws init --global   # every repo for this user (~/.claude/settings.json)
paws uninstall [--global]
```

The npm package downloads the release binary for your platform and checks it
against the release's `checksums.txt`. `paws init` copies the binary to
`~/.paws/bin/paws` and registers that path, so clearing the npm cache never
breaks the hook; re-run `init` after an upgrade. All repos share that one
binary. From source: `go build -o paws ./cmd/paws && ./paws init`.

Upgrading from 0.1: re-run `paws init` once. 0.2 also registers SessionStart
(for `session-start` rules) and PostToolUse and Stop (for local telemetry,
async).

`$PAWS_HOME`, when set to an absolute path, replaces `~/.paws`. Claude Code
passes a settings file's `env` to hooks, so a repo's `.claude/settings.json`
can set it; review `.claude/settings.json` like you review `.paws/`.

## FAQ

**An LLM following instructions isn't enforcement.** Right. paws
deterministically guarantees the skill was loaded before the action, not that
the agent followed it. That's why we say "activated", never "used".

**Isn't this just a rules file?** Claude Code's path-scoped rules load text
when the agent touches matching files. paws fires on commits, PRs, MCP sends,
session start and repo conditions, before the first write, and installs and
pins the skill itself.

**Open-core bait?** The hook, the rule format and the catalog are Apache-2.0,
and stay that way. A hosted portal for synced, opt-in telemetry will be the
paid part; the hook never needs it.

**A catalog of injected instructions is an attack surface.** Third-party
skills are installed from upstream at a pinned commit and refused if their
content hash changes; nothing is redistributed from this repo.

## Develop

```bash
go test ./...
go run ./cmd/paws validate .                     # catalog bindings and packs
go run ./cmd/paws pin .                          # pin upstream skill hashes
python3 .claude/skills/verify-paws/scripts/drive.py run --all        # live Claude Code e2e
python3 .claude/skills/verify-paws/scripts/drive.py run --npm launch-journey
```

`.claude/skills/verify-paws/` drives a real headless Claude Code session per
case against a freshly built paws and grades the transcript, the audit log
and repo side effects.

Releases run manually through the **CI and release** workflow on `main`.
The workflow requires a newer version in `npm/package.json`, runs checks,
then releases binaries and publishes to npm. Pushes and pull requests run
checks only. See [Release paws](docs/releases.md) for the first release,
the npm token setup, and installation checks.

```
cmd/paws          CLI entrypoint
internal/hook     Claude Code adapter (hook I/O, settings entries)
internal/rule     rule format: parse, match, when-conditions (pure)
internal/gate     decision engine: load, resolve, transcript scan, audit
internal/catalog  catalog bindings and pack manifests (disk or embedded)
internal/pack     paws add/remove: bindings, skill install, state
internal/cli      the commands
catalog, packs    shipped bindings and packs
skills/paws       the skill paws init installs
web               the paws ui React app; web/dist is the committed build
npm               the unbound-paws npm wrapper
docs              rules, catalog, telemetry references
examples          sample .paws rules
```
