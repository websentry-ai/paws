# linear digest

Linear MCP writes: `save_issue`, `save_project`, `save_comment`, `save_initiative`, `delete_comment` (and `save_milestone`). Narrate in one line and proceed; ask only where a rule says ask. Apply before retrying.

## T1: search before create
- Before a `save_issue` create (no `id`): `list_issues { query: "<summary keywords>", teamId: "<team-id>" }`; empty → drop `teamId`, retry with the two strongest keywords.
- Exact duplicate → don't create; `save_comment { issueId: "<existing>" }` instead. Near-match → show candidates, ask "use existing or create new". No match → create, stating the query in one line.

## T2: issue shape
- Resolve IDs first with `list_teams`, `list_projects`, `list_issue_labels`; never guess an ID or label. Team: the team of the project the user is working in (ambiguous → ask once per session, then reuse).
- Create sets `teamId`, `projectId`, `labelIds` in one call. Description uses the per-type sections (Bug: Steps / Expected / Actual / Environment; Task/Story: Context / Acceptance criteria); fill each or remove its heading.
- States only from `list_issue_statuses { teamId }`. Unknown required value → ask, never "TBD". Pick the right issue type for the work.

## T3: transitions
1. `get_issue { id }`: read state and assignee.
2. `save_issue { id, state: "<name from list_issue_statuses>" }`, stated: "Moving ENG-123 In Progress → In Review."
3. Closing or canceling someone else's issue → ask first. Own or agent-created: state, close, add a resolution `save_comment` saying what resolved it.
4. Several moves: one line listing keys and from → to, then proceed as a set.

## T4: linkage
- `projectId` or `parentId` at create time: link to the project the user is working under; if none is evident, ask. Never orphaned silently.
- PR ↔ issue both ways: `create_attachment { issueId, url: "<PR URL>", title }`; branch/commit carries the identifier (`eng-123-<slug>`).
- Milestones attach to their project.

## T5: assignee, estimates, attribution
- Never set `estimate`, `priority` or `dueDate` unless the user gave the value; empty is honest.
- `assignee` is a human (default: the requesting human), resolved via `list_users`; never the agent's service account.
- Agent-created issues and comments append `[created-by: ai-agent]`; add label `ai-agent` if it exists in `list_issue_labels`, else skip and say so.
- `delete_comment` only on comments this agent created, stated first. Deleting a human's comment never happens.

## Escalation
Rule vs user intent ("skip the dedup, just file all 15"): one line of risk, offer the compliant path (one `list_issues` sweep, create only misses), proceed on explicit override and say it is recorded.

## Checklist
- [ ] `list_issues` dedup for creates; result stated
- [ ] IDs resolved; state from `list_issue_statuses`
- [ ] Read-then-move, stated; someone else's issue → ask before closing
- [ ] `projectId`/`parentId` at create; PR attached
- [ ] No invented estimates; human assignee; attribution appended
