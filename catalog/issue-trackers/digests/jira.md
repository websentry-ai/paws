# jira digest

Jira writes: `jira issue create/edit/move/delete/link/comment add`, `jira epic add/remove`, `jira sprint add`, Jira REST, or Atlassian MCP (`createJiraIssue`, `editJiraIssue`, `transitionJiraIssue`, `addCommentToJiraIssue`). Site `<jira-site>` and project `<project-key>` default to the jira CLI's configured server and project; say which you used. Apply before retrying.

## T1: search before create
- JQL dedup on summary keywords + component, state the result:
  `jira issue list --jql "project = <project-key> AND summary ~ \"<kw1> <kw2>\" AND component = \"<component>\" AND statusCategory != Done" --plain`
  Empty → broaden: drop the component, then `text ~ "<keywords>"`.
- Exact duplicate → don't create; link or comment on the existing key. Near-match → show candidates, pick the obvious one, state it; ask only if genuinely ambiguous. No match → create and report the JQL.

## T2: org format
- Right type: Bug for defects, Task for chores, Story for user-facing scope.
- Body from the per-type template (Bug: Steps / Expected / Actual / Environment; Task/Story: Context / Acceptance criteria); fill every section or delete the heading.
- `jira issue create -tTask -s"<summary>" -b"<template body>" -C"<component>" -l"<label>" --no-input`
- Required fields, labels, components at creation. Unknown value → use the obvious one and say so; ask only when a wrong guess costs something. Never "TBD". Custom fields: REST `PUT https://<jira-site>/rest/api/3/issue/<KEY>`.

## T3: deliberate transitions
1. `jira issue view <KEY> --plain` (current status).
2. Allowed transitions: `curl -s -u "$JIRA_USER:$JIRA_TOKEN" "https://<jira-site>/rest/api/3/issue/<KEY>/transitions" | jq '.transitions[] | {id, name}'`.
3. `jira issue move <KEY> "<Transition Name>"` only to a listed transition (or the user's named target).
- Close/resolve: state "Moving <KEY> from In Progress to Done" and log `jira issue comment add <KEY> "Resolved: <what resolved it>. [created-by: ai-agent]"`. Ask first only if someone else owns the ticket.
- Bulk moves: list keys and from → to as a set first, then one by one.

## T4: everything linked
- Parent in the same session: `jira epic add <EPIC-KEY> <ISSUE-KEY>`; other relations `jira issue link <K1> <K2> "Relates"` (or Blocks, Duplicate).
- No parent named → link the epic in session context and state it; none fits → ask. Never silently unlinked.
- Branch `<KEY>-<slug>`, commit `<KEY>: <message>`; comment the PR URL on the ticket once it exists.

## T5: honest estimates and attribution
- Never set story points, estimates or due dates unless the user gave the value; leave empty and say so.
- Assignee is a human (default: the requesting human), never a bot: `jira issue edit <KEY> -a"<human-account>" --no-input`.
- Agent-created tickets carry `[created-by: ai-agent]` (description suffix or label).

## MCP
Same sequence: JQL dedup first, template body in the create payload, transitions read before `transitionJiraIssue`, comments in the org's voice with attribution.

## Escalation
Rule vs user intent → state the risk in one line and proceed (audit-logged).

## Checklist
- [ ] Dedup JQL run; result stated
- [ ] Type correct; template body; required fields filled, no "TBD"
- [ ] Status read; move is a listed transition; close stated with resolution comment
- [ ] Epic/parent linked; PR/commit cross-referenced
- [ ] No invented estimates; human assignee; attribution set
