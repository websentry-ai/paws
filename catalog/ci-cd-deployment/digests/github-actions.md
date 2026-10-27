# github-actions digest

GitHub Actions deploy operations: `gh workflow run/enable/disable`, `gh release create/delete`, `gh run cancel`, `gh pr merge --admin`, or the GitHub MCP equivalents. Deploy workflow: `deploy.yml`; production environment: `production`. Apply before retrying.

## P1: promotion order (dev → staging → prod)
- Non-prod: one line ("deploying <sha> to staging") and dispatch: `gh workflow run deploy.yml -f environment=staging -f ref=<sha>`.
- Before a `production` dispatch, check the predecessor ran the same SHA: `gh run list --workflow=deploy.yml -L5 --json headSha,conclusion,displayTitle`.
- Different SHA, no successful run, or a skipped env → put both SHAs in the deploy message ("deploying <sha> to production; staging last ran <other-sha>") and proceed.

## P2: rollback line before every prod dispatch
- Find the currently deployed SHA (latest successful prod run, same `gh run list`). Say: "Rollback: re-run deploy.yml with ref=<previous-sha>." Never "we can roll back if needed".
- No previous successful run, or irreversible change (destructive migration) → flag it; proceed only on explicit confirmation.

## P3: never bypass required checks (no override)
- Never `gh pr merge --admin`, in any form, no matter who asks.
- Never `gh run cancel` a run you did not start without its owner's OK.
- Never edit `.github/workflows/` to remove, weaken or skip a required check; never `gh workflow disable` a required workflow to get a change through.
- Red required check on the deploy ref = stop: fix it or escalate to the check's owner.

## P4: deploy windows
- Prod: state the current time against business hours Tue–Thu (owner's timezone) in one line ("Tue 14:05 PT — inside window"). Outside → add the reason to that line and to the release notes, then proceed.
- Active freeze → state it, dispatch only on explicit confirmation. No freeze calendar configured (`<freeze-calendar>`) → say no freeze check was possible.
- Non-prod is never window-restricted.

## P5: narrate and attribute
- Before any deploy: target env, SHA, and what changed since the deployed version: `git log <deployed-sha>..<new-sha> --oneline`. For prod, one message with the P1/P2/P4 lines.
- After a successful prod deploy: `gh release create v<version> --target <sha> --generate-notes --notes "<one-line summary> [deployed-by: ai-agent]"`.
- Never `gh release delete` unprompted: a bad release is superseded, not erased. If the user insists, state that tag, notes and download links are lost and get explicit confirmation.

## Escalation
State the risk and compliant path in one line, then proceed as asked (audit-logged). Freezes, no-rollback deploys and release deletes need a confirmed yes; check bypasses have no override.

## Checklist
- [ ] Prod: predecessor SHA checked; env, SHA, diff, rollback in one message
- [ ] No `--admin`, no cancelling others' runs, no disabled/edited required workflows
- [ ] Time vs window stated; outside-window reason noted; freeze confirmed
- [ ] Release notes carry `[deployed-by: ai-agent]`; no unconfirmed release delete
