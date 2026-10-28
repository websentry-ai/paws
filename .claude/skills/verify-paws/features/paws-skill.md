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

`run paws-skill-*` (eight more cases) each give one plain request and grade the rule
the agent wrote with `paws explain`: commands that should match get the expected
verdict, near-misses get `allow`. Each also checks the agent ran `paws explain` itself.

| Case | Request | Graded on |
|---|---|---|
| `paws-skill-pr-gate` | load pr-voice before gh PR create/edit | create, edit, chained → DENY; `gh pr list`, `echo gh pr create` → allow |
| `paws-skill-block-rmrf` | never `rm -rf` | `-rf`, `-fr`, `-Rf`, `sudo`, chained → block; `rm file.txt`, `echo rm -rf` → allow |
| `paws-skill-warn-prod-psql` | ask before psql against prod | host flag, URL, env prefix → ASK; localhost, bare `psql` → allow |
| `paws-skill-slack-send` | load slack-tone before Slack MCP sends | send, reply → DENY; read, search, Gmail send → allow |
| `paws-skill-advise-tsx` | remind about docs/STYLE.md on .tsx edits | Edit/Write .tsx → context naming STYLE.md; Read .tsx, Edit .ts → allow |
| `paws-skill-audit-publish` | record every npm publish | `npm publish` → audit match, allowed; `npm install` → no match |
| `paws-skill-when-react` | react-patterns on .tsx/.jsx, React repos only | DENY with react in package.json, allow without |
| `paws-skill-user-level` | block force push in all my repos | rule under `$PAWS_HOME`, none in `.paws/`; fires from another repo |

## Gotchas

- The case allows `Bash(paws:*)` so the agent can run `paws explain` itself. A chained
  or `||` command around `paws` needs approval and fails headless; the skill tells the
  agent to run one plain `paws explain` per call.
- The driver always passes `--disallowedTools` for writes under `~/.paws` and `~/.claude`,
  so a user-level request can only land in the scratch `$PAWS_HOME`.
- A custom `deny_message` replaces the default block text, so graders match `DENY` and
  `action=block`, not "Blocked by paws rule".
