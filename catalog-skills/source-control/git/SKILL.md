---
name: git
description: 'Mandatory guardrails for git write operations (commit, push, branch, merge, tag) run via the shell. Enforces the source-control baseline with git mechanics: commit-message nomenclature with issue IDs, protected-branch rules, pre-commit secret scan, and honest agent attribution. Triggers PROACTIVELY on any git write command — do NOT wait for the user to invoke it.'
---

# Git

Git-specific mechanics for the source-control baseline (S1–S5). Apply automatically on every git write — never skip.

## Commit messages (implements S1)

Format: {{COMMIT_FORMAT}}

Default org convention:

```
<type>(<scope>): <summary>  [{{ISSUE_ID_PATTERN}}]

<body: what and why, wrapped at 72>

{{AGENT_ATTRIBUTION}}
```

- `type` ∈ {{COMMIT_TYPES}}
- One issue reference matching {{ISSUE_ID_PATTERN}} when an issue is known; none known → note the absence ({{NO_TICKET_CONVENTION}} where the org has one) and proceed. Never invent an ID.
- Summary ≤ 72 chars, imperative mood, no trailing period.

## Branching and pushing (implements S2)

- Before committing, check the current branch: on {{PROTECTED_BRANCHES}} → create a branch first, named {{BRANCH_FORMAT}}, e.g. `feat/ENG-1234-short-slug`.
- `git push --force` / `--force-with-lease` only on your own unshared branch, and say so.
- Never `git rebase` or `commit --amend` on commits that exist on the remote.

## Pre-commit secret scan (implements S4)

Before every `git commit`, review `git diff --staged` for:

- Private keys (`BEGIN … PRIVATE KEY`), AWS keys (`AKIA…`), tokens (`ghp_`, `xoxb-`, `sk-…`), passwords in URLs or env files
- Internal hostnames, IPs, or customer identifiers in non-sanctioned paths
- Files that should be ignored: `.env*`, credentials files, {{FORBIDDEN_PATHS}}

Any hit: stop, show the finding, and let the user decide. Do not commit-then-revert — history keeps it.

## Merges, tags, history (implements S2, S3)

- Merging to {{PROTECTED_BRANCHES}} happens via PR, never `git merge` locally + push.
- Touching {{SECURITY_REVIEW_PATHS}} → note "requires security review" in the PR body and tag {{SECURITY_REVIEWERS}}.
- Release tags follow {{TAG_FORMAT}} and are created only when the user asks.

## Attribution (implements S5)

Every agent-authored commit carries the trailer `{{AGENT_ATTRIBUTION}}`. Author remains the human's git identity; never change `user.name`/`user.email`.

## Pre-flight checklist (before every git write)

- [ ] Message matches format, issue referenced when known, ≤72-char imperative summary
- [ ] Not committing to {{PROTECTED_BRANCHES}}; branch name conforms
- [ ] Staged diff scanned for secrets and forbidden paths
- [ ] No history rewrite on pushed commits
- [ ] Attribution trailer present

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
