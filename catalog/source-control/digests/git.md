# git digest

git write (commit, push, merge, rebase, tag, cherry-pick) or `gh pr create` / `gh pr merge`. Apply before retrying.

## Commit messages
Format `<type>(<scope>): <summary>` (Conventional Commits):

```
<type>(<scope>): <summary>  [<issue-id>]

<body: what and why, wrapped at 72>

Co-Authored-By: <agent name> <noreply@<agent-vendor>>
```

- `type` ∈ feat, fix, chore, docs, refactor, test, perf, ci.
- Reference the issue (e.g. `ENG-1234`) when one is known in session context. None known: note the absence in the body and proceed. Never invent an ID.
- Summary ≤ 72 chars, imperative, no trailing period.

## Branching and pushing
- Check the current branch first. On `main`, `master` or `release/*`: create a branch named `<type>/<slug>` (issue ID when known, e.g. `feat/ENG-1234-short-slug`) before committing.
- No direct commits or pushes to protected branches; changes land via PR.
- `git push --force` / `--force-with-lease` only on your own unshared branch, and say so. Never force-push a shared branch.
- Never `git rebase` or `commit --amend` commits that exist on the remote.

## Secret scan before every commit
Review `git diff --staged` for:
- Private keys (`BEGIN … PRIVATE KEY`), AWS keys (`AKIA…`), tokens (`ghp_`, `xoxb-`, `sk-…`), passwords in URLs or env files.
- Internal hostnames, IPs, customer identifiers in non-sanctioned paths.
- Paths that must not be committed: `.env*`, `**/credentials*`, `*.pem`, `*.key`, `**/*_rsa*`.

Any hit: stop, show the finding, let the user decide. Never commit-then-revert; history keeps it.

## Merges, tags, reviews
- Merging into a protected branch goes through a PR, never local `git merge` + push.
- Touching `auth/**`, `**/crypto/**`, `**/secrets/**`, `**/iam/**`: note "requires security review" in the PR body (no reviewers configured: the note is the mechanism; say so).
- CODEOWNERS honored; the agent is never the sole approver of its own change.
- Release tags `v<major>.<minor>.<patch>` (semver), created only when the user asks.

## Attribution
- Every agent-authored commit carries `Co-Authored-By: <agent name> <noreply@<agent-vendor>>`.
- Author stays the human's git identity; never change `user.name` / `user.email`. Never present fully agent-written changes as human-only.

## Checklist
- [ ] Message format, issue referenced when known, ≤72-char imperative summary
- [ ] Not on a protected branch; branch name conforms
- [ ] Staged diff scanned for secrets and forbidden paths
- [ ] No history rewrite on pushed commits; no force-push to shared branches
- [ ] Security paths flagged in the PR body
- [ ] Attribution trailer present
