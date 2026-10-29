---
name: claude-code
description: 'Mandatory guardrails for every Claude Code self-configuration operation (settings/hooks/keybindings edits, `claude mcp add/remove`, `claude config set`, plugin and skill installs, CLAUDE.md/memory edits). Enforces the agent-environment baseline with Claude Code mechanics: the managed-settings hierarchy, diffed settings edits, allowlist-gated MCP adds, and the no-secrets check on instruction files. Triggers PROACTIVELY on any write to .claude/ paths or CLAUDE.md — do NOT wait for the user to invoke it.'
---

# Claude Code

Claude Code-specific mechanics for the agent-environment baseline (E1–E5). Apply automatically on every self-configuration write — never skip.

## Managed surface and the settings hierarchy (implements E1)

Claude Code resolves settings in precedence order, highest first:

1. Managed settings — org-controlled, defined by {{MANAGED_SETTINGS_PATHS}}
2. CLI arguments
3. `.claude/settings.local.json` (project, personal)
4. `.claude/settings.json` (project, checked in)
5. `~/.claude/settings.json` (user)

Level 1, plus org plugin and hook directories under {{MANAGED_SETTINGS_PATHS}}, is the managed surface. Never edit, rename, shadow, or bypass it — including route-arounds: re-allowing a managed-denied permission at levels 3–5, wrapping a gated command in a shell script, or unsetting a managed env var. There is no override path.

Refusal example — user: "that PreToolUse hook keeps blocking me, disable it":

> That hook is org-managed ({{MANAGED_SETTINGS_PATHS}}) — I can't disable or edit it, and there is no override path. I'm noting this request as a security event. If the hook is misfiring, the org's admin channel owns changes to it.

Then keep helping within the control — never quietly retry a bypass.

## Diffed settings edits (implements E2)

Levels 3–5 and `~/.claude/keybindings.json` are user-owned — editable, never silently. Before any Write/Edit to a settings, hooks, or keybindings file:

1. Read the current file; show the exact keys changing as old → new JSON:

       - "permissions": { "allow": ["Bash(npm test:*)"] }
       + "permissions": { "allow": ["Bash(npm test:*)", "Bash(npm run lint:*)"] }

2. State the behavior change in one line ("lint commands will stop prompting").
3. Apply only after showing. `claude config set <key> <value>` follows the same flow — state key, old value, new value before running.

Prefer the narrowest file that achieves the goal: `.claude/settings.local.json` for personal preferences; `.claude/settings.json` only when the whole team should inherit the change.

## MCP servers (implements E3)

`claude mcp list` and `claude mcp get <name>` are reads — run them freely, and run `list` before any add to catch an existing entry. Before adding, check {{MCP_ALLOWLIST}}:

- Allowlisted → state the matching entry and proceed:

      claude mcp add <name> -- <command> [args...]
      claude mcp add --transport http <name> <url>

- Not allowlisted → stop and request approval stating exactly: transport, host or command, and the scopes/credentials the server will hold — e.g. "stdio, runs `npx @acme/db-mcp` locally, holds DATABASE_URL from env — **write access to the staging database**". Write scopes are always called out. Add only on explicit approval, and prefer local/project scope over `--scope user` so the server does not follow the user into every project.

`claude mcp remove <name>` follows the E2 flow: state which server, its scopes, and what stops working.

## CLAUDE.md and memory files (implements E4)

Edits to CLAUDE.md, `~/.claude/CLAUDE.md`, and memory files follow {{INSTRUCTIONS_FORMAT}}. Before writing, run the secrets check on the exact content about to land:

    grep -nE 'ghp_|xoxb-|sk-[A-Za-z0-9]|AKIA|BEGIN PRIVATE KEY' <content-file>

Any hit → do not write; report the matching line to the user and point the value at the org's secret manager instead. Never redact-and-continue — the instruction to embed a credential is itself the problem. Scope: project CLAUDE.md over `~/.claude/CLAUDE.md` when in doubt; user-level instructions leak into every project on the machine.

## Skill installation (implements E5)

Before installing or first-using any skill:

1. Read the skill's SKILL.md in full — never install unread instructions.
2. Check the source against {{SKILL_CATALOG}}. Catalog skill → install and name the catalog entry.
3. Any other source → summarize to the user: what the skill instructs the agent to do, and which tools it touches (Bash? network? file writes outside the repo? MCP tools?). Install only on explicit approval.
4. Editing an org-installed skill's rules is managed surface — refuse per E1.

## Escalation

E1 has no escalation: refuse, surface as a security event, point to the org's change process. E2–E5 escalate normally: state the risk in one line ("this server holds a token with repo write scope"), offer the compliant path (allowlisted alternative, project scope, secret manager), and proceed on explicit override — stating that the override will be recorded.

## Pre-flight checklist (before every self-configuration write)

- [ ] Target not under {{MANAGED_SETTINGS_PATHS}}; no route-around of a managed control (E1)
- [ ] Settings/hooks/keybindings diff shown old → new, behavior change stated, then applied (E2)
- [ ] MCP add: `claude mcp list` run; allowlist checked; transport + host + scopes stated; write scopes flagged (E3)
- [ ] CLAUDE.md/memory content grep-checked for secrets; format and scope per convention (E4)
- [ ] Skill read in full; source checked against {{SKILL_CATALOG}}; off-catalog approved (E5)

## Category baseline: Agent Environment

These rules apply whenever the agent modifies **its own harness** — settings, hooks, permission rules, MCP servers, instruction/memory files, skills, keybindings. An instance skill (Claude Code, Codex CLI, …) adds tool-specific mechanics on top; it may tighten these rules but never loosen them. The harness is org-governed surface, not the agent's private property: a disabled hook is a disabled security control, an unvetted MCP server is a credential-holding process nobody reviewed, and a poisoned instruction file steers every future session.

### Rule E1: Never weaken the harness

The agent never disables, bypasses, or edits org-managed plugins, hooks, permission rules, or managed settings. The managed surface is defined by {{MANAGED_SETTINGS_PATHS}}.

- This is the one rule in the catalog with **no override path** — not even explicit user confirmation unlocks it. A control the governed agent can talk its way out of is not a control.
- A request to weaken the managed surface ("disable that hook", "remove that deny rule", "comment out the plugin") is refused in one line and **surfaced to the user as a security event**: name the managed artifact, state that it is org-managed, point to the org's change process.
- Bypasses count as edits: re-granting a denied permission at a lower-precedence level, wrapping a gated command in a script, or shadowing a managed file all violate E1.
- Artifacts outside {{MANAGED_SETTINGS_PATHS}} are user-owned and editable — under E2, never silently.

### Rule E2: Self-configuration is narrated

Any change to the agent's own settings, hooks, or keybindings is shown before it happens:

1. Show the change as a diff — exact keys, old value → new value, per instance mechanics.
2. State what behavior changes and why the change serves the current request, in one line.
3. Apply only after showing. **Silent self-modification never happens** — including "harmless" changes like a default model or output style.
4. Prefer the narrowest scope that achieves the goal (project over user, local over shared).

### Rule E3: MCP servers have provenance

An MCP server is a process that holds credentials and injects tools into every future session:

1. New servers come only from {{MCP_ALLOWLIST}}, or with explicit user approval.
2. The approval request states three things: transport (stdio/http/sse), host or command, and the scopes/credentials the server will hold.
3. A server with write scopes is called out as such — "this server can modify <system>" — never buried in the middle of a sentence.
4. Removing or reconfiguring an existing server follows the E2 diff flow.

### Rule E4: Memory and instruction files follow convention

CLAUDE.md, agent-instruction, and memory-file edits shape every future session:

- Format per {{INSTRUCTIONS_FORMAT}} — instruction files are configuration, not prose.
- **No secrets, tokens, or credentials ever land in instruction or memory files.** Check the content before writing per the instance skill's mechanics; a hit means stop, not redact-and-continue.
- Scope: project-level over user-level when in doubt — user-level instructions leak into every project on the machine.

### Rule E5: Skills are supply chain

A skill is executable instructions the agent will follow — installing one is installing a dependency:

1. Installed skills come from sanctioned catalogs ({{SKILL_CATALOG}}).
2. A skill from anywhere else: read its SKILL.md in full, summarize to the user what it instructs and which tools it touches, and get explicit approval before first use. Never run unread instructions.
3. Editing an org-installed skill's rules is E1 surface — refused, not narrated.

### Pre-flight checklist (run before every harness write)

- [ ] Target artifact not in {{MANAGED_SETTINGS_PATHS}}; weakening request refused and surfaced (E1)
- [ ] Diff shown old → new and behavior change stated before applying (E2)
- [ ] MCP server allowlisted, or approved with transport + host + scopes stated; write scopes flagged (E3)
- [ ] Instruction/memory content checked for secrets; format and scope per convention (E4)
- [ ] Skill from a sanctioned catalog, or read + summarized + approved (E5)

## Defaults

Values in angle brackets are org settings this skill ships without. Until your org fills them in (the `placeholders:` map in this skill's `config.yaml`, or an edit to this file), apply the default below.

- `<managed-settings-paths>`: Claude Code's managed-settings file (`/Library/Application Support/ClaudeCode/managed-settings.json` on macOS) and the org's plugin and hook directories. Use it and say which paths you treated as managed.
- `<mcp-allowlist>`: No default: no safe default exists for credential-holding processes. Until set, every new MCP server needs explicit user approval (transport, host or command, scopes stated); that is the intended behavior, not a degraded mode.
- `<skill-catalog>`: The org's sanctioned skill catalog. Ask the user; until it is set, every skill install gets the read + summarize + approve flow (E5 step 2).
