---
name: linear
description: 'Mandatory guardrails for every Linear write operation (save_issue, save_project, save_comment, save_milestone, save_initiative, delete_comment) via the Linear MCP server. Enforces the issue-trackers baseline with Linear mechanics: list_issues dedup before create, statuses from list_issue_statuses, one-line state-and-proceed transitions, and project/parent linkage at create time. Triggers PROACTIVELY on any Linear MCP write — do NOT wait for the user to invoke it.'
---

# Linear

Linear mechanics for the issue-trackers baseline (T1–T5). Primary surface is the Linear MCP server — invocations below are MCP tool names with their key parameters. Apply on every write: narrate in one line and proceed; ask only where a rule says ask; confirm nothing that isn't destructive.

## Dedup search (implements T1)

Before any `save_issue` create (payload without an existing `id`):

    list_issues { query: "<summary keywords>", teamId: "<team-id>" }
    # empty result → broaden: drop teamId, retry with the two strongest keywords

Exact duplicate → don't create; `save_comment { issueId: "<existing>" }` on it instead. Near-match → show the candidates and ask "use existing or create new". No match → create, stating the query in one line so a miss is spottable.

## Issue shape (implements T2)

- Resolve IDs before the write — `list_teams`, `list_projects`, `list_issue_labels` — never guess an ID or label name. Team: {{LINEAR_TEAM}}.
- `save_issue` create sets `teamId`, `projectId`, and `labelIds` in the same call; description follows the org's per-type template ({{TICKET_TEMPLATE_REGISTRY}}) — every section filled or its heading removed.
- Workflow states come from `list_issue_statuses { teamId }` — never invent a state name. Unknown required value → ask, never "TBD".

## Status transitions (implements T3)

1. `get_issue { id }` — read current state and assignee first.
2. `save_issue { id, state: "<name from list_issue_statuses>" }` — state the move in one line and proceed: "Moving ENG-123 In Progress → In Review."
3. Closing or canceling an issue assigned to someone else → ask first; it's their ledger entry. Own or agent-created issues: state, close, and add a resolution `save_comment` saying what resolved it.
4. Multiple moves: one line listing keys and from → to, then proceed as a set.

## Linkage (implements T4)

- Every created issue carries `projectId` or `parentId` at create time ({{DEFAULT_EPIC_POLICY}}); nothing fits → ask which project — never leave it orphaned silently.
- PR ↔ issue runs both ways: `create_attachment { issueId, url: "<PR URL>", title: "<PR title>" }`, and the branch/commit carries the issue identifier (`eng-123-<slug>`).
- Milestones via `save_milestone` attach to their project, never free-floating.

## Assignee, estimates, attribution (implements T5)

- Never set `estimate`, `priority`, or `dueDate` unless the user supplied the value ({{ESTIMATION_POLICY}}) — an empty estimate is honest.
- `assignee` is a human ({{DEFAULT_ASSIGNEE}}), resolved via `list_users` — never the agent's own service account.
- Agent-created issues and comments append {{AGENT_ATTRIBUTION_CONVENTION}}; add label {{AGENT_LABEL}} where it exists in `list_issue_labels`, otherwise skip and say so.
- `delete_comment` only on comments this agent created, stated in one line first — deleting a human's comment is a never-event.

## Escalation

Rule conflicts with user intent ("skip the dedup, just file all 15"): one line of risk ("15 unswept creates risk split discussions across duplicates"), offer the compliant path (one `list_issues` sweep, create only the misses), proceed on explicit override — noting the override is recorded.

## Pre-flight checklist (before every Linear write)

- [ ] `list_issues` dedup run for creates; result stated in one line (T1)
- [ ] IDs resolved via `list_teams`/`list_projects`/`list_issue_labels`; state from `list_issue_statuses`, never invented (T2)
- [ ] Transition read-then-move, stated in one line; someone else's issue → ask before closing (T3)
- [ ] `projectId`/`parentId` set at create; PR attached to the issue (T4)
- [ ] No invented estimates; assignee is a human; attribution appended (T5)

## Category baseline: Issue Trackers

These rules apply to **every** tracker-writing tool. An instance skill (Jira, Linear, …) adds tool-specific mechanics on top; it may tighten these rules but never loosen them. The tracker is the org's work ledger: a duplicate ticket splits the discussion, an invented estimate corrupts sprint math, and an orphan ticket is work nobody will ever find.

### Rule T1: Search before create

Never create a ticket without first searching for an existing one:

1. Search on summary keywords plus the target component using the instance skill's exact query mechanics.
2. Exact duplicate → do not create; link or comment on the existing ticket instead.
3. Near-match (same symptom, overlapping keywords) → show the candidates, pick the obvious one, and state the choice; ask only when the match is genuinely ambiguous.
4. No match → create, and say what you searched so the user can spot a miss.

### Rule T2: Tickets follow the org format

A ticket is a contract with whoever picks it up next:

- Correct issue type for the work (bug vs task vs story) — never default blindly.
- Description body follows the org's template for that type ({{TICKET_TEMPLATE_REGISTRY}}); fill every template section or delete the heading — no empty scaffolding.
- Every org-required field, label, and component set at creation time; unknown required value → use the sensible value when context makes one obvious, and say so; ask only when a wrong guess has real cost. Never stub with "TBD".

### Rule T3: Workflow transitions are deliberate

Status changes drive sprints, reports, and automation — they are never a side effect:

1. Read the ticket's current status before changing it.
2. Move only along transitions the workflow actually allows; read the available transitions, never guess a target state.
3. Closing or resolving: state the move, and log a resolution note saying what resolved it. Ask first only when closing a ticket that belongs to someone else.
4. Never bulk-transition silently — multiple moves get narrated as a set (keys, from → to) before executing.

### Rule T4: Everything is linked

An unlinked ticket is invisible to planning:

- Every created ticket is linked to its epic, parent, or project per the org's policy ({{DEFAULT_EPIC_POLICY}}) in the same session it is created.
- PRs and commits cross-reference the ticket key, and the ticket references the PR — traceability runs both ways.
- No orphan tickets: no parent evident → link the epic/parent in session context and state it; genuinely none fits → ask. Never leave a ticket unlinked silently.

### Rule T5: Honest estimates and attribution

- Never invent story points, time estimates, or due dates ({{ESTIMATION_POLICY}}); an empty estimate field is honest, a guessed one poisons velocity.
- The assignee is always a human ({{DEFAULT_ASSIGNEE}}), never a bot or service account.
- Every agent-created ticket carries the org's agent-attribution convention ({{AGENT_ATTRIBUTION_CONVENTION}}) so triage can distinguish agent writes.

### Pre-flight checklist (run before every tracker write)

- [ ] Dedup search done on summary keywords + component; result stated (T1)
- [ ] Correct issue type; org template applied; required fields/labels/components filled, no "TBD" (T2)
- [ ] Transitions: current status read, target valid, close/resolve stated with a resolution note — asked first only for someone else's ticket (T3)
- [ ] Ticket linked to epic/parent; PRs/commits cross-referenced (T4)
- [ ] No invented estimates; assignee is a human; attribution set (T5)
