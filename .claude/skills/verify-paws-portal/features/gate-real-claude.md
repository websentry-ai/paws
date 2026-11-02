# Gate with real Claude Code

Skills turned on in the portal reach a laptop where real Claude Code runs; when Claude goes to commit, paws
denies once, Claude reads the skill (or the rule text), retries, and the commit follows what it read. The
portal then counts the run.

## Sub-features

- `rule-digest` a short-rule skill (`caveman-commit`, `load: rule`) whose rule text carries a random token; the deny carries the text and the commit subject starts with the token.
- `skill-load` (isolated mode) `git` as a full skill: Claude calls the Skill tool for `git`; the retry passes on `skill-load`.
- `portal-count` the portal headline counts the run (from the async Stop hook, or `paws sync` if Claude exited first).

## How to get to it (user POV)

- Turn skills on in the portal, connect the laptop, ask Claude Code to commit.

## Driving it with portal.sh

Preconditions:

- `portal.sh up` (test mode) and `doctor` ok; `claude` on PATH and logged in. For isolated mode, a token at `~/.config/paws-verify/claude-token` (see SKILL.md); otherwise the fallback runs with the normal login and the rule skill only.

- **Setup.** Run `portal.sh run claude-gate` (or `VERIFY_SKIP_CLAUDE=1 portal.sh run claude-gate` to check setup without model calls). A scratch repo with a staged `notes.txt`, `paws init` (project scope), portal user `verify-claude` with the skills on, `paws login --key`, `paws sync`.
- **Run.** `claude -p "Commit the staged change…" --setting-sources project --output-format stream-json --verbose --allowedTools "Bash(git:*)" Skill`. The turn ends with a `result` event that is not an error.
- **Proof.** `claude-gate/audit.jsonl` has a deny on the `caveman-commit--*` rule; a tool result contains the token; `git-log.txt` starts with `<token>:`; isolated mode also has a `Skill` tool_use for `git` and a `git--*` allow with `satisfied_by: skill-load`; `GET /skills` (user `verify-claude`) headline `about_to_act > 0`.

## Gotchas

- A skill whose description mentions commits may be loaded up front, so the deny path never runs; the rule-text token is what proves Claude read the portal's text either way.
- The fallback still lets Claude Code write its own session transcript under `~/.claude/projects/` (Claude Code's behaviour, not paws's); only isolated mode leaves `~/.claude` untouched.
- macOS keeps the login in the Keychain for the default config dir; a scratch `CLAUDE_CONFIG_DIR` without the token is logged out.
