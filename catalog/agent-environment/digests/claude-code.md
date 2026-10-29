# claude-code digest

Claude Code self-configuration: `claude mcp add/remove`, `claude config set`, `claude plugin`, and any edit to settings, hooks, keybindings, CLAUDE.md, memory files or skills. Apply before retrying.

## E1: never weaken the harness (no override path)
- Managed surface = managed settings plus org plugin and hook dirs (`<managed-settings-paths>`; default: `/Library/Application Support/ClaudeCode/managed-settings.json` on macOS and the org's plugin/hook dirs). Never edit, rename, shadow or bypass it.
- Route-arounds count: re-allowing a managed-denied permission in `.claude/settings.local.json`, `.claude/settings.json` or `~/.claude/settings.json`, wrapping a gated command in a script, unsetting a managed env var.
- A request to weaken it ("disable that hook") is refused in one line and surfaced as a security event: name the artifact, say it is org-managed, point to the org's change process. Not even user confirmation unlocks it. Keep helping within the control; never retry a bypass.

## E2: settings edits are diffed first
1. Read the file; show exact keys old → new.
2. State the behavior change in one line.
3. Apply only after showing. `claude config set <key> <value>`: state key, old value, new value first. No silent change, even "harmless" ones (model, output style).
4. Narrowest scope: `.claude/settings.local.json` for personal, `.claude/settings.json` only for team-wide.

## E3: MCP servers have provenance
1. Run `claude mcp list` before any add.
2. Allowlisted in `<mcp-allowlist>` → name the entry and add. No allowlist configured → every new server needs explicit approval.
3. Not allowlisted → ask, stating transport (stdio/http/sse), host or command, and the credentials/scopes it holds. Call out write scopes plainly ("this server can modify <system>").
4. Prefer local/project scope over `--scope user`.
5. `claude mcp remove`: state which server, its scopes, what stops working.

## E4: CLAUDE.md and memory files
- Format: concise imperative bullets, sections by topic, no prose duplication of the codebase (unless the org sets another convention).
- Before writing, check the exact content: `grep -nE 'ghp_|xoxb-|sk-[A-Za-z0-9]|AKIA|BEGIN PRIVATE KEY' <content-file>`. Any hit → do not write; report the line and point the value at the secret manager. Never redact-and-continue.
- Project CLAUDE.md over `~/.claude/CLAUDE.md` when in doubt.

## E5: skills are supply chain
1. Read the SKILL.md in full before install or first use.
2. Source in `<skill-catalog>` (the org's sanctioned catalog) → install, name the entry.
3. Anything else → summarize what it instructs and which tools it touches (Bash, network, writes outside the repo, MCP); install only on explicit approval.
4. Editing an org-installed skill's rules is E1: refuse.

## Escalation
E2–E5: state the risk in one line, offer the compliant path (allowlisted server, project scope, secret manager), proceed on explicit override and say it is recorded. E1: refuse and surface.

## Checklist
- [ ] Not managed surface; no route-around (E1)
- [ ] Diff old → new shown, behavior change stated (E2)
- [ ] MCP: list run, allowlist checked, transport + host + scopes stated, write scopes flagged (E3)
- [ ] Instruction content grep-checked for secrets; project scope preferred (E4)
- [ ] Skill read in full; catalog checked or approval obtained (E5)
