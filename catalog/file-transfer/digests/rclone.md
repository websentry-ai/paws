# rclone digest

rclone copy / sync / move / delete / purge / config. Apply before retrying the command.

Defaults until the org configures them: sanctioned remotes = remotes already configured on this machine (new remotes: ask); external = any remote the org does not own (confirm before pushing org data); sensitive = credentials, customer data, financials, unreleased plans.

## 1. Destination check first
`rclone listremotes`:
- listed and sanctioned → proceed;
- not listed → needs a new remote (see 6);
- listed but not org-owned (personal Drive, private S3) → cross-boundary: stop and get explicit same-session confirmation.

## 2. Direction sentence before every copy / sync / move
Push or pull, source → destination, inside or outside the org, count. E.g. "Pushing ~/exports/q2/ → s3-acme:finance-archive/q2 (org-owned) — 214 files." `rclone move` deletes at the source: say so.

## 3. Delete-propagating syncs
1. `rclone sync --dry-run <src> <dst>`, state copy and delete counts.
2. `rclone check <src> <dst>` to verify divergence.
3. State "will copy N, delete M at dst" and run. No confirmation on top; pause only if deletes are wildly out of line with intent.
- `rclone purge` and `rclone delete` on a directory: `rclone ls <remote:path>` first, state the file count, get explicit confirmation (irreversible).

## 4. Classify before moving
`rclone lsl <src> | head -50`: look for dumps, exports, `.env`, key material, customer-named files. PII, secrets or customer data never go to an unsanctioned remote. Unclear tier → ask before transferring.

## 5. Size and cost
`rclone size <src>`. Above 10 GB: one line with total size, object count, egress implication (cloud → local or cross-cloud is billed), then run constrained, e.g. `--max-transfer 10G --bwlimit 8M`. Reruns skip matching files: resume, don't restart blind.

## 6. Creating remotes
Only for sanctioned destinations. Narrate provider, account/tenant and scope (bucket/path, read or write), then `rclone config create <name> <provider> <key=value ...>`. Credentials live only in rclone's config store, password-encrypted whenever a remote holds long-lived credentials. Never inline credential flags (`--s3-secret-access-key`, any `--<provider>-key`) or pasted env one-liners.

## Escalation
User wants org data on a personal remote: state the risk in one line, name the compliant path, proceed only on explicit override. External destinations keep their confirmation; everything else is narrate-and-go.

## Baseline (all bulk transfers)
- U1 direction + endpoints restated; cross-boundary confirmed; moves named as moves.
- U2 delete-propagating syncs dry-run first; stated counts are the gate.
- U3 source spot-checked against sensitivity tiers; unclear → ask first.
- U4 destinations from sanctioned remotes; new remotes narrated; credentials only in the config store.
- U5 above threshold: size + egress cost in one line; resumable, rate-limited runs.

## Checklist
- [ ] `rclone listremotes` run; external confirmed
- [ ] Direction sentence; move vs copy named
- [ ] Sync dry-run + check, counts stated; purge/dir delete listed + confirmed
- [ ] Source spot-checked
- [ ] Size + egress above 10 GB; no inline credential flags
