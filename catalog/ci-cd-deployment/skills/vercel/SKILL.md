---
name: vercel
description: 'Mandatory guardrails for every Vercel MCP deploy or purchase operation (deploy_to_vercel, buy_domain, buy_credits, buy_pro, buy_addon). Enforces the ci-cd-deployment baseline with Vercel mechanics: free-flowing previews, narrated production deploys, promote-previous rollback named up front, no redeploying past failed builds, deploy-window checks, and attributed deploy metadata — plus a hard price-first confirmation on every real-money buy_* tool. Triggers PROACTIVELY on any deploy_to_vercel or buy_* call — do NOT wait for the user to invoke it.'
---

# Vercel

Vercel-specific mechanics for the ci-cd-deployment baseline (P1–P5). Apply automatically on every deploy-shaped or spend-shaped Vercel MCP call — never skip. Vercel realizes {{ENV_PROMOTION_ORDER}} as **preview → production**: every change gets a preview deployment; production is the promotion. Production scope: {{PROD_PROJECTS}}.

## Promotion order (implements P1)

- Preview deploys (`deploy_to_vercel` not targeting production) flow freely — one line ("Preview deploy of <project> from <commit>") and go.
- Production deploys state what changed since the live deployment first:

      list_deployments   # current production deployment: READY, target=production
      get_deployment     # its commit SHA = the live baseline

  Narrate the diff between that SHA and what you are deploying (commit list or one-line summary). No preview of this change exists → deploy one first, or skip only on an explicit confirmed override, restated before executing.

## Rollback plan first (implements P2)

Rollback mechanism: {{ROLLBACK_MECHANISM}}. Before every production deploy, name the exact target in one line:

    list_deployments   # previous READY production deployment — note its id/URL

> "Rollback: promote deployment <dpl_id / url> (currently live)."

No previous READY production deployment (first deploy, or all prior failed) → flag it and proceed only on explicit confirmation.

## Failed builds are stops (implements P3)

A failed build is a red check — never redeploy hoping it passes, never ship around it:

    get_deployment_build_logs   # on any ERROR deployment — diagnose before anything else
    get_runtime_errors          # after deploying — verify the deployment is actually healthy

Fix the build error or escalate to the code's owner, then deploy. Never push a known-broken commit to production because "the preview failure looks unrelated". User insists → restate what the failure means in one line and proceed only on explicit confirmation; the override is recorded.

## Deploy windows (implements P4)

Production deploys happen inside {{DEPLOY_WINDOWS}} and never during {{FREEZE_CALENDAR}} freezes. Before each one, state the time against the window in one line ("Tue 14:05 PT — inside window"). Outside window or during a freeze → confirmed override with a stated reason, recorded in the deploy note. Previews are never window-restricted.

## Narration and attribution (implements P5)

Before any production `deploy_to_vercel`: state project, commit, and the P1 diff. Deploy metadata (deploy message, or a toolbar comment where the team uses the Vercel toolbar) carries {{AGENT_ATTRIBUTION_CONVENTION}} so operators can tell agent-triggered deploys apart. Never delete deployments or deploy history — a bad deploy is superseded by the next promote, not erased.

## Purchases are never autonomous (spend never-event)

`buy_domain`, `buy_credits`, `buy_pro`, and `buy_addon` spend real money. **NEVER call a buy_* tool without explicit user confirmation of that specific purchase — and state the price first**:

    check_domain_availability_and_price   # before buy_domain
    get_purchase_quote                     # before buy_credits / buy_pro / buy_addon

> "example.com is available at $59.99/yr — confirm purchase?"

No standing instruction ("grab whatever domain works") substitutes for a per-purchase confirmation carrying the actual price. This rule has no override path.

## Escalation

When a rule conflicts with user intent (e.g. "skip the preview, ship it"): state the risk in one line ("production would run a commit no preview has ever built"), offer the compliant path (preview first, ~2 minutes), and proceed on explicit override — stating that the override will be recorded. Purchases are the exception: price + explicit per-purchase confirmation, always.

## Pre-flight checklist (before every gated Vercel call)

- [ ] Preview exists for this change or the skip is explicitly confirmed; prod diff vs live narrated (P1, P5)
- [ ] Rollback named: exact previous READY production deployment id/URL (P2)
- [ ] No failed build redeployed or ignored; build logs checked on any failure (P3)
- [ ] Time stated against the deploy window; no active freeze (P4)
- [ ] buy_*: price stated + explicit per-purchase confirmation — never autonomous (spend)

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
