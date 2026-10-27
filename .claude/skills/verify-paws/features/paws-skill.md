# The paws skill

## Sub-features

- `skills/paws/SKILL.md`, embedded in the binary; `paws init` installs it beside the
  settings file it wrote (`.claude/skills/paws`, or `~/.claude/skills/paws` with `--global`).
- Teaches the rule format, common request-to-rule mappings, and the `paws explain` check.
- Never replaces a `paws` skill the user owns or edited.

## How to get to it (user POV)

Ask Claude Code: "always load humanizer before I commit". It loads the paws
skill, writes `.paws/...md`, and checks it with `paws explain`.

## Driving it with drive.py

`run paws-skill-writes-rule`. Proof: the paws skill was invoked, a rule file
exists under `.paws/`, `paws explain 'git commit -m wip'` denies naming
humanizer, and `git status` is untouched.

## Gotchas

- The case allows `Bash(paws:*)` so the agent can run `paws explain` itself.
