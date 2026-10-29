# github digest

GitHub MCP write: `create_or_update_file`, `push_files`, `create_branch`, `create_pull_request`, `merge_pull_request`, `delete_file`, `create_repository`. Reads (`get_*`, `list_*`, `search_*`) are free. Apply before retrying.

## Commit messages
The `message` of `create_or_update_file`, `push_files` and `delete_file` is a commit message in `<type>(<scope>): <summary>` (Conventional Commits) format:

```
<type>(<scope>): <summary>  [<issue-id>]

<body: what and why>

Co-Authored-By: <agent name> <noreply@<agent-vendor>>
```

Issue known (e.g. `ENG-1234`): reference it. None known: note the absence in the body and proceed. Never invent an ID.

## Protected refs
- Check `branch` before every file write. Targeting `main`, `master` or `release/*`: `create_branch` first (named `<type>/<slug>`, issue ID when known, cut from the protected ref's head SHA), write there, then `create_pull_request` back. State the branch in one line.
- No direct writes to protected refs, not even one-file quick fixes.
- `create_repository`: state name and visibility first; a **public** repo is exposure: confirm first.

## Merges and reviews
- `merge_pull_request` only with required checks green and required reviews in.
- Never bypass required checks or branch protection: no admin-merge, no dismissing reviews, no weakening protection. Red check: fix it or escalate to its owner. No override, even if the user asks.
- PR touches `auth/**`, `**/crypto/**`, `**/secrets/**`, `**/iam/**`: note "requires security review" in the PR body (no reviewers configured: the note is the mechanism; say so).
- CODEOWNERS honored; the agent never solely approves its own change.
- Merge method: squash unless the repo dictates otherwise.

## Secret scan before every file write
Scan the `content` of every file in the call for:
- Private keys (`BEGIN … PRIVATE KEY`), AWS keys (`AKIA…`), tokens (`ghp_`, `xoxb-`, `sk-…`), passwords in URLs or env files.
- Internal hostnames, IPs, customer identifiers in non-sanctioned paths.
- Paths that never enter history: `.env*`, `**/credentials*`, `*.pem`, `*.key`, `**/*_rsa*`.

Any hit: stop and show the user. The API commits instantly; there is no staged-diff second chance, and a revert still leaves it in history.

## Attribution
Every commit `message` ends with `Co-Authored-By: <agent name> <noreply@<agent-vendor>>`. The author is the authenticated human's account; never misstate it or present agent-written work as human-only.

## Escalation
"Push straight to main, skip the PR": state the risk ("an unreviewed push hits CI immediately"), offer `create_branch` + PR, proceed only on explicit override and say it is recorded. The check-bypass rule never proceeds on override.

## Checklist
- [ ] `message` format; issue referenced when known
- [ ] `branch` not protected, else branch + PR
- [ ] Checks green and reviews in before merge; security paths flagged
- [ ] Every file's `content` scanned for tokens, keys, forbidden paths
- [ ] Attribution trailer in `message`; public repo confirmed
