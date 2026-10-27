---
name: jira
description: 'Mandatory guardrails for every Jira write operation (issue create, edit, transition, link, epic assignment, sprint moves, comments) via the jira CLI (ankitpokhrel/jira-cli), Jira REST curl calls, or Atlassian MCP tools. Enforces the issue-trackers baseline with Jira mechanics: JQL dedup before create, template-conformant descriptions, transition reads before move, and epic linking. Triggers PROACTIVELY on any `jira issue`/`jira epic`/`jira sprint` write — do NOT wait for the user to invoke it.'
---

# Jira

Jira-specific mechanics for the issue-trackers baseline (T1–T5). Apply automatically on every Jira write — never skip. Primary surface: the `jira` CLI (ankitpokhrel/jira-cli) and Jira REST via curl against {{JIRA_SITE}}; the Atlassian MCP tools follow the same rules. CLI commands target project {{JIRA_PROJECT_KEY}}; add `-p{{JIRA_PROJECT_KEY}}` where accepted.

## Dedup search (implements T1)

Before any `jira issue create`, run a JQL dedup query on summary keywords + component and state the result:

    jira issue list --jql "project = {{JIRA_PROJECT_KEY}} AND summary ~ \"<keyword1> <keyword2>\" AND component = \"<component>\" AND statusCategory != Done" --plain

    # If empty, broaden: drop the component clause, then search description text
    jira issue list --jql "project = {{JIRA_PROJECT_KEY}} AND text ~ \"<keywords>\"" --plain

Exact duplicate → do not create; `jira issue link` or comment on the existing key instead. Near-match → show candidate keys + summaries, use the obvious one, and state the choice; ask only when genuinely ambiguous. No match → create and report the JQL you ran.

## Ticket format (implements T2)

Issue type matches the work — Bug for defects, Task for chores, Story for user-facing scope. Description body comes from the org template for that type ({{TICKET_TEMPLATE_REGISTRY}}), every section filled or its heading removed:

    jira issue create -tTask -s"<summary>" -b"$(cat <<'EOF'
    <org template body — sections filled, no empty scaffolding>
    EOF
    )" -C"<component>" -l"<label>" --no-input

Required fields, labels, and components set at creation; unknown required value → use the obvious value from context and say so; ask only when a wrong guess has real cost — never "TBD". Custom fields the CLI lacks a flag for go via REST:

    curl -s -u "$JIRA_USER:$JIRA_TOKEN" -X PUT "https://{{JIRA_SITE}}/rest/api/3/issue/<KEY>" \
      -H "Content-Type: application/json" -d '{"fields": {"customfield_NNNNN": "<value>"}}'

## Transitions (implements T3)

    # 1. Read current status
    jira issue view <KEY> --plain

    # 2. Read the transitions the workflow actually allows from here
    curl -s -u "$JIRA_USER:$JIRA_TOKEN" "https://{{JIRA_SITE}}/rest/api/3/issue/<KEY>/transitions" | jq '.transitions[] | {id, name}'

    # 3. Move only to a listed transition (or the user's explicitly named target)
    jira issue move <KEY> "<Transition Name>"

Closing/resolving (Done, Closed, Resolved, Won't Do): state the move — "Moving <KEY> from In Progress to Done" — and log the resolution; ask first only when the ticket is assigned to someone else:

    jira issue comment add <KEY> "Resolved: <what resolved it>. {{AGENT_ATTRIBUTION_CONVENTION}}"

Never bulk-transition silently: multiple moves are listed (keys, from → to) as a set, then executed one by one.

## Linking (implements T4)

Every created ticket gets its parent in the same session ({{DEFAULT_EPIC_POLICY}}):

    jira epic add <EPIC-KEY> <ISSUE-KEY>

    # Non-epic relations
    jira issue link <KEY-1> <KEY-2> "Relates"    # or "Blocks", "Duplicate"

PRs and commits carry the key (branch `<KEY>-<slug>`, commit `<KEY>: <message>`), and the ticket gets a comment with the PR URL once it exists. No parent named → link the epic in session context and state it; genuinely none fits → ask. Never leave the ticket unlinked silently.

## Estimates and attribution (implements T5)

- Never set story points, time estimates, or due dates unless the user supplied the value ({{ESTIMATION_POLICY}}). Missing estimate → leave the field empty and say so.
- Assignee is always a human — default {{DEFAULT_ASSIGNEE}}, never a bot or service account:

      jira issue edit <KEY> -a"<human-account>" --no-input

- Every agent-created ticket carries {{AGENT_ATTRIBUTION_CONVENTION}} (description suffix or label) so triage can distinguish agent writes.

## MCP surface (Atlassian)

`createJiraIssue`, `editJiraIssue`, `transitionJiraIssue`, and `addCommentToJiraIssue` follow the same T1–T5 sequence: JQL dedup first, template body in the create payload, transitions read before `transitionJiraIssue`, and comments written in the org's voice with attribution appended.

## Escalation

Rule conflicts with user intent → state the risk in one line and proceed; the override lands in the audit log.

## Pre-flight checklist (before every Jira write)

- [ ] JQL dedup run on summary keywords + component; result stated (T1)
- [ ] Issue type correct; description from the org template for its type; required fields/labels/components filled, no "TBD" (T2)
- [ ] Current status read; move is in the transitions list; close stated with a resolution comment — asked first only for someone else's ticket (T3)
- [ ] Epic/parent linked; PR/commit cross-referenced (T4)
- [ ] No invented estimates; assignee is a human; attribution set (T5)

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

## Defaults

Values in angle brackets are org settings this skill ships without. Until your org fills them in (the `placeholders:` map in this skill's `config.yaml`, or an edit to this file), apply the default below.

- `<jira-site>`: The jira CLI's configured server (from `~/.config/.jira/.config.yml`), e.g. `acme.atlassian.net`. Use it and say which site you used.
- `<project-key>`: The jira CLI's configured default project. Use it and say which project you used.
