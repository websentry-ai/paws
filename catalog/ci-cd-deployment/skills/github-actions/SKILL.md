---
name: github-actions
description: 'Mandatory guardrails for every GitHub Actions deploy operation (workflow dispatch, workflow enable/disable, release create/delete, run cancellation, admin merge) via the gh CLI or a GitHub MCP server. Enforces the ci-cd-deployment baseline with GitHub Actions mechanics: predecessor-SHA checks before prod dispatch, one-line rollback statement, no check bypassing, deploy-window notes, and narrated, attributed releases. Triggers PROACTIVELY on any `gh workflow run` or `gh release` write — do NOT wait for the user to invoke it.'
---

# GitHub Actions

GitHub Actions-specific mechanics for the ci-cd-deployment baseline (P1–P5). Apply automatically on every deploy-shaped `gh` call — never skip. The deploy workflow is {{DEPLOY_WORKFLOW_FILE}}; production is the {{PROD_ENVIRONMENT_NAME}} environment.

## Promotion order (implements P1)

Non-production dispatch: state target and SHA in one line ("deploying <sha> to staging") and go:

    gh workflow run {{DEPLOY_WORKFLOW_FILE}} -f environment=staging -f ref=<sha>

Before a {{PROD_ENVIRONMENT_NAME}} dispatch, verify the predecessor in {{ENV_PROMOTION_ORDER}} ran the **same SHA** successfully:

    # Latest runs — check headSha + conclusion for the predecessor environment
    gh run list --workflow={{DEPLOY_WORKFLOW_FILE}} -L5 --json headSha,conclusion,displayTitle

Predecessor's latest successful `headSha` ≠ `<sha>`, no successful run, or a skipped environment → include both SHAs in the deploy message ("deploying <sha> to {{PROD_ENVIRONMENT_NAME}}; staging last ran <other-sha>") and proceed — the audit log records it.

## Rollback plan first (implements P2)

Before any dispatch with `environment={{PROD_ENVIRONMENT_NAME}}`, state the revert path in one line using {{ROLLBACK_MECHANISM}}, naming the exact target:

    # Find the currently deployed SHA (latest successful prod run) — this is the rollback target
    gh run list --workflow={{DEPLOY_WORKFLOW_FILE}} -L5 --json headSha,conclusion,displayTitle

Say it verbatim before dispatching: "Rollback: re-run {{DEPLOY_WORKFLOW_FILE}} with ref=<previous-sha>." No previous successful run, or the change is irreversible (destructive migration) → confirm-class: flag it and proceed only on explicit confirmation.

## Never bypass required checks (implements P3)

FORBIDDEN — never run these, even when asked casually:

    gh pr merge --admin      # bypasses branch protection — never, in any form
    gh run cancel <run-id>   # on a run you did not start — confirm with its owner first

- Never edit files under `.github/workflows/` to remove, weaken, or conditionally skip a required check, and never `gh workflow disable` a required workflow to get a change through.
- A red required check on the deploy ref is a stop: fix it or escalate to the check's owner. There is no override path — required checks are an org security control, and `--admin` stays forbidden no matter who asks.

## Deploy windows (implements P4)

Before any `environment={{PROD_ENVIRONMENT_NAME}}` dispatch:

1. State the current time against {{DEPLOY_WINDOWS}} in one line ("Tue 14:05 PT — inside window"). Outside the window → add the reason to that line, put it in the release notes (P5 command), and proceed.
2. Check {{FREEZE_CALENDAR}}; an active freeze is confirm-class — state the freeze and dispatch only on explicit confirmation.

Non-production environments are not window-restricted.

## Narration and attribution (implements P5)

Before triggering any deploy, state target environment, SHA, and what changed since the currently deployed version — for prod, in the same message as the P1/P2 lines:

    # Diff since the last deployed SHA (from the P1/P2 run list)
    git log <deployed-sha>..<new-sha> --oneline

After a successful {{PROD_ENVIRONMENT_NAME}} deploy, create the release — generated notes plus a one-line summary and attribution:

    gh release create v<version> --target <sha> --generate-notes --notes "<one-line summary of the change> {{AGENT_ATTRIBUTION_CONVENTION}}"

Never run `gh release delete` unprompted — a bad release is superseded by the next one, not erased. If the user insists: state that the tag, notes, and download links are lost, get explicit confirmation, then proceed — the override is recorded in the audit log.

## Escalation

Rule vs user intent: state the risk and the compliant path in one line, then proceed as asked — the override is audit-logged. Only freezes, no-rollback-path deploys, and release deletes still need a confirmed yes; check bypasses have no override at all.

## Pre-flight checklist (before every gated gh call)

- [ ] Prod: predecessor SHA checked; env, SHA, `git log` diff, and rollback line in one message (P1/P2/P5)
- [ ] No `--admin` merge, no canceling others' runs, no disabled/edited required workflows (P3)
- [ ] Time stated against {{DEPLOY_WINDOWS}}; outside-window reason noted; active freeze confirmed (P4)
- [ ] Release carries {{AGENT_ATTRIBUTION_CONVENTION}}; no unconfirmed `gh release delete` (P5)

## Category baseline: CI/CD & Deployment

These rules apply to **every** pipeline- and deploy-triggering tool. An instance skill (GitHub Actions, ArgoCD, …) adds tool-specific mechanics on top; it may tighten these rules but never loosen them. A deploy is the moment code becomes customer reality — but most deploys are routine. Non-prod flows freely with a one-line statement; prod gets one well-narrated message; only a freeze or an irreversible change stops for a confirmed yes. Bypassing checks is the one thing that never happens.

### Rule P1: Promotion order is narrated

Deploys follow {{ENV_PROMOTION_ORDER}}:

1. Non-production deploys flow freely: state target and SHA in one line and dispatch.
2. Before a production deploy, verify the predecessor environment ran the **same SHA** using the instance skill's exact query mechanics.
3. Predecessor on a different SHA, no successful predecessor run, or a skipped environment → say so in the deploy message ("deploying <sha> to prod; staging last ran <other-sha>") and proceed; the audit log records it.

### Rule P2: Rollback plan first

Before any production deploy, state in one line how this deploy is reverted — previous release tag, rollback workflow, or revert PR:

- The line names the exact mechanism and the exact target ("re-run deploy with ref=v1.4.2"), never "we can roll back if needed".
- A deploy with **no rollback path** (no previous successful release, irreversible migration) is confirm-class: flag it before it runs and proceed only on explicit confirmation.

### Rule P3: Never bypass required checks

A red check is a stop, not an obstacle:

- No force-merging red builds. No admin-bypassing branch protection.
- No disabling, skipping, or editing required workflows to get a change through.
- The response to a red check is a fix, or an escalation to the check's owner — never a bypass. This is a hard refusal with no override path: required checks are an org security control.

### Rule P4: Deploy windows respected

1. Production deploys target {{DEPLOY_WINDOWS}}. Outside the window → state the time and the reason in one line, record the reason in the deploy metadata, and proceed.
2. An active {{FREEZE_CALENDAR}} freeze is confirm-class: state the freeze and deploy only on explicit confirmation.
3. Non-production environments are never window-restricted.

### Rule P5: Deploys are narrated and attributed

Nobody should learn about a deploy from an incident:

1. Before triggering, state: target environment, exact SHA, and what changed since the currently deployed version (commit list or one-line summary). For prod, this merges with the P1/P2 lines into one message.
2. Deploy metadata (release notes, run description, deploy message) carries {{AGENT_ATTRIBUTION_CONVENTION}} so operators can distinguish agent-triggered deploys.
3. Never delete releases or deploy history — a bad release is superseded by the next one, not erased; deletion requires explicit confirmation.

### Pre-flight checklist (run before every deploy-shaped call)

- [ ] Prod: predecessor SHA checked; env, SHA, diff, and rollback stated in one message (P1/P2/P5)
- [ ] No bypassed, disabled, or edited-around required checks — no exceptions (P3)
- [ ] Outside-window noted with reason; active freeze confirmed before deploying (P4)
- [ ] Attribution in deploy metadata; releases superseded, never deleted unconfirmed (P5)

## Defaults

Values in angle brackets are org settings this skill ships without. Until your org fills them in (the `placeholders:` map in this skill's `config.yaml`, or an edit to this file), apply the default below.

- `<freeze-calendar>`: No freeze calendar configured, so no freeze check is possible. Skip the check and say so in the message before any prod deploy.
