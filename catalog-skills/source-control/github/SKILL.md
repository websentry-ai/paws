---
name: github
description: 'Mandatory guardrails for every GitHub MCP write operation (create_or_update_file, push_files, create_branch, create_pull_request, merge_pull_request, delete_file, create_repository). Enforces the source-control baseline with GitHub API mechanics: org commit-message format, branch-before-write on protected refs, PR-only merges that never bypass required checks, a secret scan of file contents before every push, and honest agent attribution. Triggers PROACTIVELY on any GitHub MCP write tool — do NOT wait for the user to invoke it.'
---

# GitHub (MCP)

GitHub MCP mechanics for the source-control baseline (S1–S5) — the MCP-surface sibling of the `git` skill, which realizes the same rules on the git CLI. Same org conventions; only the invocations differ. Apply automatically on every GitHub MCP write — never skip. Reads (`get_*`, `list_*`, `search_*`) are unrestricted.

## Commit messages (implements S1)

The `message` parameter of `create_or_update_file`, `push_files`, and `delete_file` is a commit message and follows {{COMMIT_FORMAT}}, exactly as on the CLI:

    <type>(<scope>): <summary>  [{{ISSUE_ID_PATTERN}}]

    <body: what and why>

    {{AGENT_ATTRIBUTION}}

Issue known → reference it; none known → note the absence ({{NO_TICKET_CONVENTION}} where the org has one) and proceed. Never invent an ID.

## Protected refs (implements S2)

- Before `create_or_update_file` / `push_files` / `delete_file`: check the `branch` parameter. Targeting {{PROTECTED_BRANCHES}} → `create_branch` first (named per {{BRANCH_FORMAT}}, cut from the protected ref's head SHA), write there, then `create_pull_request` back to the protected ref. State the branch in one line and proceed.
- Direct writes to protected refs never happen — not even one-file "quick fixes".
- `create_repository`: state name and visibility in one line before creating; a **public** repo is exposure-class → confirm first.

## Merges and reviews (implements S3)

- `merge_pull_request` only when required checks are green and required reviews are in. **Bypassing required checks or branch protection is a never-event** — no admin-merge, no dismissing reviews, no weakening protection rules to get a change through. A red check → fix it or escalate to the check's owner; this rule has no override path.
- PR touches {{SECURITY_REVIEW_PATHS}} → note "requires security review" in the `create_pull_request` body and tag {{SECURITY_REVIEWERS}}.
- Merge method: {{MERGE_METHOD}} unless the repo dictates otherwise.

## Secret scan (implements S4)

Before every `create_or_update_file` / `push_files`, scan the `content` of every file in the call for:

- Private keys (`BEGIN … PRIVATE KEY`), AWS keys (`AKIA…`), tokens (`ghp_`, `xoxb-`, `sk-…`), passwords in URLs or env files
- Internal hostnames, IPs, or customer identifiers in non-sanctioned paths
- Paths that never enter history: `.env*`, credentials files, {{FORBIDDEN_PATHS}}

Any hit: stop and show the user. The API commits instantly — there is no staged-diff second chance, and commit-then-revert still leaves it in history.

## Attribution (implements S5)

Every commit `message` ends with the `{{AGENT_ATTRIBUTION}}` trailer. The commit author is the authenticated human's account — never misstate it, and never present a fully agent-written change as human-authored.

## Escalation

When a rule conflicts with user intent (e.g. "push straight to main, skip the PR"): state the risk in one line ("main is deploy-watched — an unreviewed push hits CI immediately"), offer the compliant path (`create_branch` + PR, ~1 minute), and proceed on explicit override — stating that the override will be recorded. Exception: the S3 check-bypass never-event does not proceed on override.

## Pre-flight checklist (before every GitHub MCP write)

- [ ] `message` matches {{COMMIT_FORMAT}}; issue referenced when known (S1)
- [ ] `branch` is not protected — `create_branch` + `create_pull_request` for protected refs (S2)
- [ ] Required checks green before `merge_pull_request`; security paths flagged in the PR body (S3)
- [ ] Every file's `content` scanned for tokens, keys, and forbidden paths (S4)
- [ ] Attribution trailer present in `message` (S5)

## Category baseline: Source Control

These rules apply to **every** write to a source-control system. Instance skills (git, Bitbucket, …) add tool-specific mechanics.

### Rule S1: Commit messages follow the org nomenclature

Every commit message matches {{COMMIT_FORMAT}} — including a work-item reference ({{ISSUE_ID_PATTERN}}, e.g. `ENG-1234`) when one is known. No issue ID known → note its absence (or use the org's sanctioned no-ticket convention, {{NO_TICKET_CONVENTION}}, where one exists) and proceed. Never invent an issue ID.

### Rule S2: Protected refs are never written directly

No direct commits or pushes to {{PROTECTED_BRANCHES}}. Work happens on a branch named per {{BRANCH_FORMAT}} and lands via PR. Never force-push to a shared branch; never rewrite published history.

### Rule S3: Required reviews precede merge

- Security-sensitive paths ({{SECURITY_REVIEW_PATHS}}) touched → flag for security review in the PR body.
- CODEOWNERS honored; an agent is never the sole approver of its own change.

### Rule S4: Nothing secret enters history

Before every commit: scan the diff for credentials, tokens, private keys, internal hostnames, and customer data. A hit means stop and show the user — never commit and revert, since history retains it.

### Rule S5: Attribution is honest

Agent-authored changes are attributed per the org convention ({{AGENT_ATTRIBUTION}}, e.g. a `Co-Authored-By` trailer). Never impersonate a human as sole author of fully agent-written changes.

### Pre-flight checklist (run before every commit/push/PR)

- [ ] Commit message matches {{COMMIT_FORMAT}}; issue referenced when known (S1)
- [ ] Target branch is not protected; branch name matches {{BRANCH_FORMAT}} (S2)
- [ ] Security-review paths flagged if touched (S3)
- [ ] Diff scanned for secrets (S4)
- [ ] Agent attribution trailer present (S5)

## Defaults

Values in angle brackets are org settings this skill ships without. Until your org fills them in (the `placeholders:` map in this skill's `config.yaml`, or an edit to this file), apply the default below.

- `<issue-id>`: No pattern enforced. Reference an issue ID (e.g. `ENG-1234`) when one exists in session context; never invent one.
- `<no-ticket-convention>`: No convention. With no ticket known, note its absence in the commit body and proceed.
- `<security-reviewers>`: None configured. The "requires security review" note in the PR body is the mechanism; skip the tag and say so.
