# vercel digest

Vercel MCP deploys (`deploy_to_vercel`) and purchases (`buy_domain`, `buy_credits`, `buy_pro`, `buy_addon`). Promotion order dev → staging → prod is preview → production on Vercel; every project counts as production when deploying to production. Apply before retrying.

## Purchases: never autonomous (no override)
- Never call a `buy_*` tool without explicit confirmation of that specific purchase, with the price stated first.
- Price first: `check_domain_availability_and_price` before `buy_domain`; `get_purchase_quote` before `buy_credits` / `buy_pro` / `buy_addon`.
- Ask: "example.com is available at $59.99/yr — confirm purchase?" No standing instruction ("grab whatever domain works") replaces a per-purchase confirmation with the actual price.

## P1: previews flow, production is narrated
- Preview deploy: one line ("Preview deploy of <project> from <commit>") and go.
- Production: `list_deployments` (current READY production) + `get_deployment` (its commit = live baseline); narrate the diff to what you deploy.
- No preview of this change → deploy one first; skip only on an explicit confirmed override, restated.

## P2: rollback named first
- Mechanism: promote the previous READY production deployment (instant, no rebuild).
- Before every prod deploy: `list_deployments`, then say "Rollback: promote deployment <dpl_id / url> (currently live)."
- No previous READY production deployment → flag it; proceed only on explicit confirmation.

## P3: failed builds are stops
- On any ERROR deployment: `get_deployment_build_logs` and diagnose before anything else. After deploying: `get_runtime_errors` to verify health.
- Never redeploy hoping it passes, never ship a known-broken commit because the failure "looks unrelated". Fix or escalate to the code owner. User insists → restate what the failure means, proceed only on explicit confirmation.

## P4: deploy windows
- Prod deploys inside business hours Tue–Thu (owner's timezone), never during a freeze. State the time against the window in one line.
- Outside the window or during a freeze → confirmed override with a stated reason, recorded in the deploy note. No freeze calendar configured (`<freeze-calendar>`) → say no freeze check was possible.
- Previews are never window-restricted.

## P5: narrate and attribute
- Before any prod `deploy_to_vercel`: project, commit, and the P1 diff.
- Deploy metadata (deploy message, or toolbar comment) carries `[deployed-by: ai-agent]`.
- Never delete deployments or history: a bad deploy is superseded by the next promote.

## Baseline (all deploys)
- Never bypass, disable or edit around required checks: no override path.
- Rule conflicts with user intent: state the risk in one line, offer the compliant path (preview first), proceed on explicit override and say it is recorded.

## Checklist
- [ ] Preview exists or skip confirmed; prod diff vs live narrated
- [ ] Rollback named: previous READY production deployment id/URL
- [ ] No failed build redeployed or ignored; logs checked
- [ ] Time stated against window; no active freeze
- [ ] `buy_*`: price stated + explicit per-purchase confirmation
