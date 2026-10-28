# The paws skill

## Sub-features

- `skills/paws/SKILL.md`, embedded in the binary; `paws init` installs it beside the
  settings file it wrote (`.claude/skills/paws`, or `~/.claude/skills/paws` with `--global`).
- Teaches the rule format, the steps from request to rule, and the `paws explain` check.
- Never replaces a `paws` skill the user owns or edited.

## How to get to it (user POV)

Ask Claude Code: "always load humanizer before I commit". It loads the paws
skill, writes `.paws/...md`, and checks it with `paws explain`.

## Driving it with drive.py

`run paws-skill-writes-rule`. Proof: the paws skill was invoked, a rule file
exists under `.paws/`, `paws explain 'git commit -m wip'` denies naming
humanizer, and `git status` is untouched.

`run paws-skill-*` (26 more cases) each give one plain request and grade the rule
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
| `paws-skill-pre-push` | verification skill before git push | push, chained, `-C` → DENY; status, echoed, commit message → allow |
| `paws-skill-prompt-inject` | remind about smoke tests when a prompt mentions deploying | deploy prompts → context; other prompts → allow |
| `paws-skill-session-start` | load karpathy-guidelines every session | startup, clear, compact → context naming the skill |
| `paws-skill-tool-list-glob` | db-migrations before changes under migrations/ | Edit/Write → DENY; Read, other paths → allow |
| `paws-skill-mcp-delete-warn` | ask before Linear MCP deletes | `mcp__linear__delete_*` → ASK; list, save, GitHub delete → allow |
| `paws-skill-digest` | put docs/s3-rules.md in the bucket-create deny | create-bucket, mb → `paws digest` deny with the rules; `s3 ls` → allow |
| `paws-skill-exec-script` | script refuses curl/wget piped into a shell | `paws gate` denies pipe-to-sh/bash/sudo bash; allows downloads and `| jq` |
| `paws-skill-repo-marker-user` | user-level warn on terraform apply/destroy where terraform/ exists | ASK in the repo; allow in a repo without it; no repo rule |
| `paws-skill-new-file` | karpathy-guidelines before a new .py file | Write new → DENY; Write/Edit existing, .md → allow |
| `paws-skill-no-edit-tdd` | TDD skill before Go edits until a test was edited | .go → DENY; `_test.go` → allow; allow once a transcript shows a test edit |
| `paws-skill-agent-count` | skill after more than 3 subagents in a turn | 4 Agent calls → DENY; 3, or calls before the prompt → allow |
| `paws-skill-vague-delete` | "stop the agent deleting stuff" | no rule, or one that stops `rm -rf` and spares `ls`, `git status`, `cat`, `echo` |
| `paws-skill-prefixed-forms` | release-checklist before npm publish | chained, env prefix, `sudo` → DENY; quoted, `npm run publish-docs`, `npm pack` → allow |
| `paws-skill-no-lookahead` | block `--force`, allow `--force-with-lease` | force, `-f`, chained → DENY; with-lease → allow; rule parses |
| `paws-skill-edit-existing` | extend the humanizer rule to more commands | still one humanizer rule; old and new commands → DENY |
| `paws-skill-two-rules` | block .env reads and audit docker run | both rules written and matching |
| `paws-skill-cant-rewrite` | rewrite commit messages | the reply says paws can't rewrite; no broken rule |
| `paws-skill-self-block` | block DROP TABLE in any case | the agent's own check is denied; the rule still matches all cases and spares `select` |

## Gotchas

- The case allows `Bash(paws:*)` so the agent can run `paws explain` itself. A chained
  or `||` command around `paws` needs approval and fails headless; the skill tells the
  agent to run one plain `paws explain` per call.
- The driver always passes `--disallowedTools` for writes under `~/.paws` and `~/.claude`,
  so a user-level request can only land in the scratch `$PAWS_HOME`.
- `paws explain` has no transcript, so `agent-count` and `no-edit-matching` cases grade
  with `fixtures/explain_tx.py`, which writes a made-up transcript and runs
  `paws explain -`. Shell checks get its directory as `$VERIFY_FIXTURES`.
- `paws explain` never runs `exec-script`; that case pipes a hook event into `paws gate`.
