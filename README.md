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

paws guarantees the skill was loaded; the skill governs the behaviour. The
hook never rewrites tool input or reads tool output, and the gate that decides
each tool call never calls the network.

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
| events | `pre-tool`, `pre-commit`, `pre-push`, `session-start` |
| matcher | `tool` (names, `mcp__server__*`), `command_regex`, `file_glob`, `mcp_action` (`send`, `create`, `update`, `delete`) |
| when | `dependency:PKG`, `repo-marker:PATH`, `new-file`, `no-edit-matching:GLOB`, `agent-count>N` |
| action | `require-skill`, with `on_miss: digest` to send the skill's rules in the deny |

MCP calls match by server and by what the call does, so one rule covers
every Slack tool that sends:

```yaml
matcher:
  tool: mcp__slack__*
  mcp_action: send      # send, create, update or delete
```

The full reference: [docs/rules.md](docs/rules.md).

## What paws does and doesn't do

- **Local only.** The hook reads rule files and the session transcript and
  writes `~/.paws/audit.log`. Nothing leaves the machine unless you run
  `paws login` ([docs/telemetry.md](docs/telemetry.md)).
- **Fails open.** A rule that doesn't parse is skipped with a one-line
  notice; an internal error allows the call. A hook that wedges your terminal
  gets uninstalled within the hour.
- **Per context window.** A skill counts as loaded when it was invoked in the
  last 20 agent turns. After a compaction its text is gone, so the next gated
  call denies once more.
- **Not a sandbox.** Regex rules match command text, not shell semantics.
  A gate makes an agent that follows instructions load the skill; it doesn't
  stop an adversary.

## Portal (optional)

Nothing leaves the machine until you run `paws login`.

```bash
paws login              # confirm a code in the browser; saves ~/.paws/credentials.json
paws login --key pk_... # a machine with no browser: a one-time key made in the portal
paws login --key -      # the same, reading the key from stdin
paws sync --dry-run     # print exactly what the next upload holds, send nothing
paws sync               # upload and pull settings now
paws logout             # forget the token and remove the portal's rules
```

What leaves the machine once you are logged in:

- counts: each time a rule fired, its skill and verdict, and whether the
  skill was already loaded;
- the commands Claude ran, with secrets redacted (passwords, tokens, API keys,
  auth headers, JWTs; [the list](docs/telemetry.md#redaction)) and cut to 500
  characters;
- MCP calls: server, tool and action class (send, create, update, delete);
- tool actions as a short key, such as `git commit`, `gh pr create`,
  `edit migrations/**` or `DELETE`;
- file paths relative to the repo root (a path outside the repo is left out);
- the names of your installed skills and the paws rules in scope.

Never prompts, replies, file contents, tool output, absolute paths or session
ids. Field by field: [docs/telemetry.md](docs/telemetry.md#portal-upload).

When: after Claude finishes a turn, from the async Stop hook, at most every
10 minutes (`PAWS_SYNC_INTERVAL`); on `paws sync`; and once at `paws login`.

Settings come down from the portal: the skills you turn on in the browser
and the actions each one runs before. The Stop hook checks for a change at
most every 15 minutes (`PAWS_CONFIG_INTERVAL`), then writes them as rule files
under `~/.paws/portal/`. Only skills in the catalog this paws version ships
can install, into `~/.claude/skills` at their pinned commit, and are refused
if their content hash differs; the portal can't name other code to download.
A skill's rules are written only once the skill is installed; a skill
directory you wrote or edited is never replaced. The gate itself stays
offline: it reads those files like any other rule, and a tool call is never
held up by the network. `paws logout`, or a login revoked in the portal,
removes those rules.

`PAWS_PORTAL_API` sets the portal address (default
`https://backend.getunbound.ai`). It must be https; plain http is accepted
only for `localhost`, `127.0.0.1` or `::1`, with a warning.

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

Upgrading from 0.1: re-run `paws init` once. 0.2 registers SessionStart
(for `session-start` rules) and PostToolUse and Stop (for local telemetry,
async), and drops the UserPromptSubmit hook. 0.2 runs `require-skill` rules
only; a rule using an older action is skipped with a notice. From a 0.2
release candidate: re-run `paws init`, then `paws add <pack>` for each
installed pack.

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
internal/gate     decision engine: load, resolve, transcript scan, audit log
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
