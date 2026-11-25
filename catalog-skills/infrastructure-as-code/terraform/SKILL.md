---
name: terraform
description: 'Mandatory guardrails for every Terraform write operation (apply, destroy, state rm/mv/push, import, force-unlock, taint) via the terraform or tofu CLI. Enforces the infrastructure-as-code baseline with Terraform mechanics: saved plan files, workspace checks, state backups before surgery, pinned registry modules, and destroy confirmation. Triggers PROACTIVELY on any terraform/tofu write command — do NOT wait for the user to invoke it.'
---

# Terraform

Terraform-specific mechanics for the infrastructure-as-code baseline (F1–F5). Apply automatically on every Terraform write — never skip. Everything below applies identically to the `tofu` CLI (OpenTofu). Resources being provisioned are additionally governed by the matching `cloud-resources` instance skill — both apply.

## Workspace first (implements F1, F5)

Before any write command in a session:

    terraform workspace show

State the workspace and check it against {{PROD_WORKSPACES}}. Prod workspace → F1 and F5 tighten: no auto-approve, destroys need explicit confirmation. Everywhere else the flow below runs without ceremony.

## Plan → show → apply (implements F1)

The canonical flow — the plan you review is the plan you apply:

    terraform plan -out=tfplan
    terraform show tfplan        # state the summary: "N to add, N to change, N to destroy"
    terraform apply tfplan       # the SAME file — never re-plan-and-apply blind

State the add/change/destroy counts before applying — a shown plan needs no further confirmation. `terraform apply` with no plan file, or `terraform apply -auto-approve`, does not run against {{PROD_WORKSPACES}}; on any other workspace ({{EPHEMERAL_WORKSPACES}} included) auto-approve is fine — state that you're using it and go. If the plan is stale (code or state changed since), re-plan and re-review; never apply a plan you haven't shown this session.

## State surgery (implements F2)

Before any `terraform state rm`, `terraform state mv`, or `terraform import`:

    terraform state pull > backup-$(date +%Y%m%d-%H%M%S).tfstate

State where the backup landed, then narrate exactly which addresses move: "moving `aws_instance.web` to `module.web.aws_instance.this`". `terraform state push` and `terraform force-unlock` get the same backup + narration treatment — and force-unlock only after confirming no other run is live. Never edit a `.tfstate` file by hand; never commit one to VCS (check `.gitignore` covers `*.tfstate*`).

## Modules (implements F3)

Check {{MODULE_REGISTRY}} before writing raw resources; source from it with an exact pin:

    module "vpc" {
      source  = "{{MODULE_REGISTRY}}/vpc/aws"
      version = "3.2.1"          # exact — never "~> 3.0", ">= 3.2", or unpinned
    }

No covering module → raw resources are fine, but say so: "no registry module covers X, writing raw resources."

## Targeted applies and drift (implements F4)

`-target` is exceptional. Narrate why the full plan can't run, then follow with a full plan so remaining drift is visible:

    terraform plan -target=<address> -out=tfplan   # narrate the reason first
    terraform apply tfplan
    terraform plan                                  # show what the targeted apply left behind

Never edit IaC-managed resources out-of-band (console, `aws` CLI, etc.) — refuse and offer the code path: edit the `.tf`, plan, apply. When a plan shows changes nobody made in code, report the drift to the user before applying anything; never let an apply silently absorb it.

## Destroy (implements F5)

    terraform plan -destroy -out=tfplan-destroy
    terraform show tfplan-destroy    # list the resource addresses being destroyed
    # prod only: explicit user confirmation — restate: "destroys N resources in <workspace> — confirm?"
    terraform apply tfplan-destroy

Any workspace in {{PROD_WORKSPACES}}: confirmation is mandatory and same-session. Ephemeral/dev workspaces: state the destroy count and addresses, then proceed — no confirmation. Any routine plan with destroy count > 0: call out the destroyed addresses before applying — destroys never ride along unannounced.

## Escalation

Rule vs user intent ("just run apply -auto-approve"): state the risk in one line, offer the plan-file flow (30 seconds more), and proceed on explicit override — recorded.
On {{PROD_WORKSPACES}} the saved-plan flow stands: it is the mechanism, not ceremony.

## Pre-flight checklist (before every Terraform write)

- [ ] `terraform workspace show` run; workspace checked against {{PROD_WORKSPACES}}
- [ ] Plan saved with `-out`, shown, counts stated; applying the same file; auto-approve only off-prod, stated
- [ ] State surgery: `state pull` backup taken and stated; addresses narrated; no hand-edits, no `.tfstate` in VCS
- [ ] Modules sourced from {{MODULE_REGISTRY}} with exact `version`; raw-resource fallback stated
- [ ] `-target` narrated and followed by a full plan; drift reported, not absorbed
- [ ] Destroy: `-destroy` plan shown, addresses listed; prod confirmed explicitly, dev/ephemeral stated and run

## Category baseline: Infrastructure as Code

These rules apply to **every** IaC toolchain. An instance skill (Terraform, Pulumi, …) adds tool-specific mechanics on top; it may tighten these rules but never loosen them. This category governs the IaC toolchain itself — plan, apply, state, modules. The resources it provisions are additionally governed by the `cloud-resources` category's instances, and the two compose: an apply that creates an S3 bucket answers to both skills. State is the org's only shared memory of what exists: an unreviewed apply is a change nobody approved, and one corrupted state file orphans an entire environment.

### Rule F1: Plan before apply, always

Every apply executes a reviewed, saved plan — never an unreviewed live plan:

1. Generate a plan to a file, review it, then apply **that file** — the plan reviewed is the plan executed.
2. Before applying, state the plan summary out loud: resources to add, change, and destroy, by count.
3. Auto-approve does not run on {{PROD_WORKSPACES}} — the saved-plan flow is the mechanism there. On non-prod workspaces auto-approve is fine: state it and go.

### Rule F2: State is sacred

State is the single source of truth mapping code to real infrastructure; corrupt it and the mapping is gone:

- No state surgery (remove, move, import, force-unlock, push) without a **stated backup taken first** and narration of exactly which resource addresses move, from where to where.
- State files are never hand-edited — surgery goes through the toolchain's state commands only.
- State files are never committed to VCS; they contain secrets and their history is not the toolchain's history.

### Rule F3: Modules from the paved road

The org's module registry encodes hard-won defaults — encryption, tagging, networking — that raw resources silently skip:

1. Check {{MODULE_REGISTRY}} for a covering module before writing raw resources.
2. Falling back to raw resources is allowed but **stated**: "no registry module covers X, writing raw resources" — so the gap is visible.
3. Module versions are pinned to an exact version, never floating (no `latest`, no open-ended constraints).

### Rule F4: No manual drift

Infrastructure managed by code changes only through code:

- Targeted/partial applies are exceptional: narrate why the full plan can't run, and follow with a full plan so remaining drift is visible.
- Out-of-band console/CLI edits to IaC-managed resources are **refused** — offer the code path instead (edit the code, plan, apply). The console edit takes minutes; the drift it creates costs the next engineer an afternoon.
- Detected drift (plan shows changes nobody made in code) is reported to the user, never silently absorbed into the next apply.

### Rule F5: Destroys are stated; prod destroys are confirmed

Destruction is the one operation with no undo:

1. Any operation that destroys resources states exactly **what** is destroyed, by resource address and count — destroys never ride along unannounced inside a routine apply.
2. On ephemeral/dev workspaces that statement is enough: state the counts and proceed.
3. Destroy runs against {{PROD_WORKSPACES}} require explicit same-session user confirmation, restated before executing: "This destroys N resources in <workspace>, including <notable> — confirm?"

### Pre-flight checklist (run before every IaC write)

- [ ] Saved plan reviewed; add/change/destroy counts stated; auto-approve only off-prod, stated (F1)
- [ ] State surgery: backup stated first; addresses narrated in one line; no hand-edits, no state in VCS (F2)
- [ ] Registry module used or fallback stated; versions pinned exact (F3)
- [ ] Targeted applies narrated + followed by full plan; drift reported (F4)
- [ ] Destroy counts called out; prod destroys explicitly confirmed (F5)

## Defaults

Values in angle brackets are org settings this skill ships without. Until your org fills them in (the `placeholders:` map in this skill's `config.yaml`, or an edit to this file), apply the default below.

- `<module-registry>`: No registry configured. Raw resources are fine, but say "no registry module covers X, writing raw resources".
- `<prod-workspaces>`: Workspaces whose name matches `*prod*`. Use it and say which workspace you treated as prod.
