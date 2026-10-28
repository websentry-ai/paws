---
name: verify-paws
description: Verify paws end to end by driving a real Claude Code session (headless, or the interactive TUI in tmux) against a freshly built paws binary, then grading the agent's tool calls, the paws audit log and the repo's side effects. Use after any change to rules, matchers, actions, the transcript scanner, the hook adapter, init, or packs, and before merging a paws PR.
---

# Verify paws

paws runs *inside* Claude Code as a hook, so the app under test is Claude Code
itself. `scripts/drive.py` builds paws from this checkout, makes a scratch git
repo per case, runs `paws init` there, runs `claude -p` headless in it, and
grades what happened.

## Launch

Nothing stays running. Each case builds nothing extra; the run builds paws once:

```bash
python3 .claude/skills/verify-paws/scripts/drive.py run require-skill-commit
python3 .claude/skills/verify-paws/scripts/drive.py run --all
python3 .claude/skills/verify-paws/scripts/drive.py run block-tag --keep   # keep scratch repo
```

A case finishes when `claude -p` exits (end of turn). Default timeout 300 s.

`followups` in a case resume the same session for more prompts
(`claude -p --resume <session>`), e.g. `"/compact"` and then a second task.

`--npm` tests the ship path instead of `go build`: it builds a GoReleaser
snapshot at `npm/package.json`'s version, packs `npm/`, serves the archives on
a local port, and each case runs `npx --package <tgz> unbound-paws init` with
`PAWS_DOWNLOAD_BASE` pointed at that port and its own npm cache. It needs
`goreleaser` and `npm`. The server stops when the run ends.

```bash
python3 .claude/skills/verify-paws/scripts/drive.py run --npm catalog-humanizer-commit
```

## Doctor

```bash
python3 .claude/skills/verify-paws/scripts/drive.py doctor
```

Checks `go`, `git`, `claude` are on PATH, paws builds, and Claude answers a
one-line headless prompt (auth works). Run it first when a case is
INDETERMINATE.

## Drive

What one case does, in order:

1. Copies `cases/<name>/project/` into a scratch dir and runs `git init`.
2. Copies `cases/<name>/skills/` into the scratch repo's `.claude/skills/`.
   Skills live outside `project/` so Claude Code never discovers them in this repo.
3. Copies `cases/<name>/paws-home/` (if any) into a scratch `PAWS_HOME`.
4. Runs `setup` commands, then `paws init <init_args>` (writes `.claude/settings.json`).
5. Runs `claude -p <prompt> --setting-sources project --allowedTools <allowed_tools>
   --output-format stream-json --verbose` with `PAWS_HOME` set and every
   parent session's markers (`CLAUDECODE`, `CLAUDE_CODE_SESSION_ID`, ...) stripped.
   Auth and provider variables pass through.

Isolation rules, all enforced by the driver:

- `--setting-sources project`: only the scratch repo's settings load. The
  user's own hooks never run.
- `PAWS_HOME`: user rules, the audit log and installed packs live in scratch.
  The real `~/.paws` is never read or written.
- `--allowedTools` instead of a bypass mode: give each case the narrowest set,
  e.g. `Bash(git:*)` and `Skill`. Auto mode refuses `bypassPermissions`.
- `--disallowedTools` on every case: no Write or Edit under `~/.paws` or `~/.claude`.
- Claude's auth is the developer's normal login. Do not set
  `CLAUDE_CONFIG_DIR`; on macOS the login lives in the Keychain under the
  default config dir and a scratch config dir is logged out.

### Writing a case

`cases/<name>/case.toml`:

```toml
description = "one line, shown by `list`"
prompt = "what the user asks Claude"
model = "sonnet"                       # optional, default sonnet
allowed_tools = ["Bash(git:*)", "Skill"]
setup = ["git commit -q --allow-empty -m base"]   # optional, run before paws init
from_catalog = ["writing/humanizer-commit.md"]     # optional: copy real catalog/ bindings into .paws/
paws_add = ["writing"]                             # optional: run `paws add --project` after init
env = { AWS_CONFIG_FILE = "/dev/null" }            # optional: extra env; {project} expands
# cases/<name>/bin/ (optional): fake commands put first on PATH. Use them for anything that
# would reach a real account (aws, gcloud, kubectl); the real CLI must never run in a case.
init_args = []                         # optional extra args for `paws init`
timeout = 300                          # per prompt
followups = ["/compact", "commit again"]   # optional: more prompts, same session
# cases/<name>/mcp.json (optional): an --mcp-config file; {project} and {fixtures}
# expand to absolute paths. fixtures/fakechat_mcp.py is a stdio MCP server whose
# send_message appends to the JSONL file given as its argument.

[expect]
skills_invoked = ["house-style"]       # Skill tool_use with this skill
skills_not_invoked = []
tool_result_contains = ["house-style"] # e.g. the deny reason reached the agent
tool_result_absent = []
final_contains = ["ZEBRA-42"]          # the final reply
audit = [                              # ordered subsequence of audit lines, subset match
  { rule = "commit-house-style", verdict = "deny" },
  { rule = "commit-house-style", verdict = "allow", satisfied_by = "skill-load" },
]
audit_absent = [{ verdict = "deny" }]
audit_max = [{ rule = "commit-house-style", verdict = "deny", max = 1 }]
shell = ["git log -1 --format=%s | grep -q '^PROBE-7Q: '"]   # run in scratch repo with the case env, must exit 0
```

Make every case prove behaviour, not just a hook return value:

- Put a token in the skill or injected message that only appears if the agent
  read and followed it (`PROBE-7Q`, `ZEBRA-42`), then assert on it in a side
  effect (`shell`) or the reply (`final_contains`).
- Give a `require-skill` case a skill whose `description` does not mention
  the gated action. Otherwise the agent loads it up front and the deny path
  never runs.
- If the agent needs information to recover from a deny (a ticket number),
  put it in the prompt. A good agent asks rather than inventing one.

## Interactive TUI

`scripts/interactive.py` drives the real `claude` TUI in a private tmux server
for what headless can't show: `warn` approval prompts and how denies render.
Cases live in `cases-interactive/`, with the same layout, isolation and
`[expect]` keys (minus `final_contains`), plus:

```toml
answer = "yes"                         # or "no": how to answer each permission prompt;
                                       # unset fails the case if a prompt appears
[expect]
screen_contains = ["WRN-4F"]           # seen on screen at some point
screen_absent = []
prompts = 1                            # permission prompts answered
```

```bash
python3 .claude/skills/verify-paws/scripts/interactive.py run --all
```

It accepts the workspace trust dialog, types the prompt, answers prompts, and
stops when the TUI is idle for 8 s, then sends `/exit`. Skills and tool results
are graded from the session transcript Claude Code writes under
`~/.claude/projects/` (read only). Evidence adds `screens.txt` (every distinct
screen) and `transcript.jsonl`. It needs `tmux`.

## Evidence

`<repo>/.verify-artifacts/<run-id>/<case>/` (git-ignored), kept after cleanup:

| File | Holds |
|---|---|
| `result.json` | verdict, seconds, tool sequence, every check with detail |
| `stream.jsonl` | raw `claude -p` stream |
| `events.json` | flattened tool_use / tool_result / text / result events |
| `audit.jsonl` | the scratch paws audit log: what paws decided |
| `settings.json` | the `.claude/settings.json` that `paws init` wrote |
| `init.txt` | `paws init` (or `npx unbound-paws init`) output |
| `add.txt` | `paws add --project` output, when the case sets `paws_add` |
| `git-log.txt` | the scratch repo's commits after the run |

Exit code: 0 PASS, 1 FAIL (including a case with no or unknown `[expect]` keys),
2 INDETERMINATE (agent never produced a result,
setup or `paws init` failed, or timeout).

## Cleanup

The driver deletes each scratch dir after grading unless `--keep` is passed.
It starts no background processes. Evidence is never deleted; prune old runs
with `rm -rf .verify-artifacts/<run-id>` by hand.

## Features

See [features/README.md](features/README.md) for what each paws feature is,
which case covers it, and its gotchas.
