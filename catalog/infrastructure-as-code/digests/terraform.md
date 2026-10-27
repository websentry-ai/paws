# terraform digest

terraform / tofu apply, destroy, state rm/mv/push, import, force-unlock, taint/untaint, workspace delete. Same rules for OpenTofu. Resources created are also governed by the matching cloud-resources skill. Apply before retrying.

`<prod-workspaces>` default: workspaces whose name matches `*prod*`.

## 1. Workspace first
`terraform workspace show` before any write; state it and whether it is prod. Prod → no auto-approve, destroys need explicit confirmation. Elsewhere no ceremony.

## 2. Plan → show → apply the same file
```
terraform plan -out=tfplan
terraform show tfplan     # state "N to add, N to change, N to destroy"
terraform apply tfplan    # the SAME file
```
- A shown plan needs no further confirmation.
- `apply` without a plan file, or `-auto-approve`, never runs on prod workspaces. On ephemeral/sandbox/other non-prod workspaces auto-approve is fine: say you're using it.
- Stale plan (code or state changed) → re-plan and re-show. Never apply a plan not shown this session.

## 3. State surgery: backup + narration
Before `state rm`, `state mv`, `import`, `state push`, `force-unlock`:
`terraform state pull > backup-$(date +%Y%m%d-%H%M%S).tfstate`, state where it landed, narrate exactly which addresses move ("moving `aws_instance.web` to `module.web.aws_instance.this`"). `force-unlock` only after confirming no other run is live. Never hand-edit `.tfstate`; never commit it (`.gitignore` covers `*.tfstate*`).

## 4. Modules from the registry, pinned exact
Check `<module-registry>` first; `source = "<module-registry>/vpc/aws"`, `version = "3.2.1"` (never `~>`, `>=`, or unpinned). No registry configured or no covering module → raw resources are fine; say "no registry module covers X, writing raw resources".

## 5. Targeted applies and drift
- `-target` is exceptional: narrate why, `plan -target=<addr> -out=tfplan`, apply it, then run a full `terraform plan` to show what's left.
- Never change IaC-managed resources out-of-band (console, `aws` CLI): refuse and offer edit `.tf` → plan → apply.
- Plan shows changes nobody made in code → report the drift before applying anything.

## 6. Destroys
```
terraform plan -destroy -out=tfplan-destroy
terraform show tfplan-destroy   # list addresses being destroyed
terraform apply tfplan-destroy
```
- Prod workspace: explicit same-session confirmation, restated: "This destroys N resources in <workspace>, including <notable> — confirm?"
- Dev/ephemeral: state count + addresses, then proceed.
- Any routine plan with destroy > 0: call out the destroyed addresses before applying.

## Escalation
"Just run apply -auto-approve": state the risk in one line, offer the plan-file flow, proceed on explicit override off-prod. On prod the saved-plan flow stands.

## Checklist
- [ ] Workspace shown and checked against prod workspaces
- [ ] Plan saved with `-out`, shown, counts stated, same file applied
- [ ] State surgery: `state pull` backup stated; addresses narrated
- [ ] Registry module with exact version, or raw fallback stated
- [ ] `-target` narrated + full plan after; drift reported
- [ ] Destroy addresses listed; prod confirmed
